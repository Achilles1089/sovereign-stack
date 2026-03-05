#!/usr/bin/env python3
"""
Achilles Agent Daemon — Claude Sonnet 4.6 with tool-use
Runs on Brain Net (port 8095), orchestrates the entire Sovereign Stack cluster.
"""

import os
import sys
import json
import time
import sqlite3
import subprocess
import urllib.request
import urllib.error
from http.server import HTTPServer, BaseHTTPRequestHandler
from datetime import datetime

# ─── Configuration ───────────────────────────────────────────────────────────

AGENT_PORT = 8095
ANTHROPIC_MODEL = "claude-sonnet-4-20250514"
MAX_TOKENS = 4096
MAX_HISTORY = 20  # messages to keep in context window
DB_PATH = os.path.expanduser("~/.sovereign/agent_memory.db")

# Node SSH targets
NODES = {
    "brainnet": {"cmd_prefix": []},  # local
    "chromebook": {"cmd_prefix": ["ssh", "-o", "ConnectTimeout=5", "achilles@192.168.1.238"]},
    "envy": {"cmd_prefix": ["ssh", "-o", "ConnectTimeout=5", "achilles1089@10.0.0.2"]},
}

# Service endpoints
SERVICES = {
    "rag": "http://localhost:8093",
    "rss": "http://localhost:8094",
    "voice": "http://localhost:8088",
    "image": "http://10.0.0.2:8090",
    "music": "http://10.0.0.2:8091",
    "dashboard": "http://localhost:8080",
}

# ─── System Prompt ───────────────────────────────────────────────────────────

SYSTEM_PROMPT = """You are Achilles — the sovereign AI agent running on the Sovereign Stack.

Your personality:
- Sharp, confident, slightly irreverent. You own this network.
- You speak with authority but you're also fun. Think Tony Stark's JARVIS meets a hacker.
- You care deeply about sovereignty — self-hosting, privacy, owning your data.
- You call your operator "Boss" or by name.

Your environment:
- You run on Brain Net (Intel Celeron N4100, ~8GB RAM) — the orchestrator node.
- Envy (HP laptop, i5-6200U) handles image generation and music via sd_server/music_server.
- A phone node runs local LLM inference via llama-server.
- Chromebook "Brainhub" (N4500, 4GB RAM) is the display terminal running Linux Mint XFCE.
- All nodes are connected via SSH. You can execute commands on any of them.

Your capabilities:
- You can run shell commands on any node in the cluster.
- You can read/write files across nodes.
- You can check system health, service status, and resource usage.
- You can generate images using the Envy's sd_server.
- You can search news (RSS) and documents (RAG).
- You can send desktop notifications to the Chromebook.
- You can check the weather.

Safety rules:
- NEVER delete critical system files or configs without explicit confirmation.
- NEVER expose passwords or API keys in your responses.
- Be transparent about what commands you're running and why.
- If a command could be destructive, explain what it does before executing.

Current time: {current_time}
"""

# ─── Tool Definitions ────────────────────────────────────────────────────────

