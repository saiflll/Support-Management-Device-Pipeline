@echo off
cd /d "%~dp0"
cd ..
python -u usb_scanner.py >> deployment\gateway_output.log 2>&1
