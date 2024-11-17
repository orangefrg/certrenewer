#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Function to handle cleanup on exit or error
cleanup() {
    echo "Cleaning up environment variables and returning to the original directory..."
    unset GOOS
    unset GOARCH

    if [[ -n "$original_dir" ]]; then
        cd "$original_dir"
    fi
}

# Trap any errors and execute the cleanup function
trap 'echo "An error occurred during the build process."; cleanup; exit 1' ERR
trap cleanup EXIT

# Save the original directory to return to it later
original_dir=$(pwd)

# Function to display error messages and exit
error_exit() {
    echo "$1" >&2
    exit 1
}

# Determine the root directory name
rootDir=$(basename "$(pwd)")

# Define the paths for the 'cmd' and 'bin' directories
cmdDir="./cmd"
binDir="../bin"

# Check if the 'cmd' directory exists
if [[ ! -d "$cmdDir" ]]; then
    error_exit "'cmd/' directory does not exist. Please check the project structure."
fi

# Change to the 'cmd' directory
cd "$cmdDir"

# Create the 'bin' directory if it does not exist
if [[ ! -d "$binDir" ]]; then
    echo "Creating 'bin/' directory..."
    mkdir -p "$binDir"
fi

# Build for Windows
export GOOS=windows
export GOARCH=amd64
echo "Building for Windows..."
go build -o "$binDir/$rootDir.exe"

# Build for Linux
export GOOS=linux
export GOARCH=amd64
echo "Building for Linux..."
go build -o "$binDir/$rootDir"

echo "Build completed successfully. Binaries are in the 'bin/' directory."
