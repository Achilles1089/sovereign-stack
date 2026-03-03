import Foundation
import Vapor
import Combine

/// Manages the Vapor HTTP server and LLM engine
class ServerManager: ObservableObject {
    
    @Published var isRunning = false
    @Published var address = "—"
    @Published var port: Int = 8081
    @Published var activeModel = "none"
    @Published var lastTokPerSec: Double = 0
    @Published var logs: [String] = []
    
    let engine = LlamaEngine()
    private var app: Application?
    
    func start() {
        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            guard let self = self else { return }
            do {
                let app = try Application(.detect())
                self.app = app
                
                // Configure server
                app.http.server.configuration.hostname = "0.0.0.0"
                app.http.server.configuration.port = self.port
                
                // Register routes
                self.configureRoutes(app)
                
                // Get WiFi IP
                let ip = self.getWiFiAddress() ?? "localhost"
                DispatchQueue.main.async {
                    self.address = ip
                    self.isRunning = true
                    self.log("Server started on \(ip):\(self.port)")
                }
                
                // Auto-load model if available
                self.autoLoadModel()
                
                try app.run()
            } catch {
                DispatchQueue.main.async {
                    self.log("Server error: \(error)")
                    self.isRunning = false
                }
            }
        }
    }
    
    func stop() {
        app?.shutdown()
        isRunning = false
        log("Server stopped")
    }
    
    func log(_ message: String) {
        let timestamp = DateFormatter.localizedString(from: Date(), dateStyle: .none, timeStyle: .medium)
        let entry = "[\(timestamp)] \(message)"
        DispatchQueue.main.async {
            self.logs.append(entry)
            if self.logs.count > 100 { self.logs.removeFirst() }
        }
    }
    
    // MARK: - Routes
    
    private func configureRoutes(_ app: Application) {
        
        // Health check
        app.get { [weak self] req -> String in
            self?.log("GET /")
            return "SovereignAI inference node"
        }
        
        // List models (OpenAI-compatible)
        app.get("v1", "models") { [weak self] req -> Response in
            self?.log("GET /v1/models")
            let models: [[String: Any]] = [{
                let m = self?.engine.modelName ?? "none"
                return [
                    "id": m,
                    "object": "model",
                    "owned_by": "sovereign"
                ]
            }()]
            let body: [String: Any] = ["object": "list", "data": models]
            let json = try JSONSerialization.data(withJSONObject: body)
            return Response(status: .ok, headers: ["Content-Type": "application/json"], body: .init(data: json))
        }
        
        // Status endpoint
        app.get("status") { [weak self] req -> Response in
            self?.log("GET /status")
            let status: [String: Any] = [
                "status": self?.engine.isLoaded == true ? "ready" : "no_model",
                "model": self?.engine.modelName ?? "none",
                "tok_per_sec": self?.engine.lastTokPerSec ?? 0,
                "device": "iPhone",
                "chip": "A13 Bionic",
                "ram_total_mb": ProcessInfo.processInfo.physicalMemory / 1_048_576,
                "battery_pct": self?.getBatteryLevel() ?? -1,
                "gpu": "Metal"
            ]
            let json = try JSONSerialization.data(withJSONObject: status)
            return Response(status: .ok, headers: ["Content-Type": "application/json"], body: .init(data: json))
        }
        
        // Chat completions (OpenAI-compatible, streaming)
        app.on(.POST, "v1", "chat", "completions") { [weak self] req -> Response in
            guard let self = self else {
                return Response(status: .internalServerError)
            }
            self.log("POST /v1/chat/completions")
            
            // Parse request
            struct ChatRequest: Content {
                let model: String?
                let messages: [[String: String]]
                let max_tokens: Int?
                let stream: Bool?
                let temperature: Double?
            }
            
            let chatReq: ChatRequest
            do {
                chatReq = try req.content.decode(ChatRequest.self)
            } catch {
                self.log("Bad request: \(error)")
                return Response(status: .badRequest, body: .init(string: "{\"error\": \"Invalid request\"}"))
            }
            
            guard self.engine.isLoaded else {
                return Response(status: .serviceUnavailable, body: .init(string: "{\"error\": \"No model loaded\"}"))
            }
            
            let shouldStream = chatReq.stream ?? false
            let maxTok = chatReq.max_tokens ?? 512
            
            if shouldStream {
                // SSE streaming response
                let headers = HTTPHeaders([
                    ("Content-Type", "text/event-stream"),
                    ("Cache-Control", "no-cache"),
                    ("Connection", "keep-alive"),
                    ("Access-Control-Allow-Origin", "*"),
                ])
                
                let response = Response(status: .ok, headers: headers)
                let id = UUID().uuidString
                
                response.body = .init(asyncStream: { writer in
                    do {
                        _ = try self.engine.complete(
                            messages: chatReq.messages,
                            maxTokens: maxTok,
                            onToken: { token in
                                let chunk: [String: Any] = [
                                    "id": "chatcmpl-\(id)",
                                    "object": "chat.completion.chunk",
                                    "created": Int(Date().timeIntervalSince1970),
                                    "choices": [[
                                        "index": 0,
                                        "delta": ["content": token],
                                        "finish_reason": NSNull()
                                    ]]
                                ]
                                if let jsonData = try? JSONSerialization.data(withJSONObject: chunk),
                                   let jsonStr = String(data: jsonData, encoding: .utf8) {
                                    _ = try? writer.write(.buffer(.init(string: "data: \(jsonStr)\n\n")))
                                }
                            },
                            isCancelled: { false }
                        )
                        
                        // Send done
                        let doneChunk: [String: Any] = [
                            "id": "chatcmpl-\(id)",
                            "object": "chat.completion.chunk",
                            "created": Int(Date().timeIntervalSince1970),
                            "choices": [[
                                "index": 0,
                                "delta": [:] as [String: String],
                                "finish_reason": "stop"
                            ]]
                        ]
                        if let jsonData = try? JSONSerialization.data(withJSONObject: doneChunk),
                           let jsonStr = String(data: jsonData, encoding: .utf8) {
                            _ = try? writer.write(.buffer(.init(string: "data: \(jsonStr)\n\n")))
                        }
                        _ = try? writer.write(.buffer(.init(string: "data: [DONE]\n\n")))
                        try writer.write(.end)
                        
                        self.log("Streamed \(String(format: "%.1f", self.engine.lastTokPerSec)) tok/s")
                        DispatchQueue.main.async {
                            self.lastTokPerSec = self.engine.lastTokPerSec
                            self.activeModel = self.engine.modelName
                        }
                    } catch {
                        self.log("Stream error: \(error)")
                        try? writer.write(.end)
                    }
                })
                
                return response
            } else {
                // Non-streaming response
                do {
                    var fullText = ""
                    fullText = try self.engine.complete(
                        messages: chatReq.messages,
                        maxTokens: maxTok,
                        onToken: { _ in },
                        isCancelled: { false }
                    )
                    
                    let result: [String: Any] = [
                        "id": "chatcmpl-\(UUID().uuidString)",
                        "object": "chat.completion",
                        "created": Int(Date().timeIntervalSince1970),
                        "choices": [[
                            "index": 0,
                            "message": ["role": "assistant", "content": fullText],
                            "finish_reason": "stop"
                        ]],
                        "usage": [
                            "prompt_tokens": 0,
                            "completion_tokens": 0,
                            "total_tokens": 0
                        ]
                    ]
                    
                    let json = try JSONSerialization.data(withJSONObject: result)
                    self.log("Completed \(String(format: "%.1f", self.engine.lastTokPerSec)) tok/s")
                    return Response(status: .ok, headers: ["Content-Type": "application/json", "Access-Control-Allow-Origin": "*"], body: .init(data: json))
                } catch {
                    self.log("Completion error: \(error)")
                    return Response(status: .internalServerError, body: .init(string: "{\"error\": \"\(error)\"}"))
                }
            }
        }
        
        // CORS preflight
        app.on(.OPTIONS, "v1", "chat", "completions") { req -> Response in
            return Response(status: .ok, headers: [
                "Access-Control-Allow-Origin": "*",
                "Access-Control-Allow-Methods": "POST, OPTIONS",
                "Access-Control-Allow-Headers": "Content-Type, Authorization",
            ])
        }
    }
    
    // MARK: - Model Management
    
    /// Auto-load first available model from Documents
    private func autoLoadModel() {
        let docsDir = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask).first!
        do {
            let files = try FileManager.default.contentsOfDirectory(at: docsDir, includingPropertiesForKeys: nil)
            if let gguf = files.first(where: { $0.pathExtension == "gguf" }) {
                try engine.load(path: gguf.path)
                DispatchQueue.main.async {
                    self.activeModel = self.engine.modelName
                    self.log("Loaded: \(self.engine.modelName)")
                }
            } else {
                log("No GGUF models in Documents. Use iTunes File Sharing or download in-app.")
            }
        } catch {
            log("Model scan error: \(error)")
        }
    }
    
    // MARK: - Helpers
    
    func getWiFiAddress() -> String? {
        var address: String?
        var ifaddr: UnsafeMutablePointer<ifaddrs>?
        guard getifaddrs(&ifaddr) == 0, let firstAddr = ifaddr else { return nil }
        
        for ptr in sequence(first: firstAddr, next: { $0.pointee.ifa_next }) {
            let interface = ptr.pointee
            let addrFamily = interface.ifa_addr.pointee.sa_family
            
            if addrFamily == UInt8(AF_INET) {
                let name = String(cString: interface.ifa_name)
                if name == "en0" { // WiFi interface
                    var hostname = [CChar](repeating: 0, count: Int(NI_MAXHOST))
                    getnameinfo(interface.ifa_addr, socklen_t(interface.ifa_addr.pointee.sa_len),
                               &hostname, socklen_t(hostname.count), nil, socklen_t(0), NI_NUMERICHOST)
                    address = String(cString: hostname)
                }
            }
        }
        freeifaddrs(ifaddr)
        return address
    }
    
    func getBatteryLevel() -> Int {
        UIDevice.current.isBatteryMonitoringEnabled = true
        let level = UIDevice.current.batteryLevel
        return level < 0 ? -1 : Int(level * 100)
    }
}
