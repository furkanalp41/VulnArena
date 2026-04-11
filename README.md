\# 🛡️ VulnArena



> \*\*The Next-Generation Collaborative Secure Code Audit Platform\*\*



\[!\[Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat\&logo=go)](https://golang.org/)

\[!\[SvelteKit](https://img.shields.io/badge/SvelteKit-2.0-FF3E00?style=flat\&logo=svelte)](https://kit.svelte.dev/)

\[!\[Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat\&logo=docker)](https://www.docker.com/)

\[!\[License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)



VulnArena is an elite, open-source White-Box Secure Code Review platform designed for DevSecOps teams and offensive security professionals. Move beyond standard black-box CTFs: audit real-world CVEs, collaborate in real-time with your squad, and get semantic evaluations powered by AI.



\## ✨ Insane Engineering Features



\* \*\*White-Box Code Audit:\*\* Dive into thousands of lines of code. Target specific vulnerable lines in C, C++, Rust, Go, Python, Node.js, and more using the integrated Monaco editor. Contains a built-in arsenal of 30+ historical CVEs (e.g., Heartbleed, Log4Shell).

\* \*\*Live Co-op (Multiplayer):\*\* Offensive security is a team sport. Real-time WebSocket synchronization allows Squads to see remote cursors and shared line selections instantly.

\* \*\*AI-Powered Semantic Evaluation:\*\* Integrated with the Anthropic Claude API to evaluate not just \*where\* the bug is, but \*why\* it exists and \*how\* to remediate it.

\* \*\*The Anthropic Aesthetic:\*\* A highly refined, eye-strain-free, minimalist UI with dynamic Arena filtering (by language, category, and difficulty).

\* \*\*The Hacker CLI:\*\* Don't like web interfaces? Interact with the platform directly via the native `vulnarena` Go terminal tool.

\* \*\*Community Forge \& Gamification:\*\* XP-gated user submissions, First Blood mechanics, Discord C2 webhook integrations, and real-time alerts.

\* \*\*Hardened Architecture:\*\* Strict CSP, Redis-backed Rate Limiting, JSON Structured Audit Logging, and robust RBAC.



\## 🏗️ Architecture \& Tech Stack



\* \*\*Backend:\*\* Go (Golang), PostgreSQL, Redis, Gorilla WebSockets.

\* \*\*Frontend:\*\* SvelteKit, TypeScript, TailwindCSS, Monaco Editor.

\* \*\*AI Engine:\*\* Anthropic Claude 3 API.

\* \*\*Infrastructure:\*\* Docker, Docker Compose, Nginx.



\## 🚀 Getting Started (Local Development)



To spin up the complete environment locally, ensure Docker Desktop is running, then execute:



```bash

\# 1. Start Infrastructure (Postgres \& Redis)

docker-compose up -d



\# 2. Run Migrations \& Seed the 30-CVE Arsenal

go run ./cmd/migrate/main.go up

go run ./cmd/seed



\# 3. Start the Go Backend API

go run ./cmd/api/main.go

