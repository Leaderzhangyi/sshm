@echo off
chcp 65001 >nul
set CGO_ENABLED=0

echo.
echo ==============================
echo  Building Windows 386
echo ==============================
set GOOS=windows
set GOARCH=386
go build -ldflags="-s -w" -o sshm_windows_386.exe ./release

echo.
echo ==============================
echo  Building Windows amd64
echo ==============================
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o sshm_windows_amd64.exe ./release

echo.
echo ==============================
echo  Building Linux amd64
echo ==============================
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o sshm_linux_amd64 ./release

@REM echo.
@REM echo ==============================
@REM echo  Building Mac Intel amd64
@REM echo ==============================
@REM set GOOS=darwin
@REM set GOARCH=amd64
@REM go build -ldflags="-s -w" -o sshm_mac_intel .

@REM echo.
@REM echo ==============================
@REM echo  Building Mac M1/M2 arm64
@REM echo ==============================
@REM set GOOS=darwin
@REM set GOARCH=arm64
@REM go build -ldflags="-s -w" -o sshm_mac_m1 .

echo.
echo ==============================
echo  All build completed!
echo ==============================
echo  sshm_windows_386.exe      Win32
echo  sshm_windows_amd64.exe    Win64
echo  sshm_linux_amd64          Linux64
@REM echo  sshm_mac_intel            Mac Intel
@REM echo  sshm_mac_m1               Mac M1/M2
echo ==============================
echo.
pause