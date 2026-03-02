#!/data/data/com.termux/files/usr/bin/python3
"""
sysinfo_server.py - Phone hardware + model management for Sovereign Stack
Runs alongside llama-server on Android/Termux, port 8086

Endpoints:
  GET  /           - Phone hardware info (JSON)
  GET  /models     - List available GGUF models on disk
  POST /switch     - Switch active model (restarts llama-server)
"""

import http.server
import json
import subprocess
import os
import glob
import signal
import time
import threading

PORT = int(os.environ.get("SYSINFO_PORT", "8086"))
MODELS_DIR = os.environ.get("MODELS_DIR", os.path.expanduser("~/models"))
LLAMA_SERVER = os.environ.get("LLAMA_SERVER", os.path.expanduser("~/llama.cpp/bld/bin/llama-server"))
LLAMA_PORT = int(os.environ.get("LLAMA_PORT", "8085"))

# Big core IDs for taskset (A75 cores on T616)
BIG_CORES = os.environ.get("BIG_CORES", "6,7")

def get_prop(prop):
    try:
        return subprocess.check_output(["getprop", prop], text=True, stderr=subprocess.DEVNULL).strip()
    except Exception:
        return ""

def read_file(path):
    try:
        with open(path) as f:
            return f.read()
    except Exception:
        return ""

def get_system_info():
    phone_model = get_prop("ro.product.model") or "Unknown"
    soc = get_prop("ro.soc.model") or get_prop("ro.hardware.chipname") or get_prop("ro.hardware") or "Unknown"
    android_ver = get_prop("ro.build.version.release") or "Unknown"

    ram_total_mb = 0
    ram_avail_mb = 0
    meminfo = read_file("/proc/meminfo")
    for line in meminfo.split("\n"):
        if line.startswith("MemTotal:"):
            ram_total_mb = int(line.split()[1]) // 1024
        elif line.startswith("MemAvailable:"):
            ram_avail_mb = int(line.split()[1]) // 1024

    cpu_cores = os.cpu_count() or 0
    cpu_freq = 0
    freq_str = read_file("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq").strip()
    if freq_str:
        cpu_freq = int(freq_str) // 1000

    storage_total_gb = 0
    storage_free_gb = 0
    try:
        st = os.statvfs("/data")
        storage_total_gb = (st.f_blocks * st.f_frsize) // (1024 ** 3)
        storage_free_gb = (st.f_bavail * st.f_frsize) // (1024 ** 3)
    except Exception:
        pass

    battery = -1
    bat_str = read_file("/sys/class/power_supply/battery/capacity").strip()
    if bat_str:
        battery = int(bat_str)

    return {
        "phone_model": phone_model,
        "soc": soc,
        "android_version": android_ver,
        "cpu_cores": cpu_cores,
        "cpu_freq_mhz": cpu_freq,
        "ram_total_mb": ram_total_mb,
        "ram_available_mb": ram_avail_mb,
        "storage_total_gb": storage_total_gb,
        "storage_free_gb": storage_free_gb,
        "battery_pct": battery,
    }

def list_models():
    """List all .gguf files in the models directory with sizes."""
    models = []
    for f in sorted(glob.glob(os.path.join(MODELS_DIR, "*.gguf"))):
        name = os.path.basename(f)
        size_mb = os.path.getsize(f) // (1024 * 1024)
        models.append({"name": name, "path": f, "size_mb": size_mb})
    return models

def get_active_model():
    """Find the currently running llama-server and extract its model path."""
    try:
        out = subprocess.check_output(
            ["ps", "-ef"], text=True, stderr=subprocess.DEVNULL
        )
        for line in out.split("\n"):
            if "llama-server" in line and "-m " in line:
                parts = line.split("-m ")
                if len(parts) > 1:
                    model_path = parts[1].split()[0]
                    return os.path.basename(model_path)
    except Exception:
        pass
    return None

