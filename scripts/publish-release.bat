@echo off
setlocal EnableDelayedExpansion
cd /d "%~dp0.."

set "GITHUB_REPO=psvmc/z-server-backup-tools"
set "TAG="

:parse_args
if "%~1"=="" goto args_done
if /I "%~1"=="--skip-push" (
    set "SKIP_PUSH=1"
    shift
    goto parse_args
)
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
where git >nul 2>&1 || (
    echo [error] git not found
    exit /b 1
)

echo === Publish release ===
echo Repo: %GITHUB_REPO%
echo Tag:  %TAG%

REM Ensure current commit is on origin/main (workflow builds the tag)
git fetch origin --tags 2>nul
for /f %%h in ('git rev-parse HEAD') do set "HEAD_SHA=%%h"
for /f %%h in ('git rev-parse origin/main 2^>nul') do set "ORIGIN_SHA=%%h"
if defined ORIGIN_SHA if /I not "!HEAD_SHA!"=="!ORIGIN_SHA!" (
    if defined SKIP_PUSH (
        echo [error] HEAD ^(!HEAD_SHA:~0,7!^) differs from origin/main ^(!ORIGIN_SHA:~0,7!^). Push first.
        exit /b 1
    )
    echo Pushing main to origin...
    git push origin HEAD:main
    if errorlevel 1 exit /b 1
)

REM Create tag on HEAD if missing; refuse if tag points elsewhere
git rev-parse -q --verify "refs/tags/%TAG%" >nul 2>&1
if errorlevel 1 (
    echo Creating tag %TAG% at !HEAD_SHA:~0,7! ...
    git tag -a "%TAG%" -m "ZServerBackup %TAG%"
    if errorlevel 1 exit /b 1
) else (
    for /f %%h in ('git rev-list -n 1 "%TAG%"') do set "TAG_SHA=%%h"
    if /I not "!TAG_SHA!"=="!HEAD_SHA!" (
        echo [error] Tag %TAG% already points to !TAG_SHA:~0,7!, but HEAD is !HEAD_SHA:~0,7!.
        echo         Bump version ^(scripts\set-version.bat^) then publish again.
        exit /b 1
    )
    echo Tag %TAG% already exists on HEAD.
)

git ls-remote --exit-code --tags origin "refs/tags/%TAG%" >nul 2>&1
if errorlevel 1 (
    echo Pushing tag %TAG% ...
    git push origin "refs/tags/%TAG%"
    if errorlevel 1 exit /b 1
) else (
    echo Remote tag %TAG% already exists.
)

echo Triggering release-all for tag %TAG% on %GITHUB_REPO%...
gh workflow run release-all.yml -R "%GITHUB_REPO%" -f "tag=%TAG%"
exit /b %ERRORLEVEL%
