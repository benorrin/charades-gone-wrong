# 🎭 Charades Gone Wrong

A hilarious party game featuring multiple round types: Pub Quiz, Charades, Alibi, and Copycat. Built with Go (backend), React (frontend), and SQLite.

## Project Structure

```
charades-gone-wrong/
├── backend/          # Go server (HTTP + WebSocket)
│   ├── db/          # Database initialization and queries
│   ├── game/        # Game engine and scoring logic
│   ├── api/         # HTTP endpoint handlers
│   ├── ws/          # WebSocket server
│   ├── main.go      # Server entry point
│   ├── go.mod       # Go module definition
│   └── game.db      # SQLite database (auto-created)
├── frontend/        # React + Vite + TypeScript
│   ├── src/
│   │   ├── pages/   # Page components (HomePage, GameRoom, etc.)
│   │   ├── components/  # Reusable UI components
│   │   ├── hooks/   # Custom React hooks
│   │   ├── types/   # TypeScript type definitions
│   │   └── main.tsx # React entry point
│   ├── vite.config.ts
│   └── package.json
└── scripts/         # Admin tools and seed data

## Game Modes

### 🎯 Pub Quiz (10 seconds)
Answer multiple-choice questions. Scoring: correct answer + speed bonus.

### 🎭 Charades Gone Wrong (30 seconds)
Act out a prompt. Others rate your performance (Rubbish/Good/Great/Funny/Cringe). Scoring based on ratings + speed multiplier.

### 📜 Alibi (60 seconds)
Improvise an alibi for a ridiculous crime. Others vote believability simultaneously. Scoring: believable votes × 2.

### 📝 Copycat (10 seconds)
Write a free-text answer. Matching answers = everyone drinks (0 points). Unique answers earn points.

## Quick Start

### Backend

```bash
cd backend
go mod download  # Download dependencies
go run main.go   # Start server on :8080
```

The backend will:
- Initialize SQLite database with schema
- Start HTTP server on port 8080
- Set up WebSocket for real-time game updates

### Frontend

```bash
cd frontend
npm install      # Install dependencies
npm run dev      # Start dev server on :3000
```

Vite dev server includes:
- Hot module reload (HMR) for fast development
- Proxy to backend API (`/api/`) and WebSocket (`/ws`)

### Database

SQLite database auto-initializes on server start. Schema includes:
- Games, Players, Rounds
- Content tables (pub_quiz_questions, charades_prompts, alibi_crimes, copycat_prompts)
- Scoring tables (player_scores, charades_ratings, alibi_votes, copycat_answers)

## Development

### Phase 2: Core Game Engine
- [x] Phase 1: Project setup
- [ ] Game state management (Game struct)
- [ ] Player management and join codes
- [ ] Round progression and content loading
- [ ] Scoring logic for all round types

### Phase 3-14: Additional Implementation
See `PLANNING.md` for full phase breakdown.