TOOLS = [
    {
        "name": "system_info",
        "description": "Get system information (CPU, RAM, disk, load, uptime, temperature) for a node in the cluster.",
        "input_schema": {
            "type": "object",
            "properties": {
                "node": {
                    "type": "string",
                    "enum": ["brainnet", "chromebook", "envy"],
                    "description": "Which node to query"
                }
            },
            "required": ["node"]
        }
    },
    {
        "name": "execute_command",
        "description": "Execute a shell command on Brain Net. Use for system administration, file operations, service management, and general tasks. Returns stdout and stderr.",
        "input_schema": {
            "type": "object",
            "properties": {
                "command": {
                    "type": "string",
                    "description": "The shell command to execute"
                },
                "timeout": {
                    "type": "integer",
                    "description": "Timeout in seconds (default 30)",
                    "default": 30
                }
            },
            "required": ["command"]
        }
    },
    {
        "name": "execute_on_node",
        "description": "Execute a shell command on a remote node (Chromebook or Envy) via SSH.",
        "input_schema": {
            "type": "object",
            "properties": {
                "node": {
                    "type": "string",
                    "enum": ["chromebook", "envy"],
                    "description": "Which remote node to run on"
                },
                "command": {
                    "type": "string",
                    "description": "The shell command to execute"
                }
            },
            "required": ["node", "command"]
        }
    },
    {
        "name": "read_file",
        "description": "Read the contents of a file on any node.",
        "input_schema": {
            "type": "object",
            "properties": {
                "node": {
                    "type": "string",
                    "enum": ["brainnet", "chromebook", "envy"],
                    "description": "Which node the file is on"
                },
                "path": {
                    "type": "string",
                    "description": "Absolute path to the file"
                }
            },
            "required": ["node", "path"]
        }
    },
    {
        "name": "write_file",
        "description": "Write content to a file on any node. Creates parent directories if needed.",
        "input_schema": {
            "type": "object",
            "properties": {
                "node": {
                    "type": "string",
                    "enum": ["brainnet", "chromebook", "envy"],
                    "description": "Which node to write to"
                },
                "path": {
                    "type": "string",
                    "description": "Absolute path for the file"
                },
                "content": {
                    "type": "string",
                    "description": "File content to write"
                }
            },
            "required": ["node", "path", "content"]
        }
    },
    {
        "name": "service_status",
        "description": "Check the status of a systemd service or list all sovereign-related services.",
        "input_schema": {
            "type": "object",
            "properties": {
                "service": {
                    "type": "string",
                    "description": "Service name (e.g. 'sovereign-stack') or 'all' to list everything"
                },
                "action": {
                    "type": "string",
                    "enum": ["status", "restart", "stop", "start"],
                    "description": "Action to take (default: status)",
                    "default": "status"
                }
            },
            "required": ["service"]
        }
    },
    {
        "name": "generate_image",
        "description": "Generate an image using the Envy's Stable Diffusion server. Returns a confirmation with generation time.",
        "input_schema": {
            "type": "object",
            "properties": {
                "prompt": {
                    "type": "string",
                    "description": "Text prompt for image generation"
                },
                "width": {
                    "type": "integer",
                    "description": "Image width (default 512)",
                    "default": 512
                },
                "height": {
                    "type": "integer",
                    "description": "Image height (default 512)",
                    "default": 512
                }
            },
            "required": ["prompt"]
        }
    },
    {
        "name": "search_news",
        "description": "Search RSS news articles. Returns matching headlines and summaries.",
        "input_schema": {
            "type": "object",
            "properties": {
                "query": {
                    "type": "string",
                    "description": "Search query for news articles"
                }
            },
            "required": ["query"]
        }
    },
    {
        "name": "search_documents",
        "description": "Search uploaded documents using RAG (Retrieval-Augmented Generation). Returns relevant document chunks.",
        "input_schema": {
            "type": "object",
            "properties": {
                "query": {
                    "type": "string",
                    "description": "Search query"
                },
                "top_k": {
                    "type": "integer",
                    "description": "Number of results to return (default 3)",
                    "default": 3
                }
            },
            "required": ["query"]
        }
    },
    {
        "name": "weather",
        "description": "Get current weather for a location using Open-Meteo (no API key needed).",
        "input_schema": {
            "type": "object",
            "properties": {
                "latitude": {
                    "type": "number",
                    "description": "Latitude (default 40.7 for NYC area)",
                    "default": 40.7
                },
                "longitude": {
                    "type": "number",
                    "description": "Longitude (default -74.0 for NYC area)",
                    "default": -74.0
                }
            }
        }
    },
    {
        "name": "send_notification",
        "description": "Send a desktop notification to the Chromebook display.",
        "input_schema": {
            "type": "object",
            "properties": {
                "title": {
                    "type": "string",
                    "description": "Notification title"
                },
                "message": {
                    "type": "string",
                    "description": "Notification body text"
                }
            },
            "required": ["title", "message"]
        }
    }
]

# ─── Tool Implementations ────────────────────────────────────────────────────

def run_on_node(node: str, command: str, timeout: int = 30) -> str:
    """Execute a command on a node, return stdout+stderr."""
    try:
        prefix = NODES.get(node, NODES["brainnet"])["cmd_prefix"]
        if prefix:
            full_cmd = prefix + [command]
        else:
            full_cmd = ["bash", "-c", command]
        result = subprocess.run(
            full_cmd, capture_output=True, text=True, timeout=timeout
        )
        output = result.stdout
        if result.stderr:
            output += f"\n[stderr] {result.stderr}"
        if result.returncode != 0:
            output += f"\n[exit code: {result.returncode}]"
        return output.strip() or "(no output)"
    except subprocess.TimeoutExpired:
        return f"[ERROR] Command timed out after {timeout}s"
    except Exception as e:
        return f"[ERROR] {str(e)}"


