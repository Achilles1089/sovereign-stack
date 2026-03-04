#!/usr/bin/env python3
"""
music_server.py — Spectrogram-to-Audio music generation server for Sovereign Stack
Runs on Envy (HP laptop), port 8091. 100% local — no external APIs.

Approach:
  1. Uses the existing `sd` binary to generate a mel-spectrogram image from a text prompt
  2. Converts the spectrogram image to audio using additive synthesis (pure Python)
  3. Returns base64-encoded WAV

Endpoints:
  POST /generate  - Prompt → spectrogram → audio → base64 WAV
  GET  /status    - Check if music gen is available
"""

import argparse
import base64
import json
import os
import subprocess
import tempfile
import time
import struct
import math
import zlib
import socketserver
from http.server import BaseHTTPRequestHandler

DEFAULT_PORT = 8091
DEFAULT_SD_BIN = os.path.expanduser("~/sd_server/sd")
DEFAULT_MODEL = ""

SAMPLE_RATE = 44100
DURATION_S = 5.0


def _read_png_grayscale(path: str, target_w: int, target_h: int):
    """Read a PNG file and return grayscale pixel data as a 2D list [height][width].
    Pure Python PNG reader — handles most standard PNG files.
    Returns None on failure.
    """
    try:
        with open(path, "rb") as f:
            data = f.read()

        # Verify PNG signature
        if data[:8] != b'\x89PNG\r\n\x1a\n':
            print(f"[music] Not a valid PNG: {path}")
            return None

        # Parse chunks
        pos = 8
        width = height = bit_depth = color_type = 0
        raw_idat = b""

        while pos < len(data):
            chunk_len = int.from_bytes(data[pos:pos+4], "big")
            chunk_type = data[pos+4:pos+8]
            chunk_data = data[pos+8:pos+8+chunk_len]
            pos += 12 + chunk_len  # 4 len + 4 type + data + 4 crc

            if chunk_type == b"IHDR":
                width = int.from_bytes(chunk_data[0:4], "big")
                height = int.from_bytes(chunk_data[4:8], "big")
                bit_depth = chunk_data[8]
                color_type = chunk_data[9]
            elif chunk_type == b"IDAT":
                raw_idat += chunk_data
            elif chunk_type == b"IEND":
                break

        if width == 0 or height == 0:
            return None

        # Decompress
        decompressed = zlib.decompress(raw_idat)

        # Determine bytes per pixel
        if color_type == 0:      # Grayscale
            bpp = 1
        elif color_type == 2:    # RGB
            bpp = 3
        elif color_type == 4:    # Grayscale + Alpha
            bpp = 2
        elif color_type == 6:    # RGBA
            bpp = 4
        else:
            bpp = 3  # Assume RGB

        stride = 1 + width * bpp  # 1 byte filter + pixel data per row

        # Parse rows (skip filter byte)
        pixels = []
        for row in range(height):
            row_start = row * stride + 1  # Skip filter byte
            row_end = row_start + width * bpp
            row_bytes = decompressed[row_start:row_end]

            row_gray = []
            for x in range(width):
                offset = x * bpp
                if color_type == 0:   # Grayscale
                    row_gray.append(row_bytes[offset] / 255.0)
                elif color_type == 2: # RGB → grayscale
                    r, g, b = row_bytes[offset], row_bytes[offset+1], row_bytes[offset+2]
                    row_gray.append((0.299 * r + 0.587 * g + 0.114 * b) / 255.0)
                elif color_type == 4: # Grayscale+Alpha
                    row_gray.append(row_bytes[offset] / 255.0)
                elif color_type == 6: # RGBA
                    r, g, b = row_bytes[offset], row_bytes[offset+1], row_bytes[offset+2]
                    row_gray.append((0.299 * r + 0.587 * g + 0.114 * b) / 255.0)
            pixels.append(row_gray)

        # Simple nearest-neighbor resize if needed
        if width != target_w or height != target_h:
            resized = []
            for ty in range(target_h):
                src_y = int(ty * height / target_h)
                src_y = min(src_y, height - 1)
                row = []
                for tx in range(target_w):
                    src_x = int(tx * width / target_w)
                    src_x = min(src_x, width - 1)
                    row.append(pixels[src_y][src_x])
                resized.append(row)
            return resized

        return pixels

    except Exception as e:
        print(f"[music] PNG read error: {e}")
        return None


