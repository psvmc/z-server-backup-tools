@echo off
setlocal EnableExtensions
cd /d "%~dp0\.."
set "PATH=%USERPROFILE%\go\bin;%PATH%"
echo [ZServerBackup] Dev mode
echo   - Frontend: Vite HMR (edit .vue/.ts/.css, window stays open)
echo   - Backend: rebuild+restart only when *.go changes
wails3 dev -config ./build/config.yml -port 10245
exit /b %ERRORLEVEL%
