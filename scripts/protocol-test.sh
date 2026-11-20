#!/usr/bin/env bash
# Raw-byte protocol tests for the goproxy server.
# Start the server first (make run), then: ./scripts/protocol-test.sh
#
# Each test prints the full response, then pulls out the status line,
# Connection header, Content-Length and body so you can see them at a glance.

set -u
HOST=${HOST:-localhost}
PORT=${PORT:-8080}

bold=$(printf '\033[1m'); dim=$(printf '\033[2m'); off=$(printf '\033[0m')

# send <label> <raw request bytes> [expected note]
send() {
	local label="$1" raw="$2" note="${3:-}"
	printf '\n%s── %s%s' "$bold" "$label" "$off"
	[ -n "$note" ] && printf '  %s(%s)%s' "$dim" "$note" "$off"
	printf '\n'

	local resp
	resp=$(printf '%b' "$raw" | nc -w 3 "$HOST" "$PORT" | tr -d '\r')

	if [ -z "$resp" ]; then
		printf '  %s<no response — server closed without writing>%s\n' "$dim" "$off"
		return
	fi

	# full response, indented
	printf '%s\n' "$resp" | sed 's/^/  │ /'

	# extracted fields
	printf '  └─ status      : %s\n' "$(printf '%s' "$resp" | head -1)"
	printf '     connection  : %s\n' "$(printf '%s\n' "$resp" | grep -i '^Connection:'     | head -1 | cut -d' ' -f2- || echo '(absent)')"
	printf '     content-len : %s\n' "$(printf '%s\n' "$resp" | grep -i '^Content-Length:' | head -1 | cut -d' ' -f2- || echo '(absent)')"
	printf '     body        : %q\n' "$(printf '%s\n' "$resp" | sed '1,/^$/d')"
	printf '     responses   : %s\n' "$(printf '%s\n' "$resp" | grep -c '^HTTP/1')"
}

echo "${bold}=== goproxy protocol tests → $HOST:$PORT ===${off}"

### body handling
send "GET, no body" \
	'GET /hello HTTP/1.1\r\nHost: x\r\n\r\n' \
	'200, Content-Length: 0'

send "POST with body" \
	'POST /api HTTP/1.1\r\nHost: x\r\nContent-Length: 11\r\n\r\nhello world' \
	'200, echoes "hello world", Content-Length: 11'

send "POST, body declared 0" \
	'POST /api HTTP/1.1\r\nHost: x\r\nContent-Length: 0\r\n\r\n' \
	'200, empty body'

send "client LIES: declares 100, sends 5" \
	'POST /api HTTP/1.1\r\nHost: x\r\nContent-Length: 100\r\n\r\nhello' \
	'400 — must NOT echo 95 NUL bytes'

send "negative Content-Length" \
	'POST /api HTTP/1.1\r\nHost: x\r\nContent-Length: -5\r\n\r\n' \
	'400 — and the server must survive'

send "oversized Content-Length" \
	'POST /api HTTP/1.1\r\nHost: x\r\nContent-Length: 99999999\r\n\r\n' \
	'413 Content Too Large'

send "non-numeric Content-Length" \
	'POST /api HTTP/1.1\r\nHost: x\r\nContent-Length: abc\r\n\r\n' \
	'400'

### connection handling — two requests pipelined in ONE write.
### "responses: 1" means the server closed after the first.
send "1.1 default (two requests)" \
	'GET /a HTTP/1.1\r\nHost: x\r\n\r\nGET /b HTTP/1.1\r\nHost: x\r\n\r\n' \
	'responses: 2 — 1.1 keeps alive by default'

send "1.1 + Connection: close" \
	'GET /a HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\nGET /b HTTP/1.1\r\nHost: x\r\n\r\n' \
	'responses: 1, Connection: close'

