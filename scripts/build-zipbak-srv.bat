@echo off
setlocal EnableExtensions
cd /d "%~dp0\.."
echo [ZServerBackup] Building zipbak-srv.exe (remote Windows server)...
wails3 task build:zipbak-srv
if errorlevel 1 exit /b 1
echo Output: dist\zipbak-srv.exe
exit /b 0
