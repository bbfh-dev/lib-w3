#!/usr/bin/bash

project=$1
seconds=$2

if [ -z $project ] || [ -z $seconds ]; then
    echo "Usage: [project dir] [time in seconds]"
    exit 1
fi
echo -e "\x1b[95m==> Running benchmark on $project"

echo -e "\x1b[90m==> Building Go binary"
cd $project
go build -ldflags="-s -w" -o /tmp/benchmark_server main.go

echo -e "\x1b[90m==> Booting up the server"
/tmp/benchmark_server &
PID=$!

echo -e "\x1b[90m==> Will benchmark the following endpoint:"
printf '\033[34m' >/tmp/benchmark_sample
curl -i http://localhost:8081/benchmark >>/tmp/benchmark_sample 2>/dev/null
cat /tmp/benchmark_sample
echo ""

echo -e "\x1b[90m==> Performing the benchmark"
printf '\x1b[93m'
wrk -t4 -c400 -d${seconds}s http://localhost:8081/benchmark

echo -e "\x1b[90m==> Stopping the server process"
kill $PID

echo -e "\x1b[90m==> Done"
echo -e "\x1b[0m"
