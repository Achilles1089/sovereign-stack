# Sovereign Stack — Hardware Inventory & Architecture

> **Last updated:** 2026-03-04 (v4 FINAL)

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│              SOVEREIGN STACK — 4 NODE FINAL              │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  💻 Chromebook (N4500)               [REMOTE / CLIENT]  │
│     ├─ Browser mic → audio stream to Brain Net           │
│     └─ Browser speaker ← audio stream from Brain Net     │
│                                                          │
│  🔧 Brain Net (Celeron N4100)        [OPS + VOICE]      │
│     ├─ Go Backend + Dashboard (port 8080)                │
│     ├─ Whisper Tiny STT (75MB)                           │
│     ├─ Piper TTS (50MB)                                  │
│     └─ API Router / Orchestrator                         │
│                                                          │
│  🖼️🎵 Envy (i5-6200U, AVX2)          [MEDIA]           │
│     ├─ Image Gen (SDXS-512, OpenVINO, port 8090)         │
│     └─ Music Gen (Riffusion via sd binary)               │
│                                                          │
│  🧠 Phone (T616)                     [LLM]              │
│     ├─ RWKV-7 0.4B Q8  (30+ tok/s, instant)             │
│     ├─ RWKV-7 1.5B Q4  (8-15 tok/s, download pending)   │
│     └─ RWKV-7 2.9B Q4  (3 tok/s, quality mode)          │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

## Network

```
Chromebook ──WiFi──► Brain Net (Orchestrator + Voice)
                      ├── Ethernet 1Gbps ──► Envy (Media)
                      └── USB/ADB ──► Phone (LLM)
```

## Voice Flow

```
Chromebook mic → WiFi → Brain Net (Whisper STT) → text
  → USB → Phone (LLM) → response text
  → Brain Net (Piper TTS) → audio → WiFi → Chromebook speaker
```

## Model Catalog

| # | Model | Node | Size | Speed |
|---|-------|------|------|-------|
| 1 | RWKV-7 0.4B Q8 | Phone | 478MB | 30+ tok/s |
| 2 | RWKV-7 1.5B Q4 | Phone | ~900MB | 8-15 tok/s |
| 3 | RWKV-7 2.9B Q4 | Phone | 1.7GB | ~3 tok/s |
| 4 | SDXS-512 (OpenVINO) | Envy | ~2GB | ~2-3s/img |
| 5 | Riffusion | Envy | ~2GB | ~2-3s/spec |
| 6 | Whisper Tiny | Brain Net | 75MB | Real-time |
| 7 | Piper TTS | Brain Net | 50MB | Real-time |

## TODO
- [ ] Download RWKV-7 1.5B Q4 to phone
- [ ] WireGuard mesh (encrypted P2P)
- [ ] Brain Net as WiFi AP (hostapd, fully offline)