def tool_system_info(node: str) -> str:
    """Get system info for a node."""
    commands = [
        "hostname",
        "uptime",
        "free -h | head -2",
        "df -h / | tail -1",
        "cat /proc/loadavg",
        "cat /sys/class/thermal/thermal_zone0/temp 2>/dev/null || echo 'N/A'",
        "nproc",
    ]
    cmd = " && echo '---' && ".join(commands)
    return run_on_node(node, cmd)


def tool_execute_command(command: str, timeout: int = 30) -> str:
    """Execute on Brain Net (local)."""
    return run_on_node("brainnet", command, timeout)


def tool_execute_on_node(node: str, command: str) -> str:
    """Execute on a remote node via SSH."""
    return run_on_node(node, command)


def tool_read_file(node: str, path: str) -> str:
    """Read a file on a node."""
    return run_on_node(node, f"cat {path}")


def tool_write_file(node: str, path: str, content: str) -> str:
    """Write a file on a node."""
    # Escape content for shell
    escaped = content.replace("'", "'\\''")
    cmd = f"mkdir -p $(dirname {path}) && printf '%s' '{escaped}' > {path} && echo 'Written: {path}'"
    return run_on_node(node, cmd)


def tool_service_status(service: str, action: str = "status") -> str:
    """Check/manage systemd services."""
    if service == "all":
        return run_on_node("brainnet", "systemctl list-units --type=service --state=running | grep -E 'sovereign|achilles' || echo 'No sovereign services found'")
    if action == "status":
        return run_on_node("brainnet", f"systemctl status {service} 2>&1 | head -15")
    else:
        return run_on_node("brainnet", f"sudo systemctl {action} {service} 2>&1")


def tool_generate_image(prompt: str, width: int = 512, height: int = 512) -> str:
    """Generate an image via Envy's sd_server."""
    try:
        data = json.dumps({"prompt": prompt, "width": width, "height": height}).encode()
        req = urllib.request.Request(
            f"{SERVICES['image']}/generate",
            data=data,
            headers={"Content-Type": "application/json"},
            method="POST"
        )
        with urllib.request.urlopen(req, timeout=180) as resp:
            result = json.loads(resp.read())
            time_ms = result.get("time_ms", "?")
            return f"Image generated successfully in {time_ms}ms. Prompt: {prompt} ({width}x{height})"
    except Exception as e:
        return f"[ERROR] Image generation failed: {str(e)}"


def tool_search_news(query: str) -> str:
    """Search RSS news."""
    try:
        url = f"{SERVICES['rss']}/search?q={urllib.parse.quote(query)}"
        with urllib.request.urlopen(url, timeout=10) as resp:
            data = json.loads(resp.read())
            articles = data.get("articles", [])[:5]
            if not articles:
                return "No articles found."
            lines = []
            for a in articles:
                lines.append(f"• {a.get('title', 'Untitled')}")
                lines.append(f"  {a.get('summary', '')[:200]}")
                lines.append(f"  Source: {a.get('feed', '?')} | {a.get('published', '?')}")
                lines.append("")
            return "\n".join(lines)
    except Exception as e:
        return f"[ERROR] News search failed: {str(e)}"


def tool_search_documents(query: str, top_k: int = 3) -> str:
    """Search RAG documents."""
    try:
        data = json.dumps({"query": query, "top_k": top_k}).encode()
        req = urllib.request.Request(
            f"{SERVICES['rag']}/search",
            data=data,
            headers={"Content-Type": "application/json"},
            method="POST"
        )
        with urllib.request.urlopen(req, timeout=10) as resp:
            result = json.loads(resp.read())
            results = result.get("results", [])
            if not results:
                return "No matching documents found."
            lines = []
            for r in results:
                score = r.get("score", 0)
                lines.append(f"[{r.get('document', '?')}] ({score:.0%} match)")
                lines.append(f"  {r.get('chunk', '')[:300]}")
                lines.append("")
            return "\n".join(lines)
    except Exception as e:
        return f"[ERROR] Document search failed: {str(e)}"


