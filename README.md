# Storage Analysis Webapp

A local Windows storage analysis app with a Go backend and a React + TypeScript frontend. It scans folder sizes, finds duplicate files by content hash, and supports file management operations.

## Features

- Folder size analysis with expandable tree and sorting
- Disk-backed scan cache for faster repeat scans
- Duplicate file discovery using SHA-256 hashes
- Copy/move via multithreaded Robocopy
- File and folder deletion with root-path safeguards

## Requirements

- Go 1.25+ (for standard-library CSRF and confined filesystem APIs)
- Node.js 20.19+
- Windows (Robocopy is used for copy/move operations)

## Development

Run the backend and frontend in separate terminals.

Backend, from `backend/`:

- `go run ./cmd/server`
- API listens on `http://localhost:8080`.

Frontend, from `frontend/`:

- `npm ci`
- `npm run dev`
- Open `http://localhost:5173`. Vite proxies API requests to the backend.

## File operation security (Issue #1)

**Safe defaults:** The backend binds to `127.0.0.1:8080`. All copy, move
and delete endpoints return HTTP 503 unless both `OPS_AUTH_TOKEN` and
`OPS_ALLOWED_ROOTS` are configured. Read-only local scans still work
without an operation token. The browser asks for the token when a protected
API call first returns HTTP 401; the token lives **only in page memory**
and is not written to localStorage or a cookie.

Before enabling destructive operations, choose one or more **existing
directories**, not an entire drive or filesystem root, as the allowed
roots. All operation sources and destinations must be absolute and within
those directories. Copy and move destinations must already exist and be
directories **below** an allowed root. Symlink/junction paths and attempts
to delete an allowed root are rejected. Copying a directory into itself
is rejected. Deletion uses `os.Root.RemoveAll`. Robocopy uses `/XJ /SL`
to avoid traversing links; as an external executable it still has a
check/use race if a local adversary can concurrently replace paths.

Example **PowerShell** (use your own trusted directory and do not share
the generated token):

```powershell
$bytes = New-Object byte[] 32
[Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
$env:OPS_AUTH_TOKEN = [BitConverter]::ToString($bytes).Replace('-', '')
$env:OPS_ALLOWED_ROOTS = 'C:\Users\YOUR_USER\Documents'
# Optional additional directories: separate with commas.
cd backend
go run ./cmd/server
```

For command-line API calls send the token via
`Authorization: Bearer <token>` and use
`Content-Type: application/json` on operation requests. Do not put tokens
in URLs or browser storage.

### Remote deployment (opt-in only)

Non-loopback binds, for example `ADDRESS=0.0.0.0:8080`, are **rejected
at startup** unless all of the following are configured:

- `ALLOW_REMOTE_BIND=true`
- `OPS_AUTH_TOKEN` (at least 32 characters of secret random material)
- `OPS_ALLOWED_ROOTS` (one or more existing restricted directories)
- `TLS_CERT_FILE` and `TLS_KEY_FILE` (server TLS certificate and key)

The server then serves HTTPS. In remote mode, the Bearer token is also
required on read-only data API endpoints (except `/api/health`). A
production deployment must use a trusted certificate, firewall controls
and an appropriately scoped account with minimal filesystem permissions.
`ALLOWED_ORIGINS` is a comma-separated list of trusted browser origins;
the default permits the Vite dev origin `http://localhost:5173`.
Cross-origin operations from other origins are rejected, and
`net/http.CrossOriginProtection` blocks cross-site unsafe requests.
The application never uses cookie authentication, so no ambient browser
credential is accepted for file operations.

**Do not expose an insecure loopback listener through a reverse proxy,
tunnel or port forwarding.** When using a reverse proxy, explicitly enable
the protected mode and verify TLS on every externally reachable route;
the application cannot automatically detect external exposure through a
loopback-only listener.

## Production build

From `frontend/`, run `npm run build`. Vite writes the static app to `backend/public`; then run the backend from `backend/` with `go run ./cmd/server` to serve the built app.

## Tests

From `backend/`, run `go test ./...`.

From `frontend/`, run `npm ci`, `npm run test`, and `npm run e2e`.
