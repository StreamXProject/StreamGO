# StreamGO ⚡

High-performance, ultra-low-memory media streaming and API service written in Go.

## Features
- **Chi Router**: Lightweight, idiomatic HTTP router with standard middlewares (Logger, Recoverer, CORS, RealIP).
- **MongoDB Driver v2**: Modern connection pooling with ping health checks and graceful teardown.
- **Telegram MTProto (`gotd/td`)**: Pure Go MTProto client for media streaming and bot integration.
- **Minimal Footprint**: Operates in **~15 MB – 25 MB RAM** under idle/normal conditions.

## Project Structure
```
StreamGO/
├── cmd/
│   └── server/
│       └── main.go          # Application lifecycle and graceful shutdown
├── internal/
│   ├── config/
│   │   └── config.go        # .env and environment variable loader
│   ├── database/
│   │   └── mongodb.go       # MongoDB connection pool and health checks
│   ├── server/
│   │   └── router.go        # Chi router, middlewares, and API endpoints
│   └── telegram/
│       └── client.go        # gotd/td MTProto service wrapper
├── .env.example             # Template environment file
├── .gitignore
├── go.mod
└── README.md
```

## Getting Started

### 1. Configure Environment
Copy `.env.example` to `.env` and fill in your credentials:
```bash
cp .env.example .env
```

### 2. Run Locally
```bash
go run ./cmd/server
```

### 3. Check Health
```bash
curl http://localhost:8000/health
```

Sample response:
```json
{
  "database": "healthy",
  "service": "StreamGO",
  "status": "ok",
  "time": "2026-09-18T11:05:00Z",
  "uptime": "5.123s"
}
```

### 4. Build Production Binary
```bash
go build -ldflags="-s -w" -o bin/server ./cmd/server
```
Produces a single static binary (~18 MB on disk, uses ~20 MB RAM at runtime).
