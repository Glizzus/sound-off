#!/bin/bash

BLUE='\033[34m'
GREEN='\033[32m'
RESET='\033[0m'

prefix() {
  local label="$1"
  local color="$2"
  while IFS= read -r line; do
    echo -e "${color}${label}${RESET} ${line}"
  done
}

go run ./cmd/bot 2>&1 | prefix "[BOT]" "$BLUE" &
go run ./cmd/worker 2>&1 | prefix "[WORKER]" "$GREEN" &

wait
