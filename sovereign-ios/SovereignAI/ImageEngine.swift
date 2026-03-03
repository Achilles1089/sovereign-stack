import Foundation
import UIKit
import CoreML

class ImageEngine: ObservableObject {
    
    private var textEncoder: MLModel?
    private var unet: MLModel?
    private var decoder: MLModel?
    private var vocab: [String: Int] = [:]
    private var merges: [(String, String)] = []
    
    @Published var isLoaded = false
    @Published var modelName = ""
    @Published var isGenerating = false
    @Published var lastGenTime: Double = 0
    
    private let maxTokenLen = 77
    private let latentSize = 64
    private let latentChannels = 4
    
    // MARK: - Model Discovery
    
    func findModelDir() -> URL? {
        let docsDir = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask).first!
        guard let contents = try? FileManager.default.contentsOfDirectory(at: docsDir, includingPropertiesForKeys: [.isDirectoryKey]) else { return nil }
        
        // Check if Unet.mlmodelc is directly in Documents (flat transfer)
        if contents.contains(where: { $0.lastPathComponent == "Unet.mlmodelc" }) {
            return docsDir
        }
        
        // Check subdirectories
        for dir in contents {
            let isDir = (try? dir.resourceValues(forKeys: [.isDirectoryKey]))?.isDirectory ?? false
            if isDir {
                let sub = (try? FileManager.default.contentsOfDirectory(at: dir, includingPropertiesForKeys: nil)) ?? []
                if sub.contains(where: { $0.lastPathComponent == "Unet.mlmodelc" }) { return dir }
            }
        }
        return nil
    }
    
    // MARK: - Load
    
    func load(modelDir: URL) throws {
        // Tokenizer
        let vocabURL = modelDir.appendingPathComponent("vocab.json")
        let mergesURL = modelDir.appendingPathComponent("merges.txt")
        if let d = try? Data(contentsOf: vocabURL),
           let v = try? JSONSerialization.jsonObject(with: d) as? [String: Int] { vocab = v }
        if let t = try? String(contentsOf: mergesURL, encoding: .utf8) {
            merges = t.components(separatedBy: "\n").dropFirst().compactMap {
                let p = $0.components(separatedBy: " ")
                return p.count == 2 ? (p[0], p[1]) : nil
            }
        }
        
        let config = MLModelConfiguration()
        config.computeUnits = .cpuAndNeuralEngine
        
        textEncoder = try MLModel(contentsOf: modelDir.appendingPathComponent("TextEncoder.mlmodelc"), configuration: config)
        decoder = try MLModel(contentsOf: modelDir.appendingPathComponent("VAEDecoder.mlmodelc"), configuration: config)
        unet = try MLModel(contentsOf: modelDir.appendingPathComponent("Unet.mlmodelc"), configuration: config)
        
        modelName = modelDir.lastPathComponent
        isLoaded = true
    }
    
    func unload() {
        textEncoder = nil; unet = nil; decoder = nil
        vocab = [:]; merges = []
        isLoaded = false; modelName = ""
    }
    
    // MARK: - Generate
    
    func generate(prompt: String) throws -> Data {
        guard let te = textEncoder, let un = unet, let dec = decoder else {
            throw ImageError.notLoaded
        }
        isGenerating = true
        let start = CFAbsoluteTimeGetCurrent()
        defer {
            isGenerating = false
            DispatchQueue.main.async { self.lastGenTime = CFAbsoluteTimeGetCurrent() - start }
        }
        
        let steps = 15
        let guidance: Float = 7.5
        let seed = UInt32.random(in: 0...UInt32.max)
        
        let tokens = tokenize(prompt)
        let uncondTokens = tokenize("")
        
        let promptEmb = try encodeTokens(tokens, model: te)
        let uncondEmb = try encodeTokens(uncondTokens, model: te)
        
        var latents = makeNoise(seed: seed)
        
        // Linear beta schedule
        let betaStart: Float = 0.00085
        let betaEnd: Float = 0.012
        var alphasCumprod = [Float](repeating: 0, count: 1001)
        alphasCumprod[0] = 1.0
        for i in 0..<1000 {
            let beta = betaStart + Float(i) * (betaEnd - betaStart) / 999.0
            alphasCumprod[i + 1] = alphasCumprod[i] * (1.0 - beta)
        }
        
        let stepSize = 1000 / steps
        let timesteps = stride(from: 999, through: 0, by: -stepSize).map { $0 }
        
        for (i, t) in timesteps.enumerated() {
            let aT = alphasCumprod[t + 1]
            let aPrev: Float = (i + 1 < timesteps.count) ? alphasCumprod[timesteps[i + 1] + 1] : 1.0
            
            let uncondPred = try runUnet(latents, t: t, emb: uncondEmb, model: un)
            let condPred = try runUnet(latents, t: t, emb: promptEmb, model: un)
            
            let n = latents.count
            var eps = [Float](repeating: 0, count: n)
            for j in 0..<n { eps[j] = uncondPred[j] + guidance * (condPred[j] - uncondPred[j]) }
            
            let sqA = sqrt(aT)
            let sq1A = sqrt(1.0 - aT)
            let sqAP = sqrt(aPrev)
            let sq1AP = sqrt(1.0 - aPrev)
            
            for j in 0..<n {
                let x0 = (latents[j] - sq1A * eps[j]) / max(sqA, 1e-8)
                let clamped = max(-1, min(1, x0))
                latents[j] = sqAP * clamped + sq1AP * eps[j]
            }
        }
        
        for j in 0..<latents.count { latents[j] /= 0.18215 }
        return try decodeLatents(latents, model: dec)
    }
    
    // MARK: - Tokenizer
    
    private func tokenize(_ text: String) -> [Int] {
        let startToken = 49406
        let endToken = 49407
        
        if text.isEmpty {
            var result = [startToken]
            result.append(endToken)
            while result.count < maxTokenLen { result.append(0) }
            return result
        }
        
        // Simple whitespace tokenization then BPE
        let words = text.lowercased().components(separatedBy: .whitespaces).filter { !$0.isEmpty }
        var allTokens = [startToken]
        
        for word in words {
            // Look up whole word first (with end-of-word marker)
            let wordWithEnd = word + "</w>"
            if let id = vocab[wordWithEnd] {
                allTokens.append(id)
            } else {
                // Character-level fallback
                for char in word {
                    let charStr = String(char) + "</w>"
                    if let id = vocab[charStr] {
                        allTokens.append(id)
                    } else if let id = vocab[String(char)] {
                        allTokens.append(id)
                    }
                }
            }
            if allTokens.count >= maxTokenLen - 1 { break }
        }
        
        allTokens.append(endToken)
        while allTokens.count < maxTokenLen { allTokens.append(0) }
        return Array(allTokens.prefix(maxTokenLen))
    }
    
    // MARK: - Text Encoder
    
    private func encodeTokens(_ tokens: [Int], model: MLModel) throws -> MLMultiArray {
        let inputArray = try MLMultiArray(shape: [1, NSNumber(value: maxTokenLen)], dataType: .int32)
        for (i, t) in tokens.enumerated() {
            inputArray[i] = NSNumber(value: Int32(t))
        }
        
        let inputName = model.modelDescription.inputDescriptionsByName.keys.first ?? "input_ids"
        let provider = try MLDictionaryFeatureProvider(dictionary: [inputName: inputArray])
        let output = try model.prediction(from: provider)
        
        // Get the first output (hidden states)
        let outputName = model.modelDescription.outputDescriptionsByName.keys.first ?? "last_hidden_state"
        guard let result = output.featureValue(for: outputName)?.multiArrayValue else {
            throw ImageError.encodingFailed
        }
        return result
    }
    
    // MARK: - UNet
    
    private func runUnet(_ latents: [Float], t: Int, emb: MLMultiArray, model: MLModel) throws -> [Float] {
        let shape = [1, latentChannels, latentSize, latentSize] as [NSNumber]
        let latentArray = try MLMultiArray(shape: shape, dataType: .float32)
        for (i, v) in latents.enumerated() {
            latentArray[i] = NSNumber(value: v)
        }
        
        let tArray = try MLMultiArray(shape: [1] as [NSNumber], dataType: .float32)
        tArray[0] = NSNumber(value: Float(t))
        
        // Build input dict based on model's expected inputs
        var inputDict: [String: Any] = [:]
        let inputNames = Array(model.modelDescription.inputDescriptionsByName.keys)
        
        for name in inputNames {
            let lower = name.lowercased()
            if lower.contains("sample") || lower.contains("latent") || lower.contains("x") {
                inputDict[name] = latentArray
            } else if lower.contains("timestep") || lower.contains("t") || lower.contains("time") {
                inputDict[name] = tArray
            } else if lower.contains("encoder") || lower.contains("hidden") || lower.contains("text") || lower.contains("context") {
                inputDict[name] = emb
            }
        }
        
        // Fallback if we couldn't match names
        if inputDict.count < 3 && inputNames.count >= 3 {
            inputDict = [
                inputNames[0]: latentArray,
                inputNames[1]: tArray,
                inputNames[2]: emb
            ]
        }
        
        let provider = try MLDictionaryFeatureProvider(dictionary: inputDict)
        let output = try model.prediction(from: provider)
        
        let outputName = model.modelDescription.outputDescriptionsByName.keys.first ?? "noise_pred"
        guard let result = output.featureValue(for: outputName)?.multiArrayValue else {
            throw ImageError.generationFailed
        }
        
        var noise = [Float](repeating: 0, count: latents.count)
        for i in 0..<noise.count {
            noise[i] = result[i].floatValue
        }
        return noise
    }
    
    // MARK: - VAE Decoder
    
    private func decodeLatents(_ latents: [Float], model: MLModel) throws -> Data {
        let shape = [1, latentChannels, latentSize, latentSize] as [NSNumber]
        let latentArray = try MLMultiArray(shape: shape, dataType: .float32)
        for (i, v) in latents.enumerated() {
            latentArray[i] = NSNumber(value: v)
        }
        
        let inputName = model.modelDescription.inputDescriptionsByName.keys.first ?? "z"
        let provider = try MLDictionaryFeatureProvider(dictionary: [inputName: latentArray])
        let output = try model.prediction(from: provider)
        
        let outputName = model.modelDescription.outputDescriptionsByName.keys.first ?? "image"
        guard let result = output.featureValue(for: outputName)?.multiArrayValue else {
            throw ImageError.generationFailed
        }
        
        // Convert MLMultiArray [1, 3, 512, 512] -> UIImage -> JPEG
        let width = 512
        let height = 512
        var pixels = [UInt8](repeating: 255, count: width * height * 4)
        
        for y in 0..<height {
            for x in 0..<width {
                let rIdx = 0 * height * width + y * width + x
                let gIdx = 1 * height * width + y * width + x
                let bIdx = 2 * height * width + y * width + x
                let pixelIdx = (y * width + x) * 4
                
                let r = (result[rIdx].floatValue + 1.0) * 127.5
                let g = (result[gIdx].floatValue + 1.0) * 127.5
                let b = (result[bIdx].floatValue + 1.0) * 127.5
                
                pixels[pixelIdx] = UInt8(max(0, min(255, r)))
                pixels[pixelIdx + 1] = UInt8(max(0, min(255, g)))
                pixels[pixelIdx + 2] = UInt8(max(0, min(255, b)))
                pixels[pixelIdx + 3] = 255
            }
        }
        
        let colorSpace = CGColorSpaceCreateDeviceRGB()
        guard let ctx = CGContext(data: &pixels, width: width, height: height,
                                  bitsPerComponent: 8, bytesPerRow: width * 4,
                                  space: colorSpace,
                                  bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue),
              let cgImage = ctx.makeImage() else {
            throw ImageError.encodingFailed
        }
        
        let uiImage = UIImage(cgImage: cgImage)
        guard let jpegData = uiImage.jpegData(compressionQuality: 0.85) else {
            throw ImageError.encodingFailed
        }
        return jpegData
    }
    
    // MARK: - Noise
    
    private func makeNoise(seed: UInt32) -> [Float] {
        let count = latentChannels * latentSize * latentSize
        var noise = [Float](repeating: 0, count: count)
        srand48(Int(seed))
        for i in 0..<count {
            // Box-Muller transform for Gaussian noise
            let u1 = max(1e-10, Float(drand48()))
            let u2 = Float(drand48())
            noise[i] = sqrt(-2.0 * log(u1)) * cos(2.0 * .pi * u2)
        }
        return noise
    }
    
    // MARK: - Errors
    
    enum ImageError: Error, LocalizedError {
        case notLoaded, generationFailed, encodingFailed, modelNotFound
        var errorDescription: String? {
            switch self {
            case .notLoaded: return "No image model loaded"
            case .generationFailed: return "Image generation failed"
            case .encodingFailed: return "Failed to encode output"
            case .modelNotFound: return "Model directory missing required files"
            }
        }
    }
}
