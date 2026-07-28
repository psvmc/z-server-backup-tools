@echo off
setlocal
cd /d "%~dp0\.."
set "FRONTEND_DEVSERVER_URL="
"%~dp0..\dist\ZServerBackup.exe"
exit /b %ERRORLEVEL%
