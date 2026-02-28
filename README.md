<div align="center">

# ⚡ Sovereign Stack

**Own your infrastructure. Run your AI. Control your data.**

The self-hosted platform that turns any computer into a sovereign server — with one command.

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-AGPL_3.0-blue?style=flat)](LICENSE)
[![Apps](https://img.shields.io/badge/Apps-30+-green?style=flat)](#app-marketplace)
[![CI](https://github.com/Achilles1089/sovereign-stack/actions/workflows/ci.yml/badge.svg)](https://github.com/Achilles1089/sovereign-stack/actions)

</div>

---

## What is this?

Sovereign Stack is a single Go binary that:

- 🔍 **Detects your hardware** — CPU, RAM, disk, GPU (NVIDIA, AMD, Apple Silicon, Intel ARC)
- 🐳 **Deploys Docker services** — Caddy, PostgreSQL, Ollama, MinIO
- 🤖 **Runs AI locally** — 12+model catalog, auto-selects the best model for your GPU
- 📦 **Installs 30+ apps** — Nextcloud, Jellyfin, Home Assistant, Grafana, and more
- 🔒 **Encrypts backups** — Restic-based, cron-scheduled, encrypted at rest
- 🌐 **Mesh networking** — Connect servers via WireGuard with a join token
- 🎨 **Web dashboard** — Dark glassmorphism UI with real-time status

## Quick Start

```bash
# Install (macOS / Linux)
curl -fsSL https://raw.githubusercontent.com/Achilles1089/sovereign-stack/main/scripts/install.sh | bash

# Or build from source
git clone https://github.com/Achilles1089/sovereign-stack.git
cd sovereign-stack
go build -o sovereign .

# Initialize your server
sovereign init

# Launch the dashboard
sovereign dashboard
```

## CLI Commands

| Command | Description |
|---|---|
| `sovereign init` | Setup wizard — detect hardware, install Docker, configure AI |
| `sovereign status` | Health check for all services |
| `sovereign app list` | Browse 30+ self-hosted apps |
| `sovereign app install <name>` | Install an app (e.g., `nextcloud`, `grafana`) |
| `sovereign ai chat` | Chat with your local AI model |
| `sovereign ai catalog` | Browse AI models for your hardware tier |
| `sovereign backup` | Create an encrypted backup |
| `sovereign backup schedule` | Set up automated daily backups |
| `sovereign mesh create` | Create a WireGuard mesh network |
| `sovereign mesh join <token>` | Join an existing mesh |
| `sovereign dashboard` | Launch the web dashboard |
| `sovereign logs <service>` | Stream service logs |
| `sovereign update` | Pull latest images and restart |

## App Marketplace

30 apps across 13 categories:

| Category | Apps |
|---|---|
| **Productivity** | Nextcloud, Syncthing, Stirling PDF, Paperless-ngx, BookStack, Wiki.js |
| **Media** | Jellyfin, Immich, PhotoPrism, Audiobookshelf, Kavita, Navidrome |
| **Monitoring** | Uptime Kuma, Grafana, Prometheus, Changedetection, Speedtest Tracker |
| **Development** | Gitea, IT-Tools |
| **AI** | Open WebUI |
| **Security** | Vaultwarden |
| **System** | Portainer, MinIO, Homarr |
| **Automation** | n8n |
| **Network** | AdGuard Home |
| **Smart Home** | Home Assistant |
| **Analytics** | Plausible |
| **Lifestyle** | Mealie |
| **Finance** | Firefly III |

## AI Inference

Sovereign Stack auto-detects your GPU and recommends the optimal model:

| GPU Tier | VRAM | Recommended Model |
|---|---|---|
| CPU-only | — | Qwen 2.5 0.5B |
| Basic (4-8 GB) | 4-8 GB | Qwen 2.5 3B |
| Mid (8-16 GB) | 8-16 GB | Qwen 2.5 7B |
| High (16-24 GB) | 16-24 GB | Qwen 2.5 14B |
| Ultra (24+ GB) | 24+ GB | Qwen 2.5 32B |
| Apex (64+ GB) | 64+ GB | Qwen 2.5 32B |

Supports: NVIDIA (CUDA), AMD (ROCm), Apple Silicon (Metal), Intel ARC (SYCL).

## Architecture

```
sovereign (10 MB binary)
├── cmd/           CLI commands (Cobra)
├── internal/
│   ├── ai/        Ollama client + model catalog + server chat
│   ├── apps/      30-app marketplace + compose merging
│   ├── audit/     JSONL audit log with rotation
│   ├── backup/    Restic wrapper + cron scheduler
│   ├── cloud/     Sovereign Cloud client (optional)
│   ├── config/    YAML config system
│   ├── docker/    Compose generator + health checks
│   ├── hardware/  CPU/RAM/GPU detection (5 GPU types)
│   ├── mesh/      WireGuard mesh networking
│   ├── platform/  OS routing (Linux/macOS/WSL2)
│   ├── rbac/      Role-based access control
│   ├── server/    REST API + SPA handler
│   └── sso/       Authentik SSO integration
└── dashboard/     React + TypeScript (Vite)
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

**Adding an app?** Use the [app manifest template](.github/ISSUE_TEMPLATE/app_manifest.md) or submit a PR modifying `internal/apps/installer.go`.

## Sovereign Box

Want to run this on dedicated hardware? See [docs/sovereign-box.md](docs/sovereign-box.md) for the hardware BOM and assembly guide.

## License

[AGPL-3.0](LICENSE) — Your infrastructure. Your rules.

---

<div align="center">

**Sovereignty over convenience.**

Built by [Achilles1089](https://github.com/Achilles1089)

</div>
