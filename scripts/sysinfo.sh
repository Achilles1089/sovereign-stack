#!/data/data/com.termux/files/usr/bin/bash
# sysinfo.sh — Lightweight system info HTTP server for Sovereign Stack
# Runs alongside llama-server on Android/Termux, exposing hardware info at :8086
#
# Usage:  ./sysinfo.sh &
# Stop:   kill $(cat /tmp/sysinfo.pid)

PORT=${SYSINFO_PORT:-8086}

get_system_info() {
    # Phone model
    phone_model=$(getprop ro.product.model 2>/dev/null || echo "Unknown")

    # SoC / chipset
    soc=$(getprop ro.soc.model 2>/dev/null)
    if [ -z "$soc" ]; then
        soc=$(getprop ro.hardware.chipname 2>/dev/null)
    fi
    if [ -z "$soc" ]; then
        soc=$(getprop ro.hardware 2>/dev/null || echo "Unknown")
    fi

    # Android version
    android_ver=$(getprop ro.build.version.release 2>/dev/null || echo "Unknown")

    # RAM from /proc/meminfo (total in MB)
    ram_kb=$(grep MemTotal /proc/meminfo 2>/dev/null | awk '{print $2}')
    ram_mb=$((ram_kb / 1024))

    # Available RAM
    ram_avail_kb=$(grep MemAvailable /proc/meminfo 2>/dev/null | awk '{print $2}')
    ram_avail_mb=$((ram_avail_kb / 1024))

    # Storage free on /data (where Termux lives)
    storage_info=$(df /data 2>/dev/null | tail -1)
    storage_total_kb=$(echo "$storage_info" | awk '{print $2}')
    storage_free_kb=$(echo "$storage_info" | awk '{print $4}')
    storage_total_gb=$((storage_total_kb / 1048576))
    storage_free_gb=$((storage_free_kb / 1048576))

    # CPU cores
    cpu_cores=$(nproc 2>/dev/null || grep -c processor /proc/cpuinfo 2>/dev/null || echo 0)

    # CPU max frequency (MHz)
    cpu_freq=0
    if [ -f /sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq ]; then
        cpu_freq_khz=$(cat /sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq)
        cpu_freq=$((cpu_freq_khz / 1000))
    fi

    # Battery level (if accessible)
    battery="-1"
    if [ -f /sys/class/power_supply/battery/capacity ]; then
        battery=$(cat /sys/class/power_supply/battery/capacity)
    fi

    # Build JSON
    cat <<EOF
{"phone_model":"${phone_model}","soc":"${soc}","android_version":"${android_ver}","cpu_cores":${cpu_cores},"cpu_freq_mhz":${cpu_freq},"ram_total_mb":${ram_mb},"ram_available_mb":${ram_avail_mb},"storage_total_gb":${storage_total_gb},"storage_free_gb":${storage_free_gb},"battery_pct":${battery}}
EOF
}

echo $$ > /tmp/sysinfo.pid
echo "  [SYSINFO] Phone hardware endpoint: http://0.0.0.0:${PORT}"

# Simple HTTP server using socat or ncat
if command -v socat &>/dev/null; then
    while true; do
        RESPONSE=$(get_system_info)
        HEADERS="HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nAccess-Control-Allow-Origin: *\r\nContent-Length: ${#RESPONSE}\r\nConnection: close\r\n\r\n"
        echo -e "${HEADERS}${RESPONSE}" | socat - TCP-LISTEN:${PORT},reuseaddr 2>/dev/null
    done
elif command -v ncat &>/dev/null; then
    while true; do
        RESPONSE=$(get_system_info)
        HEADERS="HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nAccess-Control-Allow-Origin: *\r\nContent-Length: ${#RESPONSE}\r\nConnection: close\r\n\r\n"
        echo -e "${HEADERS}${RESPONSE}" | ncat -l -p ${PORT} 2>/dev/null
    done
elif command -v python3 &>/dev/null; then
    # Python fallback — proper HTTP server
    python3 -c "
import http.server, json, subprocess, os

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        info = json.loads(subprocess.check_output(['bash', '-c', '''$(cat <<'SCRIPT'
$(grep -o '{.*}' <<< "$(bash $0)")
SCRIPT
)'''], text=True).strip()) if False else {}
        # Just shell out to the bash function
        result = subprocess.run(['bash', '-c', '''
phone_model=\$(getprop ro.product.model 2>/dev/null || echo Unknown)
soc=\$(getprop ro.soc.model 2>/dev/null || getprop ro.hardware.chipname 2>/dev/null || getprop ro.hardware 2>/dev/null || echo Unknown)
android_ver=\$(getprop ro.build.version.release 2>/dev/null || echo Unknown)
ram_kb=\$(grep MemTotal /proc/meminfo 2>/dev/null | awk \"{print \\\$2}\")
ram_mb=\$((ram_kb / 1024))
ram_avail_kb=\$(grep MemAvailable /proc/meminfo 2>/dev/null | awk \"{print \\\$2}\")
ram_avail_mb=\$((ram_avail_kb / 1024))
cpu_cores=\$(nproc 2>/dev/null || echo 0)
battery=-1
[ -f /sys/class/power_supply/battery/capacity ] && battery=\$(cat /sys/class/power_supply/battery/capacity)
printf \"{\\\"phone_model\\\":\\\"%s\\\",\\\"soc\\\":\\\"%s\\\",\\\"android_version\\\":\\\"%s\\\",\\\"cpu_cores\\\":%d,\\\"cpu_freq_mhz\\\":0,\\\"ram_total_mb\\\":%d,\\\"ram_available_mb\\\":%d,\\\"storage_total_gb\\\":0,\\\"storage_free_gb\\\":0,\\\"battery_pct\\\":%d}\" \"\$phone_model\" \"\$soc\" \"\$android_ver\" \"\$cpu_cores\" \"\$ram_mb\" \"\$ram_avail_mb\" \"\$battery\"
'''], capture_output=True, text=True)
        body = result.stdout.strip()
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Access-Control-Allow-Origin', '*')
        self.end_headers()
        self.wfile.write(body.encode())
    def log_message(self, *a): pass

http.server.HTTPServer(('0.0.0.0', ${PORT}), Handler).serve_forever()
" &
    wait
else
    echo "  [ERROR] No socat, ncat, or python3 found. Install: pkg install socat"
    exit 1
fi
