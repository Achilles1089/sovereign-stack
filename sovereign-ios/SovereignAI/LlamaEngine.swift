import Foundation
import llama

/// Wraps llama.cpp for on-device inference with Metal GPU acceleration
class LlamaEngine: ObservableObject {
    
    private var model: OpaquePointer? // llama_model *
    private var ctx: OpaquePointer?   // llama_context *
    private var sampler: UnsafeMutablePointer<llama_sampler>?
    private var vocab: OpaquePointer?   // llama_vocab *
    
    @Published var isLoaded = false
    @Published var modelName = ""
    @Published var lastTokPerSec: Double = 0
    
    private let contextSize: UInt32 = 2048
    
    deinit {
        unload()
    }
    
    /// Load a GGUF model from the given file path
    func load(path: String) throws {
        unload()
        
        // Initialize backend (Metal enabled by default)
        llama_backend_init()
        
        // Model params
        var modelParams = llama_model_default_params()
        modelParams.n_gpu_layers = 99 // Offload all layers to Metal GPU
        
        guard let m = llama_model_load_from_file(path, modelParams) else {
            throw LlamaError.modelLoadFailed
        }
        model = m
        
        // Get vocab from model
        vocab = llama_model_get_vocab(m)
        
        // Context params
        var ctxParams = llama_context_default_params()
        ctxParams.n_ctx = contextSize
        ctxParams.n_batch = 512
        ctxParams.n_threads = Int32(max(1, ProcessInfo.processInfo.activeProcessorCount - 1))
        
        guard let c = llama_init_from_model(m, ctxParams) else {
            llama_model_free(m)
            model = nil
            throw LlamaError.contextInitFailed
        }
        ctx = c
        
        // Sampler chain: temp -> top-p -> dist
        let sparams = llama_sampler_chain_default_params()
        guard let chain = llama_sampler_chain_init(sparams) else {
            llama_free(c)
            llama_model_free(m)
            ctx = nil
            model = nil
            throw LlamaError.samplerInitFailed
        }
        llama_sampler_chain_add(chain, llama_sampler_init_temp(0.7))
        llama_sampler_chain_add(chain, llama_sampler_init_top_p(0.9, 1))
        llama_sampler_chain_add(chain, llama_sampler_init_dist(UInt32.random(in: 0...UInt32.max)))
        sampler = chain
        
        modelName = URL(fileURLWithPath: path).lastPathComponent
        isLoaded = true
    }
    
    /// Unload the current model and free memory
    func unload() {
        if let s = sampler { llama_sampler_free(s); sampler = nil }
        if let c = ctx { llama_free(c); ctx = nil }
        if let m = model { llama_model_free(m); model = nil }
        vocab = nil
        isLoaded = false
        modelName = ""
        llama_backend_free()
    }
    
    /// Run chat completion with streaming callback
    func complete(
        messages: [[String: String]],
        maxTokens: Int = 512,
        onToken: @escaping (String) -> Void,
        isCancelled: @escaping () -> Bool
    ) throws -> String {
        guard let model = model, let ctx = ctx, let sampler = sampler, let vocab = vocab else {
            throw LlamaError.notLoaded
        }
        
        // Format messages into a single prompt
        let prompt = formatChatPrompt(messages)
        
        // Tokenize
        let promptCStr = prompt.cString(using: .utf8)!
        let nPromptTokensMax = Int32(prompt.count + 256)
        var tokens = [llama_token](repeating: 0, count: Int(nPromptTokensMax))
        let nTokens = llama_tokenize(vocab, promptCStr, Int32(promptCStr.count - 1), &tokens, nPromptTokensMax, true, true)
        
        guard nTokens > 0 else {
            throw LlamaError.tokenizationFailed
        }
        tokens = Array(tokens.prefix(Int(nTokens)))
        
        // Create batch and process prompt
        var batch = llama_batch_init(Int32(tokens.count), 0, 1)
        for (i, token) in tokens.enumerated() {
            batch.n_tokens = Int32(i + 1)
            batch.token[i] = token
            batch.pos[i] = Int32(i)
            batch.n_seq_id[i] = 1
            batch.seq_id[i]![0] = 0
            batch.logits[i] = (i == tokens.count - 1) ? 1 : 0
        }
        
        if llama_decode(ctx, batch) != 0 {
            llama_batch_free(batch)
            throw LlamaError.decodeFailed
        }
        llama_batch_free(batch)
        
        // Generate tokens
        var generated = ""
        var nGenerated = 0
        let startTime = CFAbsoluteTimeGetCurrent()
        
        while nGenerated < maxTokens && !isCancelled() {
            let newToken = llama_sampler_sample(sampler, ctx, -1)
            
            // Check for EOS
            if llama_vocab_is_eog(vocab, newToken) {
                break
            }
            
            // Decode token to string
            var buf = [CChar](repeating: 0, count: 256)
            let nChars = llama_token_to_piece(vocab, newToken, &buf, 256, 0, true)
            if nChars > 0 {
                let piece = String(cString: buf)
                generated += piece
                onToken(piece)
            }
            
            // Prepare next batch (single token)
            var nextBatch = llama_batch_init(1, 0, 1)
            nextBatch.n_tokens = 1
            nextBatch.token[0] = newToken
            nextBatch.pos[0] = Int32(tokens.count) + Int32(nGenerated)
            nextBatch.n_seq_id[0] = 1
            nextBatch.seq_id[0]![0] = 0
            nextBatch.logits[0] = 1
            
            if llama_decode(ctx, nextBatch) != 0 {
                llama_batch_free(nextBatch)
                break
            }
            llama_batch_free(nextBatch)
            
            nGenerated += 1
            llama_sampler_accept(sampler, newToken)
        }
        
        let elapsed = CFAbsoluteTimeGetCurrent() - startTime
        if elapsed > 0 && nGenerated > 0 {
            lastTokPerSec = Double(nGenerated) / elapsed
        }
        
        return generated
    }
    
    /// Format messages into ChatML-style prompt
    private func formatChatPrompt(_ messages: [[String: String]]) -> String {
        var prompt = ""
        for msg in messages {
            let role = msg["role"] ?? "user"
            let content = msg["content"] ?? ""
            prompt += "<|im_start|>\(role)\n\(content)<|im_end|>\n"
        }
        prompt += "<|im_start|>assistant\n"
        return prompt
    }
    
    enum LlamaError: Error, LocalizedError {
        case modelLoadFailed
        case contextInitFailed
        case samplerInitFailed
        case notLoaded
        case tokenizationFailed
        case decodeFailed
        
        var errorDescription: String? {
            switch self {
            case .modelLoadFailed: return "Failed to load GGUF model"
            case .contextInitFailed: return "Failed to init context"
            case .samplerInitFailed: return "Failed to init sampler"
            case .notLoaded: return "No model loaded"
            case .tokenizationFailed: return "Tokenization failed"
            case .decodeFailed: return "Decode failed"
            }
        }
    }
}
