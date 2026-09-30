#!/bin/bash

# Start script for video server services

echo "Starting video server services..."

# Create necessary directories
mkdir -p storage/videos storage/temp

# Start API server in background
echo "Starting API server..."
./bin/api-server &
API_PID=$!

# Start scheduler in background
echo "Starting scheduler..."
./bin/scheduler &
SCHEDULER_PID=$!

# Start transcode worker in background
echo "Starting transcode worker..."
./bin/transcode-worker &
TRANSCODE_WORKER_PID=$!

echo "Services started successfully!"
echo "API Server PID: $API_PID"
echo "Scheduler PID: $SCHEDULER_PID" 
echo "Transcode Worker PID: $TRANSCODE_WORKER_PID"

# Function to stop services
stop_services() {
    echo "Stopping services..."
    kill $API_PID $SCHEDULER_PID $TRANSCODE_WORKER_PID
    wait $API_PID $SCHEDULER_PID $TRANSCODE_WORKER_PID
    echo "Services stopped."
}

# Trap Ctrl+C to stop services
trap stop_services INT TERM

# Wait for all processes
wait
