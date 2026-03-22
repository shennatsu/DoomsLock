# DoomsLock — Backend API

Backend service for DoomsLock, an app that helps people overcome doom scrolling through a peer-based parental guidance system (peer accountability). The current version is still in the development phase.

## Tech Stack

- **Go 1.22+** with Echo v4
- **PostgreSQL 16** (pgx/v5)
- **Redis 7** (go-redis/v9)
- **JWT** (golang-jwt/v5)
- **Zap** (uber-go/zap structured logging)
- **Viper** (config management)

## Quick Start

```bash
# 1. Clone dan masuk ke folder
cd backend

# 2. Copy env file
cp .env.example .env

# 3. Start PostgreSQL & Redis
docker compose up -d

# 4. Run migration (perlu install golang-migrate CLI)
migrate -path migrations -database 'postgres://doomslock:doomslock123@localhost:5432/doomslock?sslmode=disable' up

# 5. Run server
go run cmd/api/main.go

# 6. Test health
curl http://localhost:8080/health
```

## API Endpoints

### Auth (`/api/v1/auth`)
| Method | Endpoint     | Auth | Description       |
|--------|-------------|------|-------------------|
| POST   | /register   | ❌   | Register user     |
| POST   | /login      | ❌   | Login             |
| POST   | /refresh    | ❌   | Refresh token     |
| POST   | /logout     | ✅   | Logout            |

### Groups (`/api/v1/groups`)
| Method | Endpoint                | Auth | Description       |
|--------|------------------------|------|-------------------|
| POST   | /                      | ✅   | Create group      |
| GET    | /                      | ✅   | List my groups    |
| GET    | /:id                   | ✅   | Group detail      |
| POST   | /:id/invite            | ✅   | Generate invite   |
| POST   | /join                  | ✅   | Accept invite     |
| POST   | /:id/leave             | ✅   | Leave group       |
| DELETE | /:id/members/:user_id  | ✅   | Remove member     |

### Limits (`/api/v1/limits`)
| Method | Endpoint        | Auth | Description           |
|--------|----------------|------|-----------------------|
| POST   | /              | ✅   | Create app limit      |
| GET    | /?group_id=    | ✅   | List limits by group  |
| PATCH  | /:id           | ✅   | Update limit          |
| DELETE | /:id           | ✅   | Soft delete limit     |

### Extensions (`/api/v1/extensions`)
| Method | Endpoint                      | Auth | Description         |
|--------|------------------------------|------|---------------------|
| POST   | /                            | ✅   | Request extension   |
| GET    | /:id                         | ✅   | Extension detail    |
| POST   | /:id/vote                    | ✅   | Cast vote           |
| GET    | /limits/:limit_id/extensions | ✅   | List by limit       |

### Usage (`/api/v1/usage`)
| Method | Endpoint       | Auth | Description       |
|--------|---------------|------|-------------------|
| POST   | /sync         | ✅   | Batch sync usage  |
| GET    | /summary?date=| ✅   | Daily summary     |

### Rewards (`/api/v1/rewards`)
| Method | Endpoint       | Auth | Description       |
|--------|---------------|------|-------------------|
| GET    | /streak       | ✅   | Get current streak|
| POST   | /streak/update| ✅   | Update streak     |
| GET    | /badges       | ✅   | List badges       |

## Architecture

```
cmd/api/          → entrypoint
config/           → viper config
pkg/              → shared packages (database, redis, middleware, etc)
internal/
  auth/           → register, login, JWT
  group/          → group CRUD, invite, join
  limit/          → app limit management
  extension/      → vote request untuk perpanjangan waktu
  usage/          → sync usage dari device
  reward/         → streak & badge tracking
migrations/       → SQL migration files
```

## Concept of Application

DoomsLock is a peer-accountability app designed to combat doom scrolling:

1. **Users** register and join a **Group** (3–6 friends)
2. Users set **App Limits** (e.g., Instagram: max 30 minutes/day)
3. If the limit is reached and you want an extension, you must **Request Extension**
4. Friends in the group **Vote** — approval requires a majority vote
5. **Usage** is synced from the Android device to the server
6. If usage is < 1 hour per day, you get a **Streak** and automatically receive a **Badge**

