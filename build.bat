@echo off
echo Building Distributed Social Network for Windows...
echo.
echo This application will open in your default web browser
echo No additional dependencies required for Windows!
echo.

go build -o distributed-app.exe cmd/distributed-app/main.go

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ✅ Build successful! 
    echo.
    echo To run the application:
    echo   distributed-app.exe
    echo.
    echo The application will automatically open in your browser.
) else (
    echo.
    echo ❌ Build failed. Please check for Go installation and dependencies.
)

pause