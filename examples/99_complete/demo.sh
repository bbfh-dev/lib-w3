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

echo -e "\n\x1b[97m==> GET /benchmark:"
printf '\033[94m' >/tmp/demo
curl -i http://localhost:8081/benchmark >>/tmp/demo 2>/dev/null
cat /tmp/demo
echo ""

echo -e "\n\x1b[97m==> PUT /upload (wrong method):"
printf '\033[94m' >/tmp/demo
curl -i http://localhost:8081/upload >>/tmp/demo 2>/dev/null
cat /tmp/demo
echo ""

echo -e "\n\x1b[97m==> PUT /upload (not authenticated):"
printf '\033[94m' >/tmp/demo
curl -i -X PUT http://localhost:8081/upload >>/tmp/demo 2>/dev/null
cat /tmp/demo
echo ""

echo -e "\n\x1b[97m==> PUT /upload (missing csrf token):"
printf '\033[94m' >/tmp/demo
curl -i -X PUT -H 'Authorization: Bearer abc' http://localhost:8081/upload >>/tmp/demo 2>/dev/null
cat /tmp/demo
echo ""

echo -e "\n\x1b[97m==> PUT /upload:"
printf '\033[94m' >/tmp/demo
curl -i -X PUT -H 'Authorization: Bearer abc' 'http://localhost:8081/upload?csrf_token=hello_world!&key=hi' >>/tmp/demo 2>/dev/null
cat /tmp/demo
echo ""

echo -e "\x1b[90m==> Stopping the server process"
kill $PID

echo -e "\x1b[90m==> Done"
echo -e "\x1b[0m"
