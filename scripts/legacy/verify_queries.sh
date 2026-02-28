#!/bin/bash

QUERIES=(
    "how do i make a folder for my python project"
    "i want to see what files are in this folder"
    "how to delete a file i dont need"
    "how to move a file to another folder"
    "how to download a file from url"
    "how to checkout a git repo"
    "how to edit a config file"
    "my phone storage is full how to check"
    "how to see what is eating my ram"
    "how to unzip a file i downloaded"
)

for q in "${QUERIES[@]}"; do
    echo "---------------------------------------------------"
    echo "Query: $q"
    echo "0" | go run cmd/clipilot/main.go "$q"
    echo ""
done
