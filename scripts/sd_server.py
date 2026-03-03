#!/usr/bin/env python3
"""
sd_server.py — Lightweight HTTP wrapper for stable-diffusion.cpp
Runs on the mini PC, serves image generation via POST /generate.

Usage:
    python3 sd_server.py [--port 8090] [--sd-path ./sd] [--model ./sd-turbo-q8_0.gguf]
"""

import argparse
import base64
import json
import os
import subprocess
import tempfile
import time
from http.server import HTTPServer, BaseHTTPRequestHandler

# Defaults
DEFAULT_PORT = 8090
DEFAULT_SD_PATH = "./sd"  # or "sd.exe" on Windows
DEFAULT_MODEL = "./sd-turbo-q8_0.gguf"


class SDHandler(BaseHTTPRequestHandler):
    sd_path = DEFAULT_SD_PATH
    model_path = DEFAULT_MODEL

    def do_GET(self):
        if self.path == "/status":
            self._send_json({
                "online": True,
                "model": os.path.basename(self.model_path),
                "sd_path": self.sd_path,
            })
        else:
            self.send_error(404)

    def do_POST(self):
        if self.path == "/generate":
            self._handle_generate()
        else:
            self.send_error(404)

    def _handle_generate(self):
        content_len = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(content_len)
        try:
            req = json.loads(body)
        except json.JSONDecodeError:
            self._send_json({"error": "invalid JSON"}, 400)
            return

        prompt = req.get("prompt", "")
        width = req.get("width", 512)
        height = req.get("height", 512)
        steps = req.get("steps", 1)  # SD Turbo = 1 step
        seed = req.get("seed", -1)

        if not prompt:
            self._send_json({"error": "prompt required"}, 400)
            return

        # Generate image via sd.cpp CLI
        with tempfile.NamedTemporaryFile(suffix=".png", delete=False) as tmp:
            out_path = tmp.name

        start_time = time.time()
        cmd = [
            self.sd_path,
            "-m", self.model_path,
            "-p", prompt,
            "-o", out_path,
            "-W", str(width),
            "-H", str(height),
            "--steps", str(steps),
            "--seed", str(seed),
        ]

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=180,
            )
            elapsed_ms = int((time.time() - start_time) * 1000)

            if result.returncode != 0 or not os.path.exists(out_path):
                self._send_json({
                    "error": f"sd.cpp failed: {result.stderr[:500]}",
                    "time_ms": elapsed_ms,
                }, 500)
                return

            # Read and base64-encode the image
            with open(out_path, "rb") as f:
                img_data = f.read()
            b64 = base64.b64encode(img_data).decode("utf-8")

            self._send_json({
                "image": f"data:image/png;base64,{b64}",
                "time_ms": elapsed_ms,
                "width": width,
                "height": height,
                "steps": steps,
            })
        except subprocess.TimeoutExpired:
            self._send_json({"error": "generation timed out"}, 504)
        finally:
            try:
                os.unlink(out_path)
            except OSError:
                pass

    def _send_json(self, data, status=200):
        body = json.dumps(data).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.end_headers()

    def log_message(self, format, *args):
        print(f"[sd_server] {args[0]} {args[1]}")


def main():
    parser = argparse.ArgumentParser(description="SD Server — HTTP wrapper for stable-diffusion.cpp")
    parser.add_argument("--port", type=int, default=DEFAULT_PORT, help="Port to listen on")
    parser.add_argument("--sd-path", default=DEFAULT_SD_PATH, help="Path to sd executable")
    parser.add_argument("--model", default=DEFAULT_MODEL, help="Path to GGUF model file")
    args = parser.parse_args()

    SDHandler.sd_path = args.sd_path
    SDHandler.model_path = args.model

    server = HTTPServer(("0.0.0.0", args.port), SDHandler)
    print(f"[sd_server] Listening on 0.0.0.0:{args.port}")
    print(f"[sd_server] Model: {args.model}")
    print(f"[sd_server] SD binary: {args.sd_path}")
    server.serve_forever()


if __name__ == "__main__":
    main()
