@echo off
setlocal EnableDelayedExpansion
cd /d "%~dp0.."

set "GITHUB_REPO=psvmc/z-server-backup-tools"
set "TAG="

:parse_args
if "%~1"=="" goto args_done
if not defined TAG set "TAG=%~1"
shift
goto parse_args
:args_done

if not defined TAG (
    for /f "tokens=2 delims==" %%a in ('findstr /C:"AppVersion" version.go') do set "VER=%%a"
    set "VER=!VER:"=!"
    for /f "tokens=* delims= " %%a in ("!VER!") do set "VER=%%a"
    set "TAG=!VER!"
)

where gh >nul 2>&1 || (
    echo [error] Install GitHub CLI and run: gh auth login
    exit /b 1
)

echo Triggering release-all for tag %TAG% on %GITHUB_REPO%...
gh workflow run release-all.yml -R "%GITHUB_REPO%" -f "tag=%TAG%"
exit /b %ERRORLEVEL%
