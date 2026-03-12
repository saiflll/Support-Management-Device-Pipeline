@echo off
REM Quick start script untuk IoT Device Simulator

echo ============================================================
echo IoT Device Simulator - Quick Start
echo ============================================================
echo.

REM Check if Python is installed
python --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Python tidak ditemukan!
    echo Silakan install Python terlebih dahulu: https://www.python.org/downloads/
    pause
    exit /b 1
)

echo [INFO] Python terdeteksi
echo.

REM Check if paho-mqtt is installed
python -c "import paho.mqtt.client" >nul 2>&1
if errorlevel 1 (
    echo [INFO] Installing dependencies...
    pip install paho-mqtt
    echo.
)

echo [INFO] Dependencies OK
echo.

REM Menu
echo Pilih mode simulasi:
echo.
echo 1. Quick Test (3 devices, 5 detik interval)
echo 2. TEMP Devices (5 devices, model TEMP)
echo 3. MDCW Devices (2 devices, model MDCW)
echo 4. Offline Test (1 device, 10 detik interval)
echo 5. Custom (input manual)
echo.

set /p choice="Pilihan (1-5): "

if "%choice%"=="1" (
    echo.
    echo [RUN] Quick Test - 3 devices, 5s interval
    python simulator.py --devices 3 --interval 5
) else if "%choice%"=="2" (
    echo.
    echo [RUN] TEMP Devices - 5 devices
    python simulator.py --devices 5 --model TEMP --interval 3
) else if "%choice%"=="3" (
    echo.
    echo [RUN] MDCW Devices - 2 devices
    python simulator.py --devices 2 --model MDCW --interval 5
) else if "%choice%"=="4" (
    echo.
    echo [RUN] Offline Test - 1 device, 10s interval
    echo Stop dengan Ctrl+C untuk test offline detection
    python simulator.py --devices 1 --interval 10
) else if "%choice%"=="5" (
    echo.
    set /p broker="MQTT Broker (default: localhost): "
    set /p devices="Jumlah devices (default: 3): "
    set /p interval="Interval detik (default: 5): "
    set /p model="Model TEMP/MDCW (default: TEMP): "
    
    if "%broker%"=="" set broker=localhost
    if "%devices%"=="" set devices=3
    if "%interval%"=="" set interval=5
    if "%model%"=="" set model=TEMP
    
    echo.
    echo [RUN] Custom: %devices% devices, %interval%s interval, model %model%
    python simulator.py --broker %broker% --devices %devices% --interval %interval% --model %model%
) else (
    echo [ERROR] Pilihan tidak valid!
    pause
    exit /b 1
)

echo.
echo ============================================================
echo Simulasi selesai
echo ============================================================
pause
