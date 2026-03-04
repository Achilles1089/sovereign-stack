# Sovereign Stack — Hardware Inventory & Architecture

> **Last updated:** 2026-03-04

## Final Architecture (v3)

```
┌────────────────────────────────────────────────────────┐
│            SOVEREIGN STACK — 4 NODE FINAL              │
├────────────────────────────────────────────────────────┤
│                                                        │
│  💻 Chromebook (N4500)                [REMOTE/CLIENT] │
│     └─ Browser → Dashboard                            │
│        Closed WiFi network, no internet needed         │
│                                                        │
│  🖼️ Brain Net (Celeron N4100)        [MEDIA + OPS]   │
│     ├─ Image Gen (SDXS-512, OpenVINO)                 │
│     ├─ Go Backend + Dashboard                          │
│     └─ API Router / Orchestrator                       │
│                                                        │
│  🧠🎵 Envy (i5-6200U, AVX2)          [BRAIN + MUSIC] │
│     ├─ LLM (RWKV-7 2.9B via llama-server)            │
│     └─ Music Gen (Riffusion via sd binary)             │
│                                                        │
│  🗣️ Phone (T616)                     [VOICE]         │
│     ├─ Whisper Tiny (STT)                              │
│     └─ Piper TTS (Text-to-Speech)                      │
│                                                        │
└────────────────────────────────────────────────────────┘
```

## Network (Closed — No Internet)

```
Chromebook ──WiFi──► Brain Net (Orchestrator)
                      ├── Ethernet 1Gbps ──► Envy (LLM + Music)
                      └── USB ──► Phone (Voice)
```

Brain Net hosts WiFi AP (hostapd) or all join local router with WAN unplugged.

---

## Node 1: Brain Net (Mini PC) — MEDIA + OPS

| Spec | Value |
|------|-------|
| **CPU** | Intel Celeron N4100 @ 1.10GHz (4c/4t) |
| **RAM** | 7.6 GB |
| **GPU** | Intel UHD 600 (12 EUs, OpenVINO) |
| **Role** | Image gen, orchestration, dashboard |

## Node 2: HP Envy — BRAIN + MUSIC

| Spec | Value |
|------|-------|
| **CPU** | Intel Core i5-6200U @ 2.30GHz (2c/4t, **AVX2**) |
| **RAM** | 7.7 GB |
| **GPU** | Intel HD 520 (24 EUs) |
| **Role** | LLM inference (RWKV 2.9B), music gen (Riffusion) |

## Node 3: Phone — VOICE

| Spec | Value |
|------|-------|
| **SoC** | Unisoc T616 (Cortex-A75 + A55) |
| **RAM** | 6 GB |
| **Role** | Whisper STT + Piper TTS |

## Node 4: Samsung Chromebook XE345XDA — REMOTE

| Spec | Value |
|------|-------|
| **CPU** | Intel Celeron N4500 (2c/2t) |
| **RAM** | 4 GB |
| **Role** | Browser client only — views dashboard on closed network |

## TODO (Future)
- [ ] WireGuard mesh — encrypted P2P, no router needed
- [ ] Brain Net as WiFi AP (hostapd) — fully self-contained network
