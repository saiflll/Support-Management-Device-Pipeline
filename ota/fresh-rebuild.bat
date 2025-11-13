@echo off
cd /d "d:\iot\suhu ck 3\server\serv-lokal\ota"
echo Stopping and removing old container...
docker-compose down
docker rmi ota-app -f
echo Building fresh image...
docker-compose build --no-cache
echo Starting container...
docker-compose up -d
timeout /t 3
echo.
echo Showing logs (Ctrl+C to exit):
docker-compose logs -f
