# Storage Analysis Webapp

A local Windows storage analysis app with a Go backend and a React + TypeScript frontend. It scans folder sizes, finds duplicate files by content hash, and supports file management operations.

## Features

- Folder size analysis with expandable tree and sorting
- Disk-backed scan cache for faster repeat scans
- Duplicate file discovery using SHA-256 hashes
- Copy/move via multithreaded Robocopy
- File and folder deletion with root-path safeguards

## Requirements

- Go 1.22+
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

## Security notice

The backend defaults to `127.0.0.1:8080`. Its file copy, move, and delete endpoints do not provide authentication. Run only on a trusted local machine; do not expose the backend through a network bind, reverse proxy, tunnel, container port, or port forwarding.

## Production build

From `frontend/`, run `npm run build`. Vite writes the static app to `backend/public`; then run the backend from `backend/` with `go run ./cmd/server` to serve the built app.

## Tests

From `backend/`, run `go test ./...`.

From `frontend/`, run `npm ci`, `npm run test`, and `npm run e2e`.
