#!/bin/bash

echo "Building Distributed Social Network for Unix/Linux/macOS..."
echo
echo "This application will open in your default web browser"
echo "No additional dependencies required!"
echo

go build -o distributed-app cmd/distributed-app/main.go

if [ $? -eq 0 ]; then
    echo
    echo "✅ Build successful!"
    echo
    echo "To run the application:"
    echo "  ./distributed-app"
    echo
    echo "The application will automatically open in your browser."
else
    echo
    echo "❌ Build failed. Please check for Go installation and dependencies."
fi