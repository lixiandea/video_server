#!/bin/bash

# Cleanup script for video server

echo "Cleaning up video server..."

# Stop any running processes
pkill -f api-server 2>/dev/null
pkill -f scheduler 2>/dev/null
pkill -f worker 2>/dev/null

# Remove binaries
rm -f bin/*

# Remove temporary storage (keep directory structure)
find storage/temp -type f -delete 2>/dev/null

echo "Cleanup completed."