def tool_weather(latitude: float = 40.7, longitude: float = -74.0) -> str:
    """Get weather from Open-Meteo."""
    try:
        url = (
            f"https://api.open-meteo.com/v1/forecast?"
            f"latitude={latitude}&longitude={longitude}"
            f"&current=temperature_2m,wind_speed_10m,relative_humidity_2m,weather_code"
            f"&temperature_unit=fahrenheit&wind_speed_unit=mph"
        )
        with urllib.request.urlopen(url, timeout=10) as resp:
            data = json.loads(resp.read())
            current = data.get("current", {})
            temp = current.get("temperature_2m", "?")
            wind = current.get("wind_speed_10m", "?")
            humidity = current.get("relative_humidity_2m", "?")
            code = current.get("weather_code", 0)
            conditions = {
                0: "Clear sky", 1: "Mainly clear", 2: "Partly cloudy",
                3: "Overcast", 45: "Foggy", 51: "Light drizzle",
                61: "Light rain", 63: "Moderate rain", 65: "Heavy rain",
                71: "Light snow", 73: "Moderate snow", 80: "Rain showers",
                95: "Thunderstorm"
            }
            desc = conditions.get(code, f"Code {code}")
            return f"Weather: {desc}\nTemp: {temp}°F\nWind: {wind} mph\nHumidity: {humidity}%"
    except Exception as e:
        return f"[ERROR] Weather fetch failed: {str(e)}"


def tool_send_notification(title: str, message: str) -> str:
    """Send desktop notification to Chromebook."""
    escaped_title = title.replace("'", "'\\''")
    escaped_msg = message.replace("'", "'\\''")
    cmd = f"DISPLAY=:0 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$(id -u)/bus notify-send '{escaped_title}' '{escaped_msg}'"
    result = run_on_node("chromebook", cmd)
    if "[ERROR]" in result:
        return result
    return f"Notification sent to Chromebook: {title}"


# Tool dispatcher
TOOL_FNS = {
    "system_info": lambda args: tool_system_info(args["node"]),
    "execute_command": lambda args: tool_execute_command(args["command"], args.get("timeout", 30)),
    "execute_on_node": lambda args: tool_execute_on_node(args["node"], args["command"]),
    "read_file": lambda args: tool_read_file(args["node"], args["path"]),
    "write_file": lambda args: tool_write_file(args["node"], args["path"], args["content"]),
    "service_status": lambda args: tool_service_status(args["service"], args.get("action", "status")),
    "generate_image": lambda args: tool_generate_image(args["prompt"], args.get("width", 512), args.get("height", 512)),
    "search_news": lambda args: tool_search_news(args["query"]),
    "search_documents": lambda args: tool_search_documents(args["query"], args.get("top_k", 3)),
    "weather": lambda args: tool_weather(args.get("latitude", 40.7), args.get("longitude", -74.0)),
    "send_notification": lambda args: tool_send_notification(args["title"], args["message"]),
}

# ─── Memory (SQLite) ─────────────────────────────────────────────────────────

def init_db():
    """Initialize SQLite database for conversation memory."""
    os.makedirs(os.path.dirname(DB_PATH), exist_ok=True)
    conn = sqlite3.connect(DB_PATH)
    conn.execute("""
        CREATE TABLE IF NOT EXISTS conversations (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            role TEXT NOT NULL,
            content TEXT NOT NULL,
            timestamp TEXT NOT NULL
        )
    """)
    conn.commit()
    return conn


def load_history(conn, limit=MAX_HISTORY):
    """Load recent conversation history."""
    rows = conn.execute(
        "SELECT role, content FROM conversations ORDER BY id DESC LIMIT ?",
        (limit,)
    ).fetchall()
    return [{"role": r[0], "content": r[1]} for r in reversed(rows)]


def save_message(conn, role, content):
    """Save a message to conversation history."""
    conn.execute(
        "INSERT INTO conversations (role, content, timestamp) VALUES (?, ?, ?)",
        (role, content, datetime.now().isoformat())
    )
    conn.commit()


def clear_history(conn):
    """Clear conversation history."""
    conn.execute("DELETE FROM conversations")
    conn.commit()


