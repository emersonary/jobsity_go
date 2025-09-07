# Go Flights Microservice (assessment-ready)


## Features
- REST endpoints with JWT auth
- `POST /auth/login` → `{username, password}` returns `{token}`
- `GET /flights/search?origin=XXX&destination=YYY&date=YYYY-MM-DD` (Bearer token required)
- `GET /flights/history?origin=XXX&destination=YYY` (last 24 months synthetic)
- `GET /sse/{origin}/{destination}?date=YYYY-MM-DD` (SSE stream — updates every 30s)
- `GET /ws/{origin}/{destination}?date=YYYY-MM-DD` (WebSocket stream — updates every 30s)
- Parallel provider fetching (3 providers mocked & concurrent)
- In-memory caching (default TTL 30s)
- Deterministic synthetic data for history; swap providers with real HTTP clients later
- Graceful shutdown; configurable timeouts
- Dockerfile provided


## Quickstart
```bash
# 1) Run tests
go test ./...


# 2) Start server
export AUTH_USER=demo AUTH_PASS=demo123 JWT_SECRET=change-me
go run ./cmd/server


# 3) Obtain token
curl -s localhost:8080/auth/login -XPOST -d '{"username":"demo","password":"demo123"}' \
-H 'content-type: application/json' | jq -r .token > /tmp/tok


# 4) Search (replace date)
TOK=$(cat /tmp/tok)
curl -s "localhost:8080/flights/search?origin=AMS&destination=BCN&date=2025-10-01" \
-H "Authorization: Bearer $TOK" | jq


# 5) History
curl -s "localhost:8080/flights/history?origin=AMS&destination=BCN" \
-H "Authorization: Bearer $TOK" | jq


# 6) SSE stream (press Ctrl+C to stop)
curl -N -H "Authorization: Bearer $TOK" "localhost:8080/sse/AMS/BCN?date=2025-10-01"


# 7) WebSocket stream (requires websocat)
websocat -H "Authorization: Bearer $TOK" \
ws://localhost:8080/ws/AMS/BCN?date=2025-10-01
```


## Real providers
This repo deals with Amadeus, Duffel, and Rapid API (booking.com)
You should configure their secrets in config.yaml file or override them by environmental variables
