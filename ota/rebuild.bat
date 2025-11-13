@echo off
cd /d "d:\iot\suhu ck 3\server\serv-lokal\ota"
echo Stopping container...
docker-compose down
echo Building new image...
docker-compose build
echo Starting container...
docker-compose up -d
echo Done! Check logs with: docker-compose logs -f
pause
