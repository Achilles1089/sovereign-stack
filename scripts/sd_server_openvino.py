#!/usr/bin/env python3
"""
sd_server_openvino.py — Intel-optimized image generation server
Uses OpenVINO + SDXS (1-step distilled diffusion) for fast inference on Celerons.
Generates images in 1 step instead of 20+.

Usage:
    python3 sd_server_openvino.py [--port 8090] [--steps 1] [--width 512] [--height 512]
"""

import argparse
import base64
import io
import json
import os
import time
from http.server import HTTPServer, BaseHTTPRequestHandler
from threading import Lock

# Global pipeline (loaded once at startup)
pipeline = None
pipe_lock = Lock()

DEFAULT_PORT = 8090
DEFAULT_STEPS = 1
DEFAULT_WIDTH = 512
DEFAULT_HEIGHT = 512


def load_pipeline():
    """Load the SDXS 1-step pipeline with OpenVINO optimization."""
    global pipeline

    # Force OpenVINO threading config via env (must be set before import)
    os.environ["OMP_NUM_THREADS"] = "4"
    os.environ["OMP_WAIT_POLICY"] = "PASSIVE"

    print("[sd_server] Loading OpenVINO SDXS pipeline...")
    start = time.time()

    from optimum.intel import OVStableDiffusionPipeline

    # Pre-exported SDXS OpenVINO model — no export needed, loads instantly
    model_id = "rupeshs/sdxs-512-0.9-openvino"
    cache_dir = os.path.expanduser("~/.cache/openvino_models/sdxs")

    if os.path.exists(os.path.join(cache_dir, "model_index.json")):
        print("[sd_server] Loading cached SDXS model...")
        pipeline = OVStableDiffusionPipeline.from_pretrained(
            cache_dir,
            device="CPU",
            ov_config={
                "INFERENCE_NUM_THREADS": 4,
                "PERFORMANCE_HINT": "LATENCY",
                "ENABLE_CPU_PINNING": "YES",
            },
        )
    else:
        print("[sd_server] Downloading pre-exported SDXS OpenVINO model (first run)...")
        pipeline = OVStableDiffusionPipeline.from_pretrained(
            model_id,
            device="CPU",
            ov_config={
                "INFERENCE_NUM_THREADS": 4,
                "PERFORMANCE_HINT": "LATENCY",
                "ENABLE_CPU_PINNING": "YES",
            },
        )
        pipeline.save_pretrained(cache_dir)
        print(f"[sd_server] Model cached to {cache_dir}")

    elapsed = time.time() - start
    print(f"[sd_server] Pipeline loaded in {elapsed:.1f}s")


class SDHandler(BaseHTTPRequestHandler):
    default_steps = DEFAULT_STEPS
    default_width = DEFAULT_WIDTH
    default_height = DEFAULT_HEIGHT

    def do_GET(self):
        if self.path == "/" or self.path == "/status":
            self._send_json({
                "online": True,
                "model": "SDXS-512 (OpenVINO, 1-step)",
                "engine": "openvino",
                "default_steps": self.default_steps,
                "default_size": f"{self.default_width}x{self.default_height}",
            })
        else:
            self.send_error(404)

    def do_POST(self):
        if self.path in ("/generate", "/sdapi/v1/txt2img", "/v1/images/generations"):
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
        width = req.get("width", self.default_width)
        height = req.get("height", self.default_height)
        steps = req.get("steps", self.default_steps)
        seed = req.get("seed", None)

        if not prompt:
            self._send_json({"error": "prompt required"}, 400)
            return

        # Clamp dimensions to multiples of 8
        width = (width // 8) * 8
        height = (height // 8) * 8

        print(f"[sd_server] Generating: '{prompt}' {width}x{height} steps={steps}")
        start_time = time.time()

        try:
            import torch
            generator = None
            if seed is not None:
                generator = torch.Generator().manual_seed(int(seed))

            with pipe_lock:
                result = pipeline(
                    prompt=prompt,
                    num_inference_steps=steps,
                    width=width,
                    height=height,
                    guidance_scale=1.0,  # LCM uses guidance_scale=1.0
                    generator=generator,
                )

            image = result.images[0]
            elapsed_ms = int((time.time() - start_time) * 1000)

            # Convert to base64 PNG
            buf = io.BytesIO()
            image.save(buf, format="PNG")
            b64 = base64.b64encode(buf.getvalue()).decode("utf-8")

            print(f"[sd_server] Generated in {elapsed_ms}ms")
            self._send_json({
                "image": f"data:image/png;base64,{b64}",
                "time_ms": elapsed_ms,
                "width": width,
                "height": height,
                "steps": steps,
            })

        except Exception as e:
            elapsed_ms = int((time.time() - start_time) * 1000)
            print(f"[sd_server] Error: {e}")
            self._send_json({
                "error": str(e),
                "time_ms": elapsed_ms,
            }, 500)

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
    parser = argparse.ArgumentParser(description="SD Server — OpenVINO + SDXS")
    parser.add_argument("--port", type=int, default=DEFAULT_PORT)
    parser.add_argument("--steps", type=int, default=DEFAULT_STEPS, help="Default inference steps (2-8)")
    parser.add_argument("--width", type=int, default=DEFAULT_WIDTH, help="Default image width")
    parser.add_argument("--height", type=int, default=DEFAULT_HEIGHT, help="Default image height")
    args = parser.parse_args()

    SDHandler.default_steps = args.steps
    SDHandler.default_width = args.width
    SDHandler.default_height = args.height

    # Load pipeline at startup
    load_pipeline()

    server = HTTPServer(("0.0.0.0", args.port), SDHandler)
    print(f"[sd_server] Listening on 0.0.0.0:{args.port}")
    print(f"[sd_server] Defaults: {args.width}x{args.height}, {args.steps} steps")
    server.serve_forever()


if __name__ == "__main__":
    main()
