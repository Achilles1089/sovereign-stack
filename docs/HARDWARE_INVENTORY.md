# Sovereign Stack — Hardware Inventory & Architecture

> **Last updated:** 2026-03-04

## Final Architecture

```
┌─────────────────────────────────────────────────────┐
│            SOVEREIGN STACK v2 — FINAL               │
├─────────────────────────────────────────────────────┤
│                                                     │
│  🖼️🎵 Envy (i5-6200U, 8GB)         [MEDIA NODE]   │
│     ├─ Image Gen (SDXS-512 OpenVINO)               │
│     └─ Music Gen (MusicGen Small)                   │
│                                                     │
│  🧠 Brain Net (Celeron N4100, 8GB)  [BRAIN NODE]   │
│     ├─ RWKV 2.9B Q4 (LLM)                          │
│     ├─ Go Backend + Dashboard                       │
│     └─ API Router / Orchestrator                    │
│                                                     │
│  🗣️ Phone (Android/T616)           [VOICE NODE]    │
│     └─ Piper TTS (Text-to-Speech)                   │
│                                                     │
│  💻 Mac — Browser only (not part of stack)          │
│                                                     │
└─────────────────────────────────────────────────────┘
```

## Network

```
Brain Net ◄══ Ethernet 1Gbps ══► Envy
  10.0.0.1                        10.0.0.2
     │ USB
  Phone (ADB)

Mac connects via WiFi (Achilles New) to Brain Net dashboard
```

---

## Node 1: Brain Net Core (Mini PC) — BRAIN NODE

| Spec | Value |
|------|-------|
| **Hostname** | `brainnet` |
| **CPU** | Intel Celeron N4100 @ 1.10GHz (4c/4t, burst 2.4GHz) |
| **RAM** | 7.6 GB |
| **Disk** | 57 GB (LVM) |
| **GPU** | Intel UHD 600 (GeminiLake) |
| **OS** | Ubuntu 24.04.4 LTS |
| **Role** | LLM inference, orchestration, dashboard |

## Node 2: HP Envy Laptop — MEDIA NODE

| Spec | Value |
|------|-------|
| **Hostname** | `envy` |
| **CPU** | Intel Core i5-6200U @ 2.30GHz (2c/4t, AVX2) |
| **RAM** | 7.7 GB |
| **Disk** | 98 GB (93% free) |
| **GPU** | Intel HD Graphics 520 (Skylake GT2) |
| **OS** | Ubuntu 24.04.2 LTS |
| **Role** | Image generation, music generation |

## Node 3: Phone — VOICE NODE

| Spec | Value |
|------|-------|
| **Model** | TBD (Android, Unisoc T616) |
| **Connection** | USB to Brain Net |
| **Role** | Text-to-Speech output (Piper TTS) |

---

## TODO (Future)
- [ ] WireGuard mesh network — eliminate router dependency, encrypted P2P
