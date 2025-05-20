#!/bin/sh

# Default command if none provided
if [ $# -eq 0 ]; then
  exec /app/auth_server
  exit 0
fi

# Special handling for generate_token with JSON
if [ "$1" = "generate_token" ]; then
  shift # Remove 'generate_token' from arguments
  exec /app/generate_token "$@"
else
  # For all other commands
  exec /app/auth_server "$@"
fi
