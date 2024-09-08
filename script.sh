#!/usr/bin/zsh
source ~/.zshrc

cd "$(dirname "$0")"
# Run the Go program
go run hnconf.go

# Get the current timestamp
current_time=$(date "+%Y-%m-%d %H:%M:%S")

# Commit changes with the timestamp
git add .
git commit -m "build: $current_time"

# Push changes to remote repository
git push
