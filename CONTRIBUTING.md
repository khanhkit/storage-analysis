# Contributing

Issues and pull requests are welcome. Please include a clear description and test the changes locally.

## Checks

- Backend: from `backend/`, run `go test ./...`.
- Frontend: from `frontend/`, run `npm ci` and `npm run test`.
- E2E: from `frontend/`, run `npm run e2e` (requires the backend at `http://localhost:8080`).

Keep changes focused. Do not commit local caches, generated output, executables, or credentials.
