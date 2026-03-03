@echo off
REM setup_sd_server.bat — Sets up stable-diffusion.cpp on Windows mini PC
REM Run this script once to download sd.exe and the SD Turbo model
echo ============================================
echo  Sovereign Stack - Image Gen Setup (Mini PC)
echo ============================================
echo.

set SD_VERSION=master-e6186e3
set SD_URL=https://github.com/leejet/stable-diffusion.cpp/releases/download/%SD_VERSION%/sd-master-e6186e3-bin-win-avx2-x64.zip
set MODEL_URL=https://huggingface.co/stabilityai/sd-turbo/resolve/main/sd_turbo.safetensors

REM Create working directory
if not exist "sd_server" mkdir sd_server
cd sd_server

REM Download sd.cpp binary
echo [1/3] Downloading stable-diffusion.cpp...
if not exist "sd.exe" (
    curl -L -o sd.zip "%SD_URL%"
    tar -xf sd.zip
    del sd.zip
    echo      Done.
) else (
    echo      Already downloaded.
)

REM Download SD Turbo model
echo [2/3] Downloading SD Turbo model (~2GB)...
if not exist "sd_turbo.safetensors" (
    curl -L -o sd_turbo.safetensors "%MODEL_URL%"
    echo      Done.
) else (
    echo      Already downloaded.
)

REM Convert to GGUF (quantized)
echo [3/3] Converting model to quantized GGUF...
if not exist "sd-turbo-q8_0.gguf" (
    sd.exe --convert sd_turbo.safetensors -o sd-turbo-q8_0.gguf --type q8_0
    echo      Done.
) else (
    echo      Already converted.
)

REM Copy sd_server.py
echo.
echo Copying sd_server.py...
copy /Y "..\scripts\sd_server.py" . >nul 2>&1 || echo Note: Copy sd_server.py manually from sovereign-stack/scripts/

echo.
echo ============================================
echo  Setup complete!
echo  Start the server with:
echo    python sd_server.py --sd-path ./sd.exe --model ./sd-turbo-q8_0.gguf
echo  Server will listen on http://0.0.0.0:8090
echo ============================================
pause
