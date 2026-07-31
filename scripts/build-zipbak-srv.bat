@echo off
setlocal EnableExtensions
cd /d "%~dp0\.."
echo [ZServerBackup] Building zipbak-srv (Windows + Linux)...
wails3 task build:zipbak-srv-all
if errorlevel 1 exit /b 1
echo Output: dist\zipbak-srv.exe  ^(remote Windows^)
echo         dist\zipbak-srv      ^(remote Linux^)
exit /b 0
