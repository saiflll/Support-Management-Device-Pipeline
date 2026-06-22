@echo off
chcp 65001 >nul
title USB Gateway Startup Installer

set "startup_dir=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup"
set "target_vbs=%startup_dir%\RunUSBGateway.vbs"
set "deploy_dir=%~dp0"
set "bat_path=%deploy_dir%run_gateway.bat"

echo ========================================================
echo   USB Gateway Windows Auto-Startup Installer
echo ========================================================
echo.
echo Lokasi folder Startup: %startup_dir%
echo.

if exist "%target_vbs%" (
    del "%target_vbs%"
)
echo Set WshShell = CreateObject("WScript.Shell") > "%target_vbs%"
echo WshShell.Run "cmd.exe /c ""%bat_path%""", 0, false >> "%target_vbs%"

echo [✓] Sukses! Script otomatis telah didaftarkan di folder Startup Windows.
echo [i] Aplikasi akan berjalan otomatis di latar belakang.
echo [i] Lokasi file log monitor: %deploy_dir%gateway_output.log
echo.
echo ========================================================
pause
