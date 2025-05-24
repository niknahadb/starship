#!/bin/bash

echo "Building Starship game for web..."

# Set environment for WebAssembly
export GOOS=js
export GOARCH=wasm

# Build the game
echo "Compiling to WebAssembly..."
go build -o main.wasm main.go

# Find and copy the wasm_exec.js file from Go installation
echo "Copying wasm_exec.js..."

# Try different possible locations for wasm_exec.js
WASM_EXEC_LOCATIONS=(
    "$(go env GOROOT)/misc/wasm/wasm_exec.js"
    "/usr/local/go/misc/wasm/wasm_exec.js"
    "/opt/homebrew/opt/go/libexec/misc/wasm/wasm_exec.js"
    "/usr/lib/go/misc/wasm/wasm_exec.js"
)

FOUND=false
for location in "${WASM_EXEC_LOCATIONS[@]}"; do
    if [ -f "$location" ]; then
        cp "$location" .
        echo "Found wasm_exec.js at: $location"
        FOUND=true
        break
    fi
done

if [ "$FOUND" = false ]; then
    echo "Warning: Could not find wasm_exec.js in standard locations."
    echo "Downloading from Go repository..."
    curl -s -o wasm_exec.js https://raw.githubusercontent.com/golang/go/master/misc/wasm/wasm_exec.js
    if [ $? -eq 0 ]; then
        echo "Successfully downloaded wasm_exec.js"
    else
        echo "Failed to download wasm_exec.js. Please check your internet connection."
        exit 1
    fi
fi

echo "Build complete! Files generated:"
echo "- main.wasm (the compiled game)"
echo "- wasm_exec.js (WebAssembly runtime)"
echo "- index.html (game webpage)"
echo ""
echo "To test locally:"
echo "  python3 -m http.server 8080"
echo "  # or"
echo "  npx serve ."
echo ""
echo "Then open http://localhost:8080 in your browser" 