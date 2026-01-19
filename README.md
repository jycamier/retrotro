<img src="docs/img/logo.png" alt="Retrotro">

<p align="center">
  <img src="docs/img/gopher.png" alt="Retrotro Gopher">
</p>

> *"So so funny!"* - Retrotro the Gopher

A real-time collaborative retrospective tool for agile teams.

> **Warning**
> This project is **experimental** and under active development. It is not production-ready. Use at your own risk. APIs, database schemas, and features may change without notice.

---

## About This Project

This project was **vibecoded** - built collaboratively with AI assistance (Claude). The codebase was developed through natural language conversations, with AI helping to write, debug, and iterate on the code. This approach prioritizes rapid prototyping and exploration over traditional software development practices.

---

## Features

- **Real-time collaboration** via WebSocket
- **Multiple retrospective phases**:
  - Icebreaker (weather mood check-in)
  - Brainstorm (add items to columns)
  - Group (cluster similar items)
  - Vote (prioritize items)
  - Discuss (review top items)
  - Action (create action items)
  - ROTI (Return On Time Invested rating)
- **Customizable templates** (e.g., Start/Stop/Continue, Mad/Sad/Glad, 4Ls)
- **Timer management** with pause/resume/extend
- **Team management** with role-based access
- **OIDC authentication** support (+ dev mode for local testing)
- **Anonymous voting** option
- **Action item tracking** with assignees and due dates

---

## Tech Stack

### Backend
- **Go** (chi router)
- **PostgreSQL** (with pgx driver)
- **WebSocket** for real-time updates
- **JWT** authentication

### Frontend
- **React 18** with TypeScript
- **Vite** for development and building
- **TanStack Query** for data fetching
- **Zustand** for state management
- **Tailwind CSS** for styling
- **Lucide React** for icons

---

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Node.js 18+ (for local frontend development)
- Go 1.21+ (for local backend development)

### Running with Docker Compose

```bash
# Clone the repository
git clone https://github.com/jycamier/retrotro.git
cd retrotro

# Start all services
docker compose up -d

# Apply database migrations
cat backend/migrations/*.up.sql | docker compose exec -T postgres psql -U retrotro -d retrotro

# Access the application
open http://localhost:3000
```

### Local Development

**Backend:**
```bash
cd backend
cp .env.example .env
go run ./cmd/server
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev
```

---

## Project Structure

```
retrotro/
├── backend/
│   ├── cmd/server/          # Application entrypoint
│   ├── internal/
│   │   ├── auth/            # OIDC & JWT authentication
│   │   ├── config/          # Configuration loading
│   │   ├── handlers/        # HTTP & WebSocket handlers
│   │   ├── middleware/      # HTTP middleware
│   │   ├── models/          # Domain models
│   │   ├── repository/      # Data access layer
│   │   ├── services/        # Business logic
│   │   └── websocket/       # WebSocket hub
│   └── migrations/          # SQL migrations
├── frontend/
│   ├── src/
│   │   ├── api/             # API client
│   │   ├── components/      # React components
│   │   ├── hooks/           # Custom hooks
│   │   ├── pages/           # Page components
│   │   ├── store/           # Zustand stores
│   │   └── types/           # TypeScript types
│   └── public/
└── docker-compose.yml
```

---

## Environment Variables

### Backend

Copy `backend/.env.example` to `backend/.env` and adjust the values.

#### Server

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://retrotro:retrotro@localhost:5432/retrotro?sslmode=disable` |
| `CORS_ORIGINS` | Comma-separated list of allowed origins | `http://localhost:3000` |
| `DEV_MODE` | Enable dev login endpoints (no OIDC required) | `false` |

#### Logging

| Variable | Description | Default |
|----------|-------------|---------|
| `LOGGER_LEVEL` | Log level: `debug`, `info`, `warn`, `error` | `info` |
| `LOGGER_FORMAT` | Log format: `txt`, `json` | `txt` |
| `FX_LOGS` | Enable framework logs | `false` |

#### JWT

| Variable | Description | Default |
|----------|-------------|---------|
| `JWT_SECRET` | Secret for JWT signing | `change-me-in-production` |
| `JWT_ACCESS_TOKEN_TTL` | Access token TTL in minutes | `15` |
| `JWT_REFRESH_TOKEN_TTL` | Refresh token TTL in hours | `168` (7 days) |

#### OIDC (OpenID Connect)

| Variable | Description | Default |
|----------|-------------|---------|
| `OIDC_ISSUER_URL` | OIDC provider URL | - |
| `OIDC_CLIENT_ID` | OIDC client ID | - |
| `OIDC_CLIENT_SECRET` | OIDC client secret | - |
| `OIDC_REDIRECT_URL` | Callback URL after authentication | `http://localhost:8080/auth/callback` |
| `OIDC_SCOPES` | Comma-separated OIDC scopes | `openid,profile,email,groups` |

#### OIDC JIT (Just-In-Time) Provisioning

| Variable | Description | Default |
|----------|-------------|---------|
| `OIDC_JIT_ENABLED` | Enable automatic user/team provisioning | `true` |
| `OIDC_JIT_GROUPS_CLAIM` | Claim name containing user groups | `groups` |
| `OIDC_JIT_GROUPS_PREFIX` | Prefix to filter relevant groups | - |
| `OIDC_JIT_DEFAULT_ROLE` | Default role for new users | `participant` |
| `OIDC_JIT_ADMIN_GROUPS` | Comma-separated groups granting admin role | - |
| `OIDC_JIT_FACILITATOR_GROUPS` | Comma-separated groups granting facilitator role | - |
| `OIDC_JIT_SYNC_ON_LOGIN` | Sync user groups on each login | `true` |
| `OIDC_JIT_REMOVE_STALE_MEMBERS` | Remove users from teams they no longer belong to | `false` |

---

## License

MIT

---

## Contributing

This is an experimental project. Feel free to fork and experiment! Issues and PRs are welcome, but response times may vary.

---

*Built with vibes and AI assistance*
