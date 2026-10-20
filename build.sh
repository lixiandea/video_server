#!/bin/bash

set -euo pipefail

echo "Building video server services..."
mkdir -p bin

echo "Building API server..."
go build -o bin/api-server ./cmd/api-server

echo "Building scheduler..."
go build -o bin/scheduler ./cmd/scheduler

echo "Building transcode worker..."
go build -o bin/transcode-worker ./cmd/transcode-worker

echo "Building migrate CLI..."
go build -o bin/migrate ./cmd/migrate

echo "Building admin CLI..."
go build -o bin/admin ./cmd/admin

echo "Build completed successfully!"
ls -la bin/
