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

# Start worker in background
echo "Starting worker..."
./bin/worker &
WORKER_PID=$!

echo "Services started successfully!"
echo "API Server PID: $API_PID"
echo "Scheduler PID: $SCHEDULER_PID" 
echo "Worker PID: $WORKER_PID"

# Function to stop services
stop_services() {
    echo "Stopping services..."
    kill $API_PID $SCHEDULER_PID $WORKER_PID
    wait $API_PID $SCHEDULER_PID $WORKER_PID
    echo "Services stopped."
}

# Trap Ctrl+C to stop services
trap stop_services INT TERM

# Wait for all processes
wait