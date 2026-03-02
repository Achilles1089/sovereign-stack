#!/data/data/com.termux/files/usr/bin/python3
"""
sysinfo_server.py - Phone hardware info HTTP server for Sovereign Stack
Runs alongside llama-server on Android/Termux, exposing hardware info at :8086

Usage:  python3 sysinfo_server.py &
Stop:   kill $(cat /tmp/sysinfo.pid)
"""

import http.server
import json
import subprocess
import os

PORT = int(os.environ.get("SYSINFO_PORT", "8086"))

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
    # Phone model
    phone_model = get_prop("ro.product.model") or "Unknown"

    # SoC
    soc = get_prop("ro.soc.model") or get_prop("ro.hardware.chipname") or get_prop("ro.hardware") or "Unknown"

    # Android version
    android_ver = get_prop("ro.build.version.release") or "Unknown"

    # RAM
    ram_total_mb = 0
    ram_avail_mb = 0
    meminfo = read_file("/proc/meminfo")
    for line in meminfo.split("\n"):
        if line.startswith("MemTotal:"):
            ram_total_mb = int(line.split()[1]) // 1024
        elif line.startswith("MemAvailable:"):
            ram_avail_mb = int(line.split()[1]) // 1024

    # CPU cores
    try:
        cpu_cores = os.cpu_count() or 0
    except Exception:
        cpu_cores = 0

    # CPU max frequency
    cpu_freq = 0
    freq_str = read_file("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq").strip()
    if freq_str:
        cpu_freq = int(freq_str) // 1000

    # Storage
    storage_total_gb = 0
    storage_free_gb = 0
    try:
        st = os.statvfs("/data")
        storage_total_gb = (st.f_blocks * st.f_frsize) // (1024 ** 3)
        storage_free_gb = (st.f_bavail * st.f_frsize) // (1024 ** 3)
    except Exception:
        pass

    # Battery
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

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        info = get_system_info()
        body = json.dumps(info).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, format, *args):
        pass  # Suppress request logs

if __name__ == "__main__":
    # Save PID for easy cleanup
    tmpdir = os.environ.get("TMPDIR", os.environ.get("HOME", "/data/data/com.termux/files/home"))
    pidfile = os.path.join(tmpdir, "sysinfo.pid")
    with open(pidfile, "w") as f:
        f.write(str(os.getpid()))

    print(f"  [SYSINFO] Phone hardware endpoint: http://0.0.0.0:{PORT}")
    server = http.server.HTTPServer(("0.0.0.0", PORT), Handler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n  [SYSINFO] Stopped")
        server.shutdown()
