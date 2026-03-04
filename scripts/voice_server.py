#!/usr/bin/env python3
"""
voice_server.py — Whisper STT + Piper TTS HTTP server for Sovereign Stack
Runs on Brain Net, port 8088. 100% local — no external APIs.

Endpoints:
  POST /transcribe  - Audio blob → text (Whisper STT)
  POST /speak       - Text → base64 WAV (Piper TTS)
  GET  /status      - Check if STT/TTS are available
"""

import argparse
import base64
import json
import os
import subprocess
import tempfile
import time
from http.server import HTTPServer, BaseHTTPRequestHandler

# Defaults — all local paths on Brain Net
DEFAULT_PORT = 8088
DEFAULT_WHISPER_BIN = os.path.expanduser("~/whisper.cpp/build/bin/whisper-cli")
DEFAULT_WHISPER_MODEL = os.path.expanduser("~/whisper.cpp/models/ggml-tiny.en.bin")
DEFAULT_PIPER_BIN = os.path.expanduser("~/piper-env/bin/piper")  # venv on Brain Net
DEFAULT_PIPER_MODEL = os.path.expanduser("~/models/piper/en_US-amy-medium.onnx")


class VoiceHandler(BaseHTTPRequestHandler):
    whisper_bin = DEFAULT_WHISPER_BIN
    whisper_model = DEFAULT_WHISPER_MODEL
    piper_bin = DEFAULT_PIPER_BIN
    piper_model = DEFAULT_PIPER_MODEL

    def do_GET(self):
        if self.path == "/status":
            stt_ok = os.path.exists(self.whisper_bin) and os.path.exists(self.whisper_model)
            tts_ok = self._check_piper()
            self._send_json({
                "stt_online": stt_ok,
                "tts_online": tts_ok,
                "whisper_model": os.path.basename(self.whisper_model),
            })
        else:
            self.send_error(404)

    def do_POST(self):
        if self.path == "/transcribe":
            self._handle_transcribe()
        elif self.path == "/speak":
            self._handle_speak()
        else:
            self.send_error(404)

    def _handle_transcribe(self):
        """Accept audio blob, run Whisper, return text."""
        content_len = int(self.headers.get("Content-Length", 0))
        if content_len == 0:
            self._send_json({"error": "no audio data"}, 400)
            return

        audio_data = self.rfile.read(content_len)
        content_type = self.headers.get("Content-Type", "")

        # Save to temp file — Whisper needs a file on disk
        suffix = ".wav"
        if "webm" in content_type:
            suffix = ".webm"
        elif "ogg" in content_type:
            suffix = ".ogg"

        with tempfile.NamedTemporaryFile(suffix=suffix, delete=False) as tmp:
            tmp.write(audio_data)
            audio_path = tmp.name

        # If not WAV, convert with ffmpeg
        wav_path = audio_path
        if suffix != ".wav":
            wav_path = audio_path.replace(suffix, ".wav")
            try:
                subprocess.run(
                    ["ffmpeg", "-y", "-i", audio_path, "-ar", "16000", "-ac", "1", "-f", "wav", wav_path],
                    capture_output=True, timeout=10,
                )
            except Exception as e:
                self._cleanup(audio_path, wav_path)
                self._send_json({"error": f"ffmpeg conversion failed: {e}"}, 500)
                return

        start_time = time.time()
        try:
            result = subprocess.run(
                [self.whisper_bin, "-m", self.whisper_model, "-f", wav_path,
                 "--no-timestamps", "--threads", "4", "--language", "en"],
                capture_output=True, text=True, timeout=30,
            )
            elapsed_ms = int((time.time() - start_time) * 1000)

            if result.returncode != 0:
                self._cleanup(audio_path, wav_path)
                self._send_json({"error": f"whisper failed: {result.stderr[:300]}", "time_ms": elapsed_ms}, 500)
                return

            # Whisper outputs text to stdout
            text = result.stdout.strip()
            self._cleanup(audio_path, wav_path)
            self._send_json({"text": text, "time_ms": elapsed_ms})

        except subprocess.TimeoutExpired:
            self._cleanup(audio_path, wav_path)
            self._send_json({"error": "transcription timed out"}, 504)

    def _handle_speak(self):
        """Accept text, run Piper TTS, return base64 WAV."""
        content_len = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(content_len)
        try:
            req = json.loads(body)
        except json.JSONDecodeError:
            self._send_json({"error": "invalid JSON"}, 400)
            return

        text = req.get("text", "")
        if not text:
            self._send_json({"error": "text required"}, 400)
            return

        with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
            out_path = tmp.name

        start_time = time.time()
        try:
            # Pipe text into piper
            result = subprocess.run(
                [self.piper_bin, "-m", self.piper_model, "--output_file", out_path],
                input=text, capture_output=True, text=True, timeout=30,
            )
            elapsed_ms = int((time.time() - start_time) * 1000)

            if result.returncode != 0 or not os.path.exists(out_path):
                self._send_json({"error": f"piper failed: {result.stderr[:300]}", "time_ms": elapsed_ms}, 500)
                return

            with open(out_path, "rb") as f:
                wav_data = f.read()
            b64 = base64.b64encode(wav_data).decode("utf-8")

            self._send_json({
                "audio": f"data:audio/wav;base64,{b64}",
                "time_ms": elapsed_ms,
            })
        except subprocess.TimeoutExpired:
            self._send_json({"error": "TTS timed out"}, 504)
        finally:
            try:
                os.unlink(out_path)
            except OSError:
                pass

    def _check_piper(self):
        try:
            subprocess.run([self.piper_bin, "--help"], capture_output=True, timeout=3)
            return os.path.exists(self.piper_model)
        except Exception:
            return False

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
        print(f"[voice] {args[0]} {args[1]}")


import socketserver


class ReusableTCPServer(socketserver.TCPServer):
    """TCP server with SO_REUSEADDR to avoid 'Address already in use'."""
    allow_reuse_address = True
    allow_reuse_port = True


def main():
    parser = argparse.ArgumentParser(description="Voice Server — Whisper STT + Piper TTS")
    parser.add_argument("--port", type=int, default=DEFAULT_PORT)
    parser.add_argument("--whisper-bin", default=DEFAULT_WHISPER_BIN)
    parser.add_argument("--whisper-model", default=DEFAULT_WHISPER_MODEL)
    parser.add_argument("--piper-bin", default=DEFAULT_PIPER_BIN)
    parser.add_argument("--piper-model", default=DEFAULT_PIPER_MODEL)
    args = parser.parse_args()

    VoiceHandler.whisper_bin = args.whisper_bin
    VoiceHandler.whisper_model = args.whisper_model
    VoiceHandler.piper_bin = args.piper_bin
    VoiceHandler.piper_model = args.piper_model

    server = ReusableTCPServer(("0.0.0.0", args.port), VoiceHandler)
    print(f"[voice] Listening on 0.0.0.0:{args.port}")
    print(f"[voice] Whisper: {args.whisper_bin}")
    print(f"[voice] Model:   {args.whisper_model}")
    print(f"[voice] Piper:   {args.piper_bin}")
    server.serve_forever()


if __name__ == "__main__":
    main()
