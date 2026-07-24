# Monorepo Deployment on Vercel

This guide documents how the StockForge monorepo (`apps/web` + `apps/api`) is deployed as a single Vercel project using Vercel's Services feature.

## Project Structure

```
stock-forge/
├── apps/
│   ├── web/          # Next.js frontend
│   └── api/          # Go REST API (Framework Preset)
├── vercel.json       # Vercel services + routing config
└── docs/
```

## Vercel Services Configuration

`vercel.json` at the project root defines two **services** — each built independently but sharing one domain:

```json
{
  "services": {
    "web": {
      "root": "apps/web"
    },
    "api": {
      "root": "apps/api",
      "framework": "go",
      "entrypoint": "cmd/main.go"
    }
  },
  "rewrites": [
    {
      "source": "/api/(.*)",
      "destination": { "service": "api" }
    },
    {
      "source": "/(.*)",
      "destination": { "service": "web" }
    }
  ]
}
```

### How routing works

- All requests enter through the top-level route table.
- `/api/*` paths are forwarded to the `api` service (Go server).
- Everything else goes to the `web` service (Next.js).

> The service receives the **original** request path. `GET /api/test` reaches the Go server as `/api/test`, not `/test`. All Go handlers must be prefixed with `/api/`.

## Go API Service

### Framework Preset

The API uses Vercel's Go Framework Preset (not serverless functions). Vercel detects the Go service from `go.mod` at the service root and builds the server from the specified entrypoint.

**Entrypoint:** `cmd/main.go`

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/", pkg.Root)
mux.HandleFunc("/api/test", pkg.Test)
```

### Environment variables

- `PORT` is injected automatically by Vercel.
- Additional variables (database URLs, secrets, etc.) are set in the Vercel dashboard per service.
- The `.env` file is for local development only and **must not** be committed.

`godotenv.Load()` is called with a non-fatal guard so the server starts fine on Vercel where no `.env` exists:

```go
if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
    log.Fatal("Error loading .env file: ", err)
}
```

### CORS (local development)

In production, the frontend and API share the same origin (`stock-forge-golang-nextjs.vercel.app`), so no CORS is needed.

Locally, the Next.js dev server runs on `http://localhost:3000` and the Go server on `http://localhost:8080` — different ports mean different origins, which triggers CORS errors.

A CORS middleware is added to the Go server to handle this:

```go
func CORSMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

The middleware is applied by wrapping the mux in `main.go`:

```go
handler := pkg.CORSMiddleware(mux)
log.Fatal(http.ListenAndServe(addr, handler))
```

### Local development

```bash
cd apps/api
go run ./cmd
```

Uses [air](https://github.com/air-verse/air) for hot reload (`.air.toml`).

## Available Endpoints

| Endpoint | Response |
|----------|----------|
| `GET /api/` | `"Hello World!"` |
| `GET /api/test` | `{"status":200,"message":"Hello this is Test"}` |

## Deployment Flow

1. Push to `main` branch.
2. Vercel detects the monorepo, builds both services in parallel:
   - **web** — Next.js build → static + serverless output.
   - **api** — `go build -o server ./cmd` → Go server binary.
3. Vercel runs the Go server, listening on `$PORT`.
4. Rewrite rules route traffic to the correct service.

## Key Learnings

- The `"services"` key in `vercel.json` is the correct way to deploy multiple runtimes (Next.js + Go) in a single project.
- Each service's `"framework"` and `"entrypoint"` must be explicitly set when the auto-detected defaults don't match the project layout.
- Vercel Go serverless functions (`api/*.go`) are an alternative, but the Framework Preset (standalone server) is simpler when you already have a Go `http.ServeMux` set up.
- `.env` loading must be tolerant of missing files for Vercel compatibility.
