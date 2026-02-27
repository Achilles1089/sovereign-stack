# Sovereign Stack

**Own your cloud. Own your AI. One command.**

Sovereign Stack turns any computer — an old PC, a mini server, or a VPS — into your personal cloud with built-in AI inference. Run `sovereign init` and get a fully configured server with local AI, database, automatic SSL, encrypted backups, and a web dashboard. Install apps like Nextcloud, Jellyfin, and Immich with one click.

## Quick Start

```bash
# Install
curl -fsSL https://get.sovereign.dev | sh

# Set up your server
sudo sovereign init

# Chat with your local AI
sovereign ai chat

# Install an app
sovereign app install nextcloud

# Check status
sovereign status
```

## Features

- 🧠 **AI-First** — Built-in local AI inference via Ollama with smart model selection
- 🖥️ **Cross-Platform** — Linux (server), macOS (personal), WSL2 (Windows)
- 📦 **App Marketplace** — 15+ one-click self-hosted apps
- 🔒 **Sovereign** — Your data stays on your hardware. Period.
- 🎯 **One Command** — `sovereign init` detects your hardware and sets everything up
- 💾 **Encrypted Backups** — Automated Restic-based backup system
- 🌐 **Auto-SSL** — Caddy reverse proxy with automatic HTTPS
- 🖥️ **Web Dashboard** — Beautiful dark-themed management interface

## Supported Platforms

| Platform | Mode | AI Performance |
|---|---|---|
| Linux (Ubuntu/Debian) | Full server | GPU passthrough (NVIDIA/AMD) or CPU |
| macOS (Apple Silicon) | Personal | Native Metal GPU acceleration |
| macOS (Intel) | Personal | CPU inference |
| Windows (WSL2) | WSL2 | Docker-based |

## App Catalog

| App | Category | Description |
|---|---|---|
| Nextcloud | Productivity | File sync & collaboration |
| Jellyfin | Media | Media streaming server |
| Immich | Media | Photo & video management |
| Home Assistant | IoT | Home automation |
| AdGuard Home | Network | Ad blocking |
| Vaultwarden | Security | Password manager |
| Gitea | Development | Git hosting |
| n8n | Automation | Workflow automation |
| Open WebUI | AI | ChatGPT-style local AI interface |
| + 6 more | Various | See `sovereign app list` |

## License

AGPL-3.0 — See [LICENSE](LICENSE)