def switch_model(model_name):
    """Kill current llama-server and restart with the requested model."""
    model_path = os.path.join(MODELS_DIR, model_name)
    if not os.path.exists(model_path):
        return {"ok": False, "error": f"Model not found: {model_name}"}

    # Kill existing llama-server
    try:
        subprocess.run(["killall", "llama-server"], stderr=subprocess.DEVNULL)
        time.sleep(2)
    except Exception:
        pass

    # Optimize flags based on model size — T616 has 6GB RAM, ~4.5GB usable
    size_mb = os.path.getsize(model_path) // (1024 * 1024)
    threads = 2  # 2 big A75 cores

    # Dynamic context: with -fa + q8_0 KV cache, memory is minimal (~28MB per 1024 tokens for 7B)
    if size_mb > 4000:      # 7B Q4 (4.3GB) — leaves ~200MB, fits 2048 ctx
        ctx_size = 2048
    elif size_mb > 3000:    # 7B Q3 (3.5GB), 2.9B Q8 (3.0GB)
        ctx_size = 2048
    elif size_mb > 1500:    # 3B models (~2GB)
        ctx_size = 4096
    else:                   # 1.5B and smaller
        ctx_size = 4096

    # Launch new server pinned to big cores
    cmd = [
        "taskset", "-c", BIG_CORES,
        LLAMA_SERVER,
        "-m", model_path,
        "--host", "0.0.0.0",
        "--port", str(LLAMA_PORT),
        "--threads", str(threads),
        "--parallel", "1",
        "--ctx-size", str(ctx_size),
        "--batch-size", "256",
        "--ubatch-size", "128",
        "-fa",                   # Flash attention — reduces KV cache memory
    ]

    # KV cache quantization for models > 3GB — saves ~50% KV memory
    if size_mb > 3000:
        cmd.extend(["--cache-type-k", "q8_0", "--cache-type-v", "q8_0"])

    try:
        proc = subprocess.Popen(
            cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
            start_new_session=True
        )
        # Wait for startup
        time.sleep(4)
        if proc.poll() is not None:
            return {"ok": False, "error": "llama-server exited immediately"}
        return {"ok": True, "model": model_name, "pid": proc.pid}
    except Exception as e:
        return {"ok": False, "error": str(e)}


class Handler(http.server.BaseHTTPRequestHandler):
    def _send_json(self, data, status=200):
        body = json.dumps(data).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_OPTIONS(self):
        self._send_json({})

    def do_GET(self):
        if self.path == "/models":
            active = get_active_model()
            models = list_models()
            self._send_json({"models": models, "active": active})
        else:
            self._send_json(get_system_info())

    def do_POST(self):
        if self.path == "/switch":
            length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(length).decode() if length > 0 else "{}"
            try:
                data = json.loads(body)
            except Exception:
                self._send_json({"ok": False, "error": "Invalid JSON"}, 400)
                return

            model_name = data.get("model", "")
            if not model_name:
                self._send_json({"ok": False, "error": "Missing 'model' field"}, 400)
                return

            # Run switch in background to avoid blocking
            def do_switch():
                result = switch_model(model_name)
                # Store result for status check
                Handler._last_switch = result

            Handler._last_switch = {"ok": True, "model": model_name, "status": "switching"}
            t = threading.Thread(target=do_switch)
            t.start()
            self._send_json({"ok": True, "model": model_name, "status": "switching"})
        else:
            self._send_json({"error": "Not found"}, 404)

    def log_message(self, format, *args):
        pass

if __name__ == "__main__":
    tmpdir = os.environ.get("TMPDIR", os.environ.get("HOME", "/data/data/com.termux/files/home"))
    pidfile = os.path.join(tmpdir, "sysinfo.pid")
    with open(pidfile, "w") as f:
        f.write(str(os.getpid()))

    print(f"  [SYSINFO] Phone hardware + model management: http://0.0.0.0:{PORT}")
    print(f"  [SYSINFO] Models dir: {MODELS_DIR}")
    print(f"  [SYSINFO] Big cores: {BIG_CORES}")
    server = http.server.HTTPServer(("0.0.0.0", PORT), Handler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n  [SYSINFO] Stopped")
        server.shutdown()