def spectrogram_image_to_audio(image_path: str, output_path: str) -> bool:
    """Convert a spectrogram image to audio using pixel intensity → frequency mapping.
    Pure Python — no ffmpeg or numpy dependency.
    """
    try:
        pixels = _read_png_grayscale(image_path, 256, 128)
        if pixels is None:
            return False

        width = 256
        height = 128
        num_samples = int(SAMPLE_RATE * DURATION_S)
        samples_per_col = num_samples // width
        audio = [0.0] * num_samples

        min_freq = 80.0
        max_freq = 8000.0

        for col in range(width):
            for row in range(height):
                amplitude = pixels[height - 1 - row][col]
                if amplitude < 0.05:
                    continue

                freq = min_freq * ((max_freq / min_freq) ** (row / max(height - 1, 1)))
                phase_offset = (col * 0.1 + row * 0.3)

                for s in range(samples_per_col):
                    sample_idx = col * samples_per_col + s
                    if sample_idx < num_samples:
                        t = sample_idx / SAMPLE_RATE
                        audio[sample_idx] += amplitude * 0.003 * math.sin(
                            2.0 * math.pi * freq * t + phase_offset
                        )

        max_val = max(abs(s) for s in audio) or 1.0
        audio = [s / max_val * 0.8 for s in audio]

        write_wav(output_path, audio, SAMPLE_RATE)
        return True

    except Exception as e:
        print(f"[music] spectrogram_to_audio error: {e}")
        return False


def write_wav(path: str, samples: list, sample_rate: int):
    """Write a mono 16-bit PCM WAV file."""
    num_samples = len(samples)
    data_size = num_samples * 2
    file_size = 36 + data_size

    with open(path, "wb") as f:
        f.write(b"RIFF")
        f.write(struct.pack("<I", file_size))
        f.write(b"WAVE")
        f.write(b"fmt ")
        f.write(struct.pack("<I", 16))
        f.write(struct.pack("<H", 1))
        f.write(struct.pack("<H", 1))
        f.write(struct.pack("<I", sample_rate))
        f.write(struct.pack("<I", sample_rate * 2))
        f.write(struct.pack("<H", 2))
        f.write(struct.pack("<H", 16))
        f.write(b"data")
        f.write(struct.pack("<I", data_size))
        for s in samples:
            clamped = max(-1.0, min(1.0, s))
            f.write(struct.pack("<h", int(clamped * 32767)))


class MusicHandler(BaseHTTPRequestHandler):
    sd_bin = DEFAULT_SD_BIN
    model_path = ""

    def do_GET(self):
        if self.path == "/status":
            sd_ok = os.path.exists(self.sd_bin)
            self._send_json({
                "online": sd_ok,
                "engine": "sd-spectrogram",
                "model": os.path.basename(self.model_path) if self.model_path else "sdxs (default)",
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
        if not prompt:
            self._send_json({"error": "prompt required"}, 400)
            return

        music_prompt = f"a spectral visualization of {prompt}, mel spectrogram, frequency bands, audio waveform art"

        with tempfile.NamedTemporaryFile(suffix=".png", delete=False) as tmp:
            output_img = tmp.name

        start_time = time.time()

        cmd = [self.sd_bin, "-p", music_prompt, "-o", output_img,
               "--steps", "4", "-W", "256", "-H", "128"]
        if self.model_path:
            cmd.extend(["-m", self.model_path])

        try:
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
            gen_time = time.time() - start_time

            if result.returncode != 0 or not os.path.exists(output_img):
                self._cleanup(output_img)
                self._send_json({"error": f"sd failed: {result.stderr[:300]}"}, 500)
                return

            with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as wav_tmp:
                wav_path = wav_tmp.name

            if not spectrogram_image_to_audio(output_img, wav_path):
                self._cleanup(output_img, wav_path)
                self._send_json({"error": "spectrogram to audio conversion failed"}, 500)
                return

            total_time = time.time() - start_time

            with open(wav_path, "rb") as f:
                wav_data = f.read()
            b64 = base64.b64encode(wav_data).decode("utf-8")

            self._cleanup(output_img, wav_path)
            self._send_json({
                "audio": f"data:audio/wav;base64,{b64}",
                "prompt": prompt,
                "gen_time_ms": int(gen_time * 1000),
                "total_time_ms": int(total_time * 1000),
                "duration_s": DURATION_S,
            })

        except subprocess.TimeoutExpired:
            self._cleanup(output_img)
            self._send_json({"error": "generation timed out"}, 504)

    def _cleanup(self, *paths):
        for p in paths:
            try:
                os.unlink(p)
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
        print(f"[music] {args[0]} {args[1]}")


class ReusableTCPServer(socketserver.TCPServer):
    allow_reuse_address = True
    allow_reuse_port = True


def main():
    parser = argparse.ArgumentParser(description="Music Gen Server — Spectrogram → Audio")
    parser.add_argument("--port", type=int, default=DEFAULT_PORT)
    parser.add_argument("--sd-bin", default=DEFAULT_SD_BIN)
    parser.add_argument("--model", default=DEFAULT_MODEL)
    args = parser.parse_args()

    MusicHandler.sd_bin = args.sd_bin
    MusicHandler.model_path = args.model

    server = ReusableTCPServer(("0.0.0.0", args.port), MusicHandler)
    print(f"[music] Listening on 0.0.0.0:{args.port}")
    print(f"[music] SD binary: {args.sd_bin}")
    print(f"[music] Model: {args.model or '(default)'}")
    server.serve_forever()


if __name__ == "__main__":
    main()
