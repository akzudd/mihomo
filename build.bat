@echo off
setlocal

for /f "delims=" %%i in ('git describe --tags --always --dirty 2^>nul') do set "VERSION=%%i"
if not defined VERSION (
    echo Failed to get git version.
    exit /b 1
)

for /f "delims=" %%i in ('powershell -NoProfile -Command "Get-Date -Format 'yyyy-MM-ddTHH:mm:sszzz'"') do set "BUILD_TIME=%%i"
if not defined BUILD_TIME (
    echo Failed to get build time.
    exit /b 1
)

set "CGO_ENABLED=0"
set "GOOS=windows"
set "GOARCH=amd64"
set "GOAMD64=v3"

go build -tags with_gvisor -trimpath -ldflags "-X github.com/metacubex/mihomo/constant.Version=%VERSION% -X github.com/metacubex/mihomo/constant.BuildTime=%BUILD_TIME% -w -s -buildid=" -o mihomo.exe
if errorlevel 1 exit /b %errorlevel%

echo Built mihomo.exe
echo Version: %VERSION%
echo BuildTime: %BUILD_TIME%
