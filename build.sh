#!/bin/bash

# Build script for video server

echo "Building video server services..."

# Create output directory
mkdir -p bin

# Build API server
echo "Building API server..."
cd cmd/api-server
go build -o ../../bin/api-server .
if [ $? -ne 0 ]; then
    echo "Failed to build API server"
    exit 1
fi
cd ../..

# Build scheduler
echo "Building scheduler..."
cd cmd/scheduler
go build -o ../../bin/scheduler .
if [ $? -ne 0 ]; then
    echo "Failed to build scheduler"
    exit 1
fi
cd ../..

# Build worker
echo "Building worker..."
cd cmd/worker
go build -o ../../bin/worker .
if [ $? -ne 0 ]; then
    echo "Failed to build worker"
    exit 1
fi
cd ../..

echo "Build completed successfully!"
echo "Binaries are located in the 'bin' directory:"
ls -la bin/