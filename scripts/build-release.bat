@echo off
setlocal EnableDelayedExpansion
cd /d "%~dp0\.."

for /f "delims=" %%G in ('go env GOPATH 2^>nul') do set "GOPATH=%%G"
if defined GOPATH set "PATH=!GOPATH!\bin;!PATH!"

where wails3 >nul 2>&1
if %errorlevel% neq 0 (
    echo wails3 CLI not found. Install: go install github.com/wailsapp/wails/v3/cmd/wails3@latest
    exit /b 1
)

where makensis >nul 2>&1
if %errorlevel% neq 0 (
    if exist "%ProgramFiles(x86)%\NSIS\makensis.exe" (
        set "PATH=%ProgramFiles(x86)%\NSIS;!PATH!"
    ) else if exist "%ProgramFiles%\NSIS\makensis.exe" (
        set "PATH=%ProgramFiles%\NSIS;!PATH!"
    )
)
where makensis >nul 2>&1
if %errorlevel% neq 0 (
    echo makensis not found. Install NSIS or: winget install NSIS.NSIS
    exit /b 1
)

echo [ZServerBackup] Building release...
wails3 task build
if errorlevel 1 exit /b 1
echo [ZServerBackup] Packaging NSIS installer...
set INSTALL_SCOPE=user
wails3 task package
if errorlevel 1 exit /b 1
echo [ZServerBackup] Building zipbak-srv for remote servers...
wails3 task build:zipbak-srv-all
if errorlevel 1 exit /b 1
echo [ZServerBackup] Done. Output: dist\
echo   Client: ZServerBackup.exe + *-installer.exe
echo   Server: zipbak-srv.exe  ^(copy to remote Windows app dir^)
echo           zipbak-srv      ^(copy to remote Linux app dir, chmod +x^)
dir /b dist\*.exe 2>nul
exit /b 0
