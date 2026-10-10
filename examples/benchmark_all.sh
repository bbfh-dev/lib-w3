#!/usr/bin/bash

seconds=$1

if [ -z $seconds ]; then
    echo "Usage: [time in seconds]"
    exit 1
fi

for entry in ./*; do
    if [ -d "$entry" ]; then
        ./benchmark.sh "$entry" "$seconds"
    fi
done
