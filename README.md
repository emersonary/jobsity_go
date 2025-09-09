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

## Environment Variables
The application can be configured via `config.yaml` (at the project's root) or environment variables. Environment variables take precedence.

| Config key                 | Env variable           | Description |
|----------------------------|------------------------|-------------|
| `jwt_secret`               | `JWT_SECRET`           | Secret key used to sign JWT tokens |
| `auth_user`                | `AUTH_USER`            | Username for login (default `demo`) |
| `auth_pass`                | `AUTH_PASS`            | Password for login (default `demo123`) |
| `search_timeout`           | `SEARCH_TIMEOUT`       | Timeout for provider API requests (e.g. `10s`) |
| `cache_ttl`                | `CACHE_TTL`            | Duration to cache flight results in memory (e.g. `30s`) |
| `tls_cert_file`            | `TLS_CERT_FILE`        | Path to TLS certificate (leave empty to disable TLS) |
| `tls_key_file`             | `TLS_KEY_FILE`         | Path to TLS key file (leave empty to disable TLS) |
| `amadeus_url`              | `AMADEUS_URL`          | Base URL for Amadeus API (default `https://test.api.amadeus.com`) |
| `amadeus_clientid`         | `AMADEUS_CLIENT_ID`    | Amadeus API client ID |
| `amadeus_clientsecret`     | `AMADEUS_CLIENT_SECRET`| Amadeus API client secret |
| `duffel_host`              | `DUFFEL_HOST`          | Base URL for Duffel API (default `https://api.duffel.com`) |
| `duffel_token`             | `DUFFEL_TOKEN`         | Duffel API token |
| `rapid_booking_host`       | `RAPIDAPI_HOST`        | RapidAPI Booking.com host (default `booking-com15.p.rapidapi.com`) |
| `rapid_booking_rapidapikey`| `RAPIDAPI_KEY`         | RapidAPI key for Booking.com flights |

Example `config.yaml`:
```yaml
jwt_secret: "jobsity-assessment-secret"
auth_user: "demo"
auth_pass: "demo123"
search_timeout: "10s"
cache_ttl: "30s"
tls_cert_file: ""
tls_key_file: ""
amadeus_url: "https://test.api.amadeus.com"
amadeus_clientid: "<your-client-id>"
amadeus_clientsecret: "<your-client-secret>"
duffel_host: "https://api.duffel.com"
duffel_token: "<your-duffel-token>"
rapid_booking_host: "booking-com15.p.rapidapi.com"
rapid_booking_rapidapikey: "<your-rapidapi-key>"
```

---

## How to Run the Application

### Local Development
```bash
# clone the repository
git clone https://github.com/your-org/go-flights-service.git
cd go-flights-service

# set environment variables or create config.yaml
export AUTH_USER=demo AUTH_PASS=demo123 JWT_SECRET=change-me

# run tests
go test ./...

# run the application
go run ./cmd/server
```

### Docker
```bash
# build docker image
docker build -t flights-service .

# run container with env variables
docker run -p 8080:8080   -e AUTH_USER=demo   -e AUTH_PASS=demo123   -e JWT_SECRET=change-me   flights-service
```

The server will be available at `http://localhost:8080`. Use the Quickstart steps above to login and query endpoints.

