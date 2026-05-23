# VulnArena

> The next-generation collaborative secure-code-audit platform.

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat&logo=go)](https://golang.org/)
[![SvelteKit](https://img.shields.io/badge/SvelteKit-2.0-FF3E00?style=flat&logo=svelte)](https://kit.svelte.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

VulnArena is a white-box secure-code-review platform for DevSecOps teams and offensive-security practitioners. Move beyond black-box CTFs: audit real-world CVEs, collaborate in real time with your squad, and get rigorous semantic evaluations powered by Claude.

## Features

- **White-box code audit** — read thousands of lines of real-world vulnerable code in C, C++, Rust, Go, Python, Node.js, Java, PHP, Ruby, C#, and more. 50 hand-crafted challenges spanning Heartbleed, Log4Shell, Shellshock, modern OWASP Top 10, and LLM/AI-era flaws.
- **Live co-op (multiplayer)** — real-time WebSocket-backed synchronization shows remote cursors and shared line selections inside the Monaco editor so squads can audit together.
- **AI-powered semantic evaluation** — Claude evaluates not only where the bug is but why it exists and how to remediate it. Falls back to a keyword evaluator on LLM failures so submissions are never blocked.
- **Deterministic F1 line scoring** — every challenge ships with verified vulnerable-line indices, validated by a build-time gate (`cmd/seed -verify`) that rejects empty lines, block-comment continuations, and out-of-range indices before any DB write.
- **Hacker CLI** — interact with the platform via the native `vulnarena` Go terminal tool.
- **Community Forge & gamification** — XP-gated user submissions, first-blood mechanics, Discord webhook integrations.
- **Hardened production stack** — TLS via Let's Encrypt, strict CSP, rate-limiting at both nginx and the Go middleware layer, JSON-structured audit logging, RBAC.

## Architecture

| Layer | Stack |
|---|---|
| Backend | Go 1.25 · chi · pgx v5 · go-redis · gorilla/websocket |
| Frontend | SvelteKit 2 · TypeScript · TailwindCSS · Monaco Editor |
| Storage | PostgreSQL 16 · Redis 7 |
| AI Engine | Anthropic Claude (Sonnet 4) |
| Infra | Docker Compose · Nginx · Certbot |

## Local development

```bash
# 1. Start infrastructure (Postgres + Redis)
docker compose up -d

# 2. Run migrations and seed the 50-challenge arsenal
go run ./cmd/migrate -direction up
go run ./cmd/seed

# 3. Run the API
go run ./cmd/api

# 4. Run the frontend
cd web && npm install && npm run dev
```

The API listens on `:8080`, the SvelteKit dev server on `:5173`.

## Verifying challenge data

Every seed run verifies that each challenge's `VulnerableLines` array points to non-empty, non-comment code. To run verification without touching the database:

```bash
go run ./cmd/seed -verify        # silent on success
go run ./cmd/seed -verify -v     # prints every matched line
```

The verifier gates the seed binary itself — there is no path to insert a challenge whose lines are wrong.

## Production deployment (vulnarena.com)

The production stack terminates TLS in nginx using a Let's Encrypt cert provisioned and renewed by an in-container certbot service.

```bash
# 1. Copy the prod env template and fill in real values
cp .env.production.example .env
$EDITOR .env   # DOMAIN, CERTBOT_EMAIL, POSTGRES_PASSWORD, JWT_SECRET, ...

# 2. Point DNS for vulnarena.com and www.vulnarena.com at this host

# 3. One-time cert bootstrap (creates the initial Let's Encrypt cert)
./scripts/init-letsencrypt.sh

# 4. Bring up the full stack
docker compose -f docker-compose.prod.yml up -d

# 5. Run migrations + seed (idempotent; safe to re-run)
docker compose -f docker-compose.prod.yml run --rm migrate
docker compose -f docker-compose.prod.yml run --rm api /app/seed
```

The certbot service runs `certbot renew` every 12 hours; nginx reloads on the same cadence to pick up renewed certs. No manual renewal needed.

### Required production environment variables

See `.env.production.example` for the full list. Critical ones:

- `DOMAIN`, `CERTBOT_EMAIL` — for TLS issuance
- `POSTGRES_PASSWORD`, `REDIS_PASSWORD` — strong, unique secrets
- `JWT_SECRET` — generate with `openssl rand -hex 32`
- `ORIGIN` — must match the public URL (`https://vulnarena.com`)
- `ALLOWED_ORIGINS` — comma-separated CORS allowlist
- `ANTHROPIC_API_KEY` — optional; enables AI semantic evaluation
- `DB_MAX_CONNS`, `DB_MIN_CONNS` — pgxpool sizing

### Production health checks

After `docker compose up -d`, confirm:

```bash
curl https://vulnarena.com/health        # expect 200 OK
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs api | grep "connected to PostgreSQL"
```

## License

MIT
