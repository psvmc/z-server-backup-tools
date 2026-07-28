@echo off
setlocal EnableExtensions
cd /d "%~dp0\.."
set "PATH=%USERPROFILE%\go\bin;%PATH%"
echo [ZServerBackup] Dev mode: rebuild frontend/dist on save, then restart app
echo Edit frontend/src or backend *.go - wait for rebuild in this window
wails3 dev -config ./build/config.yml -port 10245
exit /b %ERRORLEVEL%
