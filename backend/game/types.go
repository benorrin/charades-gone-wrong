package game

import "time"

// GameType represents the difficulty/theme of a game
type GameType string

const (
	GameTypeNormal   GameType = "normal"
	GameTypeSpicy    GameType = "spicy"
	GameTypeUnhinged GameType = "unhinged"
)

// RoundType represents the type of round
type RoundType string

const (
	RoundTypePubQuiz  RoundType = "pub_quiz"
	RoundTypeCharades RoundType = "charades"
	RoundTypeAlibi    RoundType = "alibi"
	RoundTypeCopycat  RoundType = "copycat"
)

// GameStatus represents the current state of a game
type GameStatus string

const (
	GameStatusLobby      GameStatus = "lobby"
	GameStatusInProgress GameStatus = "in_progress"
	GameStatusFinished   GameStatus = "finished"
)

// Player represents a player in a game
type Player struct {
	ID       string
	GameID   string
	Name     string
	IsHost   bool
	Score    int
	JoinedAt time.Time
}

// Round represents a single round
type Round struct {
	ID          string
	GameID      string
	RoundNum    int
	RoundType   RoundType
	Prompt      string // for charades/copycat
	CrimeText   string // for alibi
	QuestionID  string // for pub_quiz
	ActorID     string // for charades/alibi
	StartTime   time.Time
	Duration    time.Duration // 10s, 30s, or 60s depending on type
	SubmitCount int
	Answers     map[string]int // player_id -> answer_index for pub_quiz
}

// Initialize Answers map in constructor
func (r *Round) InitAnswers() {
	r.Answers = make(map[string]int)
}

// Game represents a game session
type Game struct {
	ID            string
	Code          string
	HostID        string
	GameType      GameType
	RoundCount    int
	CurrentRound  int
	Status        GameStatus
	Players       map[string]*Player
	Rounds        []*Round
	CreatedAt     time.Time
	LastRoundType RoundType
}
