# SovereignAI — iPhone Inference Node

Private, local AI inference server for iPhone. Turns your phone into an OpenAI-compatible API endpoint on your local network.

## Architecture
- **llama.cpp** with Metal GPU acceleration for fast inference
- **Vapor** HTTP server exposing `/v1/chat/completions` (OpenAI-compatible)
- **SwiftUI** dashboard showing server status, IP, tok/s
- Models stored in Documents (transfer via Finder File Sharing)

## Supported iPhone
- **iPhone 11 Pro Max** (A13 Bionic, 4GB RAM)
- Metal GPU: 4-core
- Max model: ~1.5B Q4 (Qwen 2.5 1.5B recommended)
- Expected: ~10-14 tok/s

## Setup

### 1. Open in Xcode
```bash
cd sovereign-ios
open SovereignAI.xcodeproj
```

### 2. Add Swift Packages (in Xcode)
- File → Add Package Dependencies:
  - `https://github.com/StanfordBDHG/llama.cpp-spm` (llama.cpp for iOS)
  - `https://github.com/vapor/vapor.git` (version 4.89+)

### 3. Transfer a Model
- Build & run on your iPhone
- In Finder: iPhone → Files → SovereignAI → drag in a .gguf model
- Or download in-app from the model catalog

### 4. Use from Dashboard
```bash
# Check status
curl http://IPHONE-IP:8081/status

# Chat
curl http://IPHONE-IP:8081/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"hello"}],"stream":true}'
```

## Recommended Models (4GB RAM safe)
| Model | Size | Speed |
|---|---|---|
| Qwen 2.5 0.5B Q4_K_M | ~400 MB | ~30+ tok/s |
| Qwen 2.5 1.5B Q4_K_M | ~1.0 GB | ~10-14 tok/s |
| SmolLM2 360M Q8_0 | ~380 MB | ~40+ tok/s |

## API Endpoints
| Method | Path | Description |
|---|---|---|
| POST | `/v1/chat/completions` | Chat with streaming SSE |
| GET | `/v1/models` | List loaded models |
| GET | `/status` | Device info + inference stats |