send "1.1 + Connection: CLOSE (mixed case)" \
	'GET /a HTTP/1.1\r\nHost: x\r\nConnection: CLOSE\r\n\r\nGET /b HTTP/1.1\r\nHost: x\r\n\r\n' \
	'responses: 1 — values are case-insensitive'

send "1.0 default" \
	'GET /a HTTP/1.0\r\nHost: x\r\n\r\nGET /b HTTP/1.0\r\nHost: x\r\n\r\n' \
	'responses: 1 — 1.0 closes by default'

send "1.0 + Connection: keep-alive" \
	'GET /a HTTP/1.0\r\nHost: x\r\nConnection: keep-alive\r\n\r\nGET /b HTTP/1.0\r\nHost: x\r\nConnection: keep-alive\r\n\r\n' \
	'responses: 2 — 1.0 opting in'

### request line and header errors
send "bad method" \
	'BREW /a HTTP/1.1\r\nHost: x\r\n\r\n' \
	'405 Method Not Allowed'

send "bad version" \
	'GET /a HTTP/9.9\r\nHost: x\r\n\r\n' \
	'505 HTTP Version Not Supported'

send "no spaces in request line" \
	'GET\r\n\r\n' \
	'400'

send "two spaces in request line" \
	'GET  /a HTTP/1.1\r\nHost: x\r\n\r\n' \
	'rejected — currently 505, arguably should be 400'

send "header with no colon" \
	'GET /a HTTP/1.1\r\nHost: x\r\nBrokenHeader\r\n\r\n' \
	'400'

send "lowercase header name" \
	'GET /a HTTP/1.1\r\nhost: x\r\ncontent-length: 0\r\n\r\n' \
	'200 — canonical keys make these match'

send "HEAD request" \
	'HEAD /a HTTP/1.1\r\nHost: x\r\n\r\n' \
	'Content-Length set, body MUST be empty'

send "duplicate Content-Length" \
	'POST /a HTTP/1.1\r\nHost: x\r\nContent-Length: 5\r\nContent-Length: 100\r\n\r\nhello' \
	'should be 400 (smuggling) — currently accepted'

### limits
printf '\n%s── 5000 header lines%s  %s(431 Header Fields Too Large)%s\n' "$bold" "$off" "$dim" "$off"
{ printf 'GET / HTTP/1.1\r\nHost: x\r\n'
  for i in $(seq 1 5000); do printf 'X-Pad-%d: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\r\n' "$i"; done
  printf '\r\n'; } | nc -w 3 "$HOST" "$PORT" | tr -d '\r' | head -1 | sed 's/^/  │ /'

printf '\n%s── one 200KB header line%s  %s(431)%s\n' "$bold" "$off" "$dim" "$off"
{ printf 'GET / HTTP/1.1\r\nHost: x\r\nX-Big: '
  head -c 200000 /dev/zero | tr '\0' 'a'
  printf '\r\n\r\n'; } | nc -w 3 "$HOST" "$PORT" | tr -d '\r' | head -1 | sed 's/^/  │ /'

### real client — keep-alive reuse. num_connects=0 means the socket was reused.
printf '\n%s── curl keep-alive reuse%s  %s(req2/req3 connects=0)%s\n' "$bold" "$off" "$dim" "$off"
curl -s --max-time 5 \
	-o /dev/null -w '  │ req1 status=%{http_code} connects=%{num_connects}\n' "http://$HOST:$PORT/a" \
	--next -s -o /dev/null -w '  │ req2 status=%{http_code} connects=%{num_connects}\n' "http://$HOST:$PORT/b" \
	--next -s -o /dev/null -w '  │ req3 status=%{http_code} connects=%{num_connects}\n' "http://$HOST:$PORT/c"

### did the server survive everything?
printf '\n%s── server still alive?%s\n' "$bold" "$off"
if curl -s --max-time 3 -o /dev/null -w '  │ yes — status %{http_code}\n' "http://$HOST:$PORT/alive"; then :; else
	printf '  │ %sNO — the server died during these tests%s\n' "$bold" "$off"
fi
echo
