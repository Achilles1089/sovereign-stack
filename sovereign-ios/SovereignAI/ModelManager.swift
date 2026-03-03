import Foundation

/// Manages GGUF model files in the app's Documents directory
class ModelManager: ObservableObject {
    
    @Published var availableModels: [ModelInfo] = []
    @Published var isDownloading = false
    @Published var downloadProgress: Double = 0
    
    struct ModelInfo: Identifiable {
        let id = UUID()
        let name: String
        let path: String
        let sizeBytes: Int64
        
        var sizeFormatted: String {
            let gb = Double(sizeBytes) / 1_073_741_824
            if gb >= 1.0 {
                return String(format: "%.1f GB", gb)
            }
            let mb = Double(sizeBytes) / 1_048_576
            return String(format: "%.0f MB", mb)
        }
    }
    
    /// Recommended models for iPhone 11 Pro Max (4GB RAM)
    /// Only 1.5B and smaller — 3B models won't fit in 4GB
    static let catalog: [(name: String, url: String, sizeLabel: String)] = [
        (
            "Qwen2.5-0.5B-Instruct-Q4_K_M.gguf",
            "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q4_k_m.gguf",
            "~400 MB • ~30+ tok/s"
        ),
        (
            "Qwen2.5-1.5B-Instruct-Q4_K_M.gguf",
            "https://huggingface.co/Qwen/Qwen2.5-1.5B-Instruct-GGUF/resolve/main/qwen2.5-1.5b-instruct-q4_k_m.gguf",
            "~1.0 GB • ~10-14 tok/s"
        ),
        (
            "SmolLM2-360M-Instruct-Q8_0.gguf",
            "https://huggingface.co/HuggingFaceTB/SmolLM2-360M-Instruct-GGUF/resolve/main/smollm2-360m-instruct-q8_0.gguf",
            "~380 MB • ~40+ tok/s"
        ),
    ]
    
    private var documentsDir: URL {
        FileManager.default.urls(for: .documentDirectory, in: .userDomainMask).first!
    }
    
    /// Scan Documents directory for .gguf files
    func scanModels() {
        do {
            let files = try FileManager.default.contentsOfDirectory(at: documentsDir, includingPropertiesForKeys: [.fileSizeKey])
            availableModels = files
                .filter { $0.pathExtension == "gguf" }
                .compactMap { url -> ModelInfo? in
                    let attrs = try? FileManager.default.attributesOfItem(atPath: url.path)
                    let size = attrs?[.size] as? Int64 ?? 0
                    return ModelInfo(name: url.lastPathComponent, path: url.path, sizeBytes: size)
                }
                .sorted { $0.name < $1.name }
        } catch {
            print("Model scan error: \(error)")
        }
    }
    
    /// Download a model from URL to Documents
    func download(name: String, from urlString: String, completion: @escaping (Result<URL, Error>) -> Void) {
        guard let url = URL(string: urlString) else {
            completion(.failure(NSError(domain: "ModelManager", code: 1, userInfo: [NSLocalizedDescriptionKey: "Invalid URL"])))
            return
        }
        
        isDownloading = true
        downloadProgress = 0
        
        let destURL = documentsDir.appendingPathComponent(name)
        
        // If already exists, skip
        if FileManager.default.fileExists(atPath: destURL.path) {
            isDownloading = false
            scanModels()
            completion(.success(destURL))
            return
        }
        
        let session = URLSession(configuration: .default, delegate: nil, delegateQueue: nil)
        let task = session.downloadTask(with: url) { [weak self] tmpURL, response, error in
            DispatchQueue.main.async {
                self?.isDownloading = false
            }
            
            if let error = error {
                completion(.failure(error))
                return
            }
            
            guard let tmpURL = tmpURL else {
                completion(.failure(NSError(domain: "ModelManager", code: 2, userInfo: [NSLocalizedDescriptionKey: "No download data"])))
                return
            }
            
            do {
                try FileManager.default.moveItem(at: tmpURL, to: destURL)
                DispatchQueue.main.async {
                    self?.scanModels()
                }
                completion(.success(destURL))
            } catch {
                completion(.failure(error))
            }
        }
        
        // Observe progress
        let observation = task.progress.observe(\.fractionCompleted) { [weak self] progress, _ in
            DispatchQueue.main.async {
                self?.downloadProgress = progress.fractionCompleted
            }
        }
        _ = observation // Keep alive
        
        task.resume()
    }
}