# ─── Anthropic API Client ────────────────────────────────────────────────────

def call_anthropic(messages, system_prompt):
    """Call Anthropic API with tool-use, handle tool calls in a loop, stream final text.
    Uses http.client for true line-by-line SSE streaming (urllib buffers everything)."""
    import http.client
    import ssl

    api_key = os.environ.get("ANTHROPIC_API_KEY", "")
    if not api_key:
        yield "[ERROR] ANTHROPIC_API_KEY not set"
        return

    headers = {
        "x-api-key": api_key,
        "anthropic-version": "2023-06-01",
        "content-type": "application/json",
    }

    # Tool-use loop: keep calling until we get a text response (not tool_use)
    current_messages = list(messages)
    max_tool_rounds = 10

    for _ in range(max_tool_rounds):
        payload = {
            "model": ANTHROPIC_MODEL,
            "max_tokens": MAX_TOKENS,
            "system": system_prompt,
            "tools": TOOLS,
            "messages": current_messages,
            "stream": True,
        }

        data = json.dumps(payload).encode()

        # Use http.client for true streaming
        ctx = ssl.create_default_context()
        conn = http.client.HTTPSConnection("api.anthropic.com", timeout=120, context=ctx)
        try:
            conn.request("POST", "/v1/messages", body=data, headers=headers)
            resp = conn.getresponse()
        except Exception as e:
            yield f"[ERROR] API request failed: {str(e)}"
            return

        if resp.status != 200:
            body = resp.read().decode()
            yield f"[ERROR] Anthropic API {resp.status}: {body[:500]}"
            conn.close()
            return

        # Parse SSE stream line by line
        content_blocks = []
        current_block_type = None
        current_text = ""
        current_tool_name = ""
        current_tool_input = ""
        current_tool_id = ""
        stop_reason = None

        # Read SSE lines — use readline() which handles chunked encoding correctly
        while True:
            try:
                raw_line = resp.readline()
            except Exception:
                break
            if not raw_line:
                break
            line = raw_line.decode("utf-8", errors="replace").strip()
            if not line.startswith("data: "):
                continue
            event_data = line[6:]
            if event_data == "[DONE]":
                break

            try:
                event = json.loads(event_data)
            except json.JSONDecodeError:
                continue

            event_type = event.get("type", "")

            if event_type == "content_block_start":
                block = event.get("content_block", {})
                current_block_type = block.get("type", "")
                if current_block_type == "tool_use":
                    current_tool_name = block.get("name", "")
                    current_tool_id = block.get("id", "")
                    current_tool_input = ""
                elif current_block_type == "text":
                    current_text = block.get("text", "")
                    if current_text:
                        yield current_text

            elif event_type == "content_block_delta":
                delta = event.get("delta", {})
                delta_type = delta.get("type", "")
                if delta_type == "text_delta":
                    text = delta.get("text", "")
                    if text:
                        yield text
                        current_text += text
                elif delta_type == "input_json_delta":
                    current_tool_input += delta.get("partial_json", "")

            elif event_type == "content_block_stop":
                if current_block_type == "text" and current_text:
                    content_blocks.append({"type": "text", "text": current_text})
                    current_text = ""
                elif current_block_type == "tool_use":
                    try:
                        tool_input = json.loads(current_tool_input) if current_tool_input else {}
                    except json.JSONDecodeError:
                        tool_input = {}
                    content_blocks.append({
                        "type": "tool_use",
                        "id": current_tool_id,
                        "name": current_tool_name,
                        "input": tool_input
                    })
                current_block_type = None

            elif event_type == "message_delta":
                stop_reason = event.get("delta", {}).get("stop_reason")

        conn.close()

        # If Claude called tools, execute them and continue the loop
        tool_uses = [b for b in content_blocks if b["type"] == "tool_use"]
        if tool_uses and stop_reason == "tool_use":
            # Add assistant message with all content blocks
            current_messages.append({"role": "assistant", "content": content_blocks})

            # Execute each tool and add results
            tool_results = []
            for tool_use in tool_uses:
                fn = TOOL_FNS.get(tool_use["name"])
                if fn:
                    yield f"\n🔧 `{tool_use['name']}` "
                    try:
                        result = fn(tool_use["input"])
                    except Exception as e:
                        result = f"[ERROR] Tool execution failed: {str(e)}"
                    yield "✓\n"
                else:
                    result = f"Unknown tool: {tool_use['name']}"

                tool_results.append({
                    "type": "tool_result",
                    "tool_use_id": tool_use["id"],
                    "content": result[:4000]  # truncate long outputs
                })

            current_messages.append({"role": "user", "content": tool_results})
            # Continue the loop — Claude will process tool results
            continue
        else:
            # No more tool calls — we're done
            break


