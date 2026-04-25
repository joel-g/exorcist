#!/bin/bash

# Simple helper script to run Exorcist locally
# Usage: ./run.sh

# Check if SIGNAL_ACCOUNT is set
if [ -z "$SIGNAL_ACCOUNT" ]; then
    echo "Error: SIGNAL_ACCOUNT environment variable not set"
    echo "Usage: export SIGNAL_ACCOUNT='+1234567890' && ./run.sh"
    exit 1
fi

# Build if binary doesn't exist
if [ ! -f "./exorcist" ]; then
    echo "Building exorcist..."
    go build -o exorcist ./cmd/exorcist
fi

# Run in tmux if available, otherwise run directly
if command -v tmux &> /dev/null; then
    echo "Starting exorcist in tmux session 'exorcist'..."
    tmux new-session -d -s exorcist "./exorcist"
    echo "Exorcist is running in background."
    echo "To view logs: tmux attach -t exorcist"
    echo "To detach: Ctrl+B, then D"
else
    echo "tmux not found, running in foreground..."
    ./exorcist
fi
