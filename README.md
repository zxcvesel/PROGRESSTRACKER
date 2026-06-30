# Progress Tracker

Personal progress tracker learning project.

## Current version

- Go backend with `GET /health`
- Go backend with goals and sessions REST API
- Go backend with `GET /goals`
- Go backend with `POST /goals`
- Go backend with `GET /goals/{id}`
- Go backend with `POST /goals/{id}/sessions`
- Go backend with `GET /stats`
- SQLite database at `backend/data/progress.db`
- React frontend with goals, session timer, finish modal, and statistics

## Run backend

```powershell
cd backend
go run ./cmd/api
```

## Run frontend

```powershell
cd frontend
npm.cmd run dev
```