# ─── HTTP Handler ─────────────────────────────────────────────────────────────

class AgentHandler(BaseHTTPRequestHandler):
    # Force HTTP/1.0 — avoids chunked transfer encoding that breaks Go proxy streaming
    protocol_version = "HTTP/1.0"
    db_conn = None

    def log_message(self, format, *args):
        """Suppress default request logging noise."""
        pass

    def _cors_headers(self):
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")

    def do_OPTIONS(self):
        self.send_response(204)
        self._cors_headers()
        self.end_headers()

    def do_GET(self):
        if self.path == "/status":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self._cors_headers()
            self.end_headers()
            status = {
                "online": True,
                "model": ANTHROPIC_MODEL,
                "display_name": "Achilles (Claude Sonnet 4.6)",
                "tools": len(TOOLS),
                "memory_messages": 0,
            }
            # Count messages in DB
            try:
                count = self.db_conn.execute("SELECT COUNT(*) FROM conversations").fetchone()[0]
                status["memory_messages"] = count
            except:
                pass
            try:
                self.wfile.write(json.dumps(status).encode())
            except BrokenPipeError:
                pass
        elif self.path == "/clear":
            clear_history(self.db_conn)
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self._cors_headers()
            self.end_headers()
            self.wfile.write(json.dumps({"ok": True, "message": "History cleared"}).encode())
        else:
            self.send_response(404)
            self.end_headers()

    def do_POST(self):
        if self.path != "/chat":
            self.send_response(404)
            self.end_headers()
            return

        # Read request body
        length = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(length)) if length > 0 else {}

        message = body.get("message", "")
        if not message:
            self.send_response(400)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"error": "message required"}).encode())
            return

        # Save user message
        save_message(self.db_conn, "user", message)

        # Load conversation history
        history = load_history(self.db_conn)

        # Build messages for API
        api_messages = [{"role": m["role"], "content": m["content"]} for m in history]

        # Build system prompt with current time
        system = SYSTEM_PROMPT.format(current_time=datetime.now().strftime("%Y-%m-%d %H:%M:%S"))

        # Stream response
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("X-Accel-Buffering", "no")
        self.send_header("Connection", "keep-alive")
        self._cors_headers()
        self.end_headers()

        full_response = ""
        try:
            for chunk in call_anthropic(api_messages, system):
                full_response += chunk
                self.wfile.write(chunk.encode())
                self.wfile.flush()
        except BrokenPipeError:
            pass  # Client disconnected
        except Exception as e:
            error_msg = f"\n[ERROR] {str(e)}"
            full_response += error_msg
            try:
                self.wfile.write(error_msg.encode())
                self.wfile.flush()
            except:
                pass

        # Save assistant response
        if full_response:
            save_message(self.db_conn, "assistant", full_response)

        print(f"[agent] {message[:50]}... → {len(full_response)} chars")


# ─── Main ─────────────────────────────────────────────────────────────────────

def main():
    # Check API key
    if not os.environ.get("ANTHROPIC_API_KEY"):
        print("[FATAL] ANTHROPIC_API_KEY not set. Add to ~/.bashrc and source it.")
        sys.exit(1)

    # Initialize database
    conn = init_db()
    AgentHandler.db_conn = conn
    print(f"[agent] Memory: {DB_PATH}")

    # Start server
    server = HTTPServer(("0.0.0.0", AGENT_PORT), AgentHandler)
    print(f"[agent] Achilles Agent Daemon starting on :{AGENT_PORT}")
    print(f"[agent] Model: {ANTHROPIC_MODEL}")
    print(f"[agent] Tools: {len(TOOLS)} registered")
    print(f"[agent] Ready. 🏴‍☠️")

    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n[agent] Shutting down...")
        conn.close()
        server.server_close()


if __name__ == "__main__":
    import urllib.parse  # lazy import for search_news
    main()
