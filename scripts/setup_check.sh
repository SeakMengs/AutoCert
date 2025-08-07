#!/bin/bash
# Check for external dependency: Chromium

printf "🔍 Checking for Chromium dependency...\n"
if command -v chromium &> /dev/null; then
    printf "✅ Chromium is already installed!\n"
else
    if command -v apt &> /dev/null; then
        printf "❌ Chromium is not installed, installing using apt...\n"
        sudo apt install -y chromium
    else
        printf "❌ Chromium is not installed. Please install Chromium using your system's package manager.\n"
    fi
    exit 1
fi
