#!/bin/bash
set -e

# Default values
CLIENT_ID="test-client"
SECRET="dev-secret"

# Parse arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        -client) CLIENT_ID="$2"; shift ;;
        -secret) SECRET="$2"; shift ;;
        *) echo "Unknown parameter passed: $1"; exit 1 ;;
    esac
    shift
done

echo "Generating API Key for Client: $CLIENT_ID (Secret: $SECRET)"
make apikey CLIENT="$CLIENT_ID" SECRET="$SECRET"
