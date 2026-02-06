#!/bin/bash
set -e

# NATS CLI check
if ! command -v nats &> /dev/null; then
    echo "nats cli could not be found. Please install it."
    exit 1
fi

echo "Setting up NATS streams..."

# Define streams
streams=("user" "discord" "leaderboard" "round" "score" "guild" "club")

for stream in "${streams[@]}"; do
    echo "Creating stream: $stream"
    # Create stream if it doesn't exist. 
    # Subjects are assumed to be $stream.> (wildcard) for simplicity, or specific if needed.
    # --subjects "$stream.>" covers all events starting with the stream name.
    nats stream add "$stream" \
        --subjects "$stream.>" \
        --storage file \
        --retention limits \
        --max-msgs 10000 \
        --discard old \
        --replicas 1 \
        --defaults \
        --no-confirm || echo "Stream $stream might already exist or failed to create."
done

echo "NATS streams setup complete."
