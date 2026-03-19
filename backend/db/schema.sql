-- Games table
CREATE TABLE IF NOT EXISTS games (
    id TEXT PRIMARY KEY,
    code TEXT UNIQUE NOT NULL,
    host_id TEXT NOT NULL,
    game_type TEXT NOT NULL CHECK(game_type IN ('normal', 'spicy', 'unhinged')),
    round_count INTEGER NOT NULL DEFAULT 20,
    current_round INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'lobby' CHECK(status IN ('lobby', 'in_progress', 'finished')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Players table
CREATE TABLE IF NOT EXISTS players (
    id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL,
    name TEXT,
    is_host BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(game_id) REFERENCES games(id) ON DELETE CASCADE
);

-- Pub Quiz Questions
CREATE TABLE IF NOT EXISTS pub_quiz_questions (
    id TEXT PRIMARY KEY,
    text TEXT NOT NULL,
    answers TEXT NOT NULL, -- JSON array of 4 answers
    correct_answer_idx INTEGER NOT NULL CHECK(correct_answer_idx >= 0 AND correct_answer_idx < 4),
    game_type TEXT NOT NULL CHECK(game_type IN ('normal', 'spicy', 'unhinged')),
    is_player_specific BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Charades Prompts
CREATE TABLE IF NOT EXISTS charades_prompts (
    id TEXT PRIMARY KEY,
    prompt TEXT NOT NULL,
    game_type TEXT NOT NULL CHECK(game_type IN ('normal', 'spicy', 'unhinged')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Alibi Crimes
CREATE TABLE IF NOT EXISTS alibi_crimes (
    id TEXT PRIMARY KEY,
    crime_text TEXT NOT NULL,
    game_type TEXT NOT NULL CHECK(game_type IN ('normal', 'spicy', 'unhinged')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Copycat Prompts
CREATE TABLE IF NOT EXISTS copycat_prompts (
    id TEXT PRIMARY KEY,
    prompt TEXT NOT NULL,
    game_type TEXT NOT NULL CHECK(game_type IN ('normal', 'spicy', 'unhinged')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Rounds table
CREATE TABLE IF NOT EXISTS rounds (
    id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL,
    round_num INTEGER NOT NULL,
    round_type TEXT NOT NULL CHECK(round_type IN ('pub_quiz', 'charades', 'alibi', 'copycat')),
    prompt TEXT, -- for charades/copycat
    crime_text TEXT, -- for alibi
    question_id TEXT, -- for pub_quiz
    actor_player_id TEXT, -- for charades and alibi
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY(actor_player_id) REFERENCES players(id) ON DELETE SET NULL,
    FOREIGN KEY(question_id) REFERENCES pub_quiz_questions(id) ON DELETE SET NULL
);

-- Player Scores
CREATE TABLE IF NOT EXISTS player_scores (
    id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL,
    player_id TEXT NOT NULL,
    round_num INTEGER NOT NULL,
    round_type TEXT NOT NULL CHECK(round_type IN ('pub_quiz', 'charades', 'alibi', 'copycat')),
    points INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY(player_id) REFERENCES players(id) ON DELETE CASCADE
);

-- Charades Ratings
CREATE TABLE IF NOT EXISTS charades_ratings (
    id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL,
    round_num INTEGER NOT NULL,
    actor_player_id TEXT NOT NULL,
    rater_player_id TEXT NOT NULL,
    rating TEXT NOT NULL CHECK(rating IN ('rubbish', 'good', 'great', 'funny', 'cringe')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY(actor_player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY(rater_player_id) REFERENCES players(id) ON DELETE CASCADE
);

-- Alibi Votes
CREATE TABLE IF NOT EXISTS alibi_votes (
    id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL,
    round_num INTEGER NOT NULL,
    suspect_player_id TEXT NOT NULL,
    voter_player_id TEXT NOT NULL,
    believable BOOLEAN NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY(suspect_player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY(voter_player_id) REFERENCES players(id) ON DELETE CASCADE
);

-- Copycat Answers
CREATE TABLE IF NOT EXISTS copycat_answers (
    id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL,
    round_num INTEGER NOT NULL,
    player_id TEXT NOT NULL,
    answer_text TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY(player_id) REFERENCES players(id) ON DELETE CASCADE
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_games_code ON games(code);
CREATE INDEX IF NOT EXISTS idx_players_game_id ON players(game_id);
CREATE INDEX IF NOT EXISTS idx_rounds_game_id ON rounds(game_id);
CREATE INDEX IF NOT EXISTS idx_player_scores_game_id ON player_scores(game_id);
CREATE INDEX IF NOT EXISTS idx_charades_ratings_game_id ON charades_ratings(game_id);
CREATE INDEX IF NOT EXISTS idx_alibi_votes_game_id ON alibi_votes(game_id);
CREATE INDEX IF NOT EXISTS idx_copycat_answers_game_id ON copycat_answers(game_id);
