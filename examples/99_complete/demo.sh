#!/usr/bin/bash

echo -e "\x1b[90m==> Building Go binary"
go build -ldflags="-s -w" -o /tmp/demo_server main.go

echo -e "\x1b[90m==> Booting up the server"
/tmp/demo_server &
PID=$!

echo -e "\x1b[97m==> Serving a static dir:"
printf '\033[94m' >/tmp/demo
curl -i http://localhost:8081/static/ >>/tmp/demo 2>/dev/null
cat /tmp/demo
echo ""

echo -e "\x1b[97m==> Serving a static file:"
printf '\033[94m' >/tmp/demo
curl -i http://localhost:8081/static/script.js >>/tmp/demo 2>/dev/null
cat /tmp/demo
echo ""

echo -e "\x1b[97m==> Benchmark page:"
printf '\033[94m' >/tmp/demo
curl -i http://localhost:8081/benchmark >>/tmp/demo 2>/dev/null
cat /tmp/demo
echo ""

echo -e "\x1b[90m==> Stopping the server process"
kill $PID

echo -e "\x1b[90m==> Done"
echo -e "\x1b[0m"
