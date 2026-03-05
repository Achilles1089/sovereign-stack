# Sovereign OS — Project Status & Context

> **PERSISTENT MEMORY DOC** — This file is the handoff document for any new AI session.
> Read this FIRST before doing anything on this project.

## Last Updated: 2026-03-05T01:10:00-05:00

## Current State

### What's Built (Phases 0-4 ✅)
| Phase | Feature | Status | Commit |
|-------|---------|--------|--------|
| 0 | Music Gen Debug | ✅ Done | `2602289` |
| 1 | Voice Assistant | ✅ Done | `2602289` |
| 2 | RAG Doc Chat | ✅ Done | `2602289` |
| 3 | AI Art Gallery | ✅ Done | `cda6dee` |
| 4 | News Reader | ✅ Done | `d3d3c2a` |

### What's In Progress (Phase 5: Sovereign OS)
**Phase A — Foundation:** ✅ Complete (2026-03-04).
- ✅ Chromebook flashed with Linux Mint 22 XFCE (hostname: Brainhub)
- ✅ SSH key auth working: `ssh -p 2223 achilles@localhost` via Brain Net tunnel
- ✅ Base packages: curl, git, htop, nmap, xdotool, xbindkeys, cmatrix, redshift, chromium, unclutter
- ✅ LightDM auto-login configured
- ✅ Chromium kiosk mode → `http://192.168.1.206:8080` (autostart)
- ✅ XFCE dark theme: Mint-Y-Dark
- ✅ Screensaver/power management disabled (always-on)
- ✅ Global hotkeys: Super+A (AI), Super+T (terminal), Super+M (media), Super+G (games)
- ✅ Redshift night mode (6500K→3500K)
- ✅ ANTHROPIC_API_KEY deployed to Brain Net ~/.bashrc
- ⏳ HDMI output (needs physical testing)

**Phase B — Achilles Agent Daemon:** ✅ Complete (2026-03-05).
- ✅ `achilles_agent.py` — Python daemon on Brain Net:8095
- ✅ Claude Sonnet 4.6 with tool-use SSE streaming (http.client)
- ✅ 11 tools: system_info, execute_command, execute_on_node, read_file, write_file, service_status, generate_image, search_news, search_documents, weather, send_notification
- ✅ SQLite conversation memory (~/.sovereign/agent_memory.db)
- ✅ Go proxy: /api/agent/chat, /api/agent/status, /api/agent/clear
- ✅ Dashboard: /agent toggle, terminal header branding
- ✅ systemd service file created (needs sudo install)
- ✅ End-to-end streaming chat verified

**Phase C — Next:** Dashboard build+deploy, HDMI testing, further integrations.

## Architecture

### Network Topology
```
Mac (dev machine, Antigravity runs here)
  └── SSH tunnel: localhost:2222 → Brain Net:22
  └── SSH tunnel: localhost:2223 → Chromebook:22 (via Brain Net)

Brain Net (192.168.1.206) — Intel Celeron, orchestrator
  ├── sovereign-stack Go binary (:8080)
  ├── achilles_agent.py (:8095)  ← Achilles Agent (Claude Sonnet 4.6)
  ├── rag_server.py (:8093)
  ├── rss_server.py (:8094) — PID 1371823
  ├── voice_server.py (:8088)
  └── SSH to Envy (10.0.0.2)

Envy (10.0.0.2) — AMD GPU node
  ├── sd_server (:8090) — image gen
  └── music_server.py (:8091)

Chromebook / Brainhub (192.168.1.238) — N4500, 4GB RAM
  └── Linux Mint 22 XFCE, SSH server running
```

### SSH Access Patterns
```bash
# Brain Net (from Mac)
ssh -p 2222 achilles1089@localhost

# Chromebook (from Mac, via Brain Net tunnel)
# First ensure tunnel: ssh -p 2222 -L 2223:192.168.1.238:22 -N -f achilles1089@localhost
ssh -p 2223 achilles@localhost

# Envy (from Brain Net)
ssh achilles1089@10.0.0.2
```

### Credentials Location
All credentials in `docs/.credentials.md` (gitignored). Includes:
- Brain Net login, Envy login, Chromebook login
- WiFi credentials
- Anthropic API key (Claude Sonnet 4.6)

## Services Running on Brain Net
| Service | Port | Type | Status |
|---------|------|------|--------|
| sovereign-stack | 8080 | systemd (Go) | Active |
| rag_server.py | 8093 | manual (Python) | Running |
| rss_server.py | 8094 | manual (Python) | Running (PID 1371823) |
| voice_server.py | 8088 | manual (Python) | Needs restart |

## Key Files
| File | Purpose |
|------|---------|
| `internal/server/server.go` | Go backend — all API routes, proxies |
| `internal/config/config.go` | Config structs (AIConfig, RAGHost, etc) |
| `dashboard/src/App.tsx` | React router, tab navigation |
| `dashboard/src/pages/AI.tsx` | AI chat page (terminal commands, gallery) |
| `dashboard/src/pages/News.tsx` | News reader page |
| `dashboard/src/api/client.ts` | Frontend API client |
| `scripts/rag_server.py` | RAG document search server |
| `scripts/rss_server.py` | RSS news aggregation server |
| `docs/.credentials.md` | All passwords and API keys (GITIGNORED) |
| `docs/HARDWARE_INVENTORY.md` | Hardware specs |

## Next Steps (Phase B: Achilles Agent Daemon)
1. Build `achilles_agent.py` — Python agent with Anthropic API (tool-use mode)
2. WebSocket server (port 8095) for real-time UI communication
3. Core tool registry (10 tools: execute_command, system_info, generate_image, etc.)
4. Go proxy: `/api/agent/chat` → agent daemon
5. Dashboard AI tab: switch between local LLM and Achilles Agent
6. Safety model: green/yellow/red tool tiers
7. systemd service: `achilles-agent.service`

## Full Task List
See: `brain/<conversation-id>/task.md` (in Antigravity artifacts)
Or the implementation plan: `brain/<conversation-id>/implementation_plan.md`

## Build & Deploy Workflow
```bash
# Go backend
cd /path/to/sovereign-stack
GOOS=linux GOARCH=amd64 go build -o sovereign-linux-amd64 .
scp -P 2222 sovereign-linux-amd64 achilles1089@localhost:~/sovereign-stack/sovereign

# Frontend
cd dashboard && npm run build
scp -r -P 2222 dist/ achilles1089@localhost:~/sovereign-stack/dashboard/dist/

# Restart service
ssh -p 2222 achilles1089@localhost 'echo habeeb1089 | sudo -S systemctl restart sovereign-stack'
```
