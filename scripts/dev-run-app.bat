@echo off
setlocal
cd /d "%~dp0\.."
REM 保留 wails3 注入的 FRONTEND_DEVSERVER_URL，走 Vite 热更新
"%~dp0..\dist\ZServerBackup.exe"
exit /b %ERRORLEVEL%
