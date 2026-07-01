# Progress Tracker

Personal progress tracker learning project.

## Current version

- Go backend with `GET /health`
- Go backend with goals and sessions REST API
- Go backend with `GET /goals`
- Go backend with `POST /goals`
- Go backend with `GET /goals/{id}`
- Go backend with `DELETE /goals/{id}`
- Go backend with `POST /goals/{id}/sessions`
- Go backend with `GET /stats`
- SQLite database at `backend/data/progress.db`
- React frontend with goals, session timer, finish modal, and statistics

## Run locally

Use two terminals.

### Terminal 1: backend

```powershell
cd C:\Users\Admin\Desktop\ProgressTracker\backend
go run ./cmd/api
```

Backend runs at `http://127.0.0.1:8080`.

### Terminal 2: frontend

```powershell
cd C:\Users\Admin\Desktop\ProgressTracker\frontend
npm.cmd run dev
```

Open `http://127.0.0.1:5173/` in a browser to click through the app.
