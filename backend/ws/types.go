package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

// MessageType represents the type of WebSocket message
type MessageType string

// Message types for WebSocket communication
const (
	MsgTypePlayerJoined     MessageType = "player_joined"
	MsgTypeGameStarted      MessageType = "game_started"
	MsgTypeRoundStarted     MessageType = "round_started"
	MsgTypeRoundEnded       MessageType = "round_ended"
	MsgTypeScoresUpdated    MessageType = "scores_updated"
	MsgTypeLeaderboard      MessageType = "leaderboard"
	MsgTypeGameEnded        MessageType = "game_ended"
	MsgTypePlayerList       MessageType = "player_list"
	MsgTypePlayerSubmission MessageType = "player_submission"
	MsgTypeSubmissionAck    MessageType = "submission_ack"
	MsgTypeError            MessageType = "error"
)

// Message is the base message type sent over WebSocket
type Message struct {
	Type    MessageType `json:"type"`
	GameID  string      `json:"game_id"`
	Payload interface{} `json:"payload"`
}

// Client represents a WebSocket client connection
type Client struct {
	GameID   string
	PlayerID string
	Conn     *websocket.Conn
	Send     chan *Message
}

// Hub manages all WebSocket connections for a game
type Hub struct {
	GameID       string
	Clients      map[*Client]bool
	Broadcast    chan *Message
	Register     chan *Client
	Unregister   chan *Client
	Submissions  map[int]map[string]*PlayerSubmissionPayload
	RoundAnswers map[int]int
	mutex        sync.RWMutex
	done         chan struct{}
}

// PlayerJoinedPayload is sent when a player joins
type PlayerJoinedPayload struct {
	PlayerID string       `json:"player_id"`
	Name     string       `json:"name"`
	Count    int          `json:"player_count"`
	Players  []PlayerInfo `json:"players"`
}

// PlayerSubmissionPayload is sent when a player submits an answer
type PlayerSubmissionPayload struct {
	PlayerID       string `json:"player_id"`
	RoundNum       int    `json:"round_num"`
	AnswerIdx      int    `json:"answer_idx,omitempty"`      // for pub_quiz
	Rating         string `json:"rating,omitempty"`          // for charades
	VoteBelievable bool   `json:"vote_believable,omitempty"` // for alibi
	Answer         string `json:"answer,omitempty"`          // for copycat
	SubmissionTime int64  `json:"-"`                         // unix milliseconds when received by server (never from client)
}

// SubmissionAckPayload is sent to acknowledge a player's answer submission
type SubmissionAckPayload struct {
	RoundNum  int  `json:"round_num"`
	IsCorrect bool `json:"is_correct"`
	AnswerIdx int  `json:"answer_idx"`
}

// PlayerInfo contains basic player information
type PlayerInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	IsHost bool   `json:"is_host"`
	Score  int    `json:"score"`
}

// GameStartedPayload is sent when the game starts
type GameStartedPayload struct {
	RoundCount   int    `json:"round_count"`
	GameType     string `json:"game_type"`
	TotalPlayers int    `json:"total_players"`
}

// RoundStartedPayload is sent when a round starts
type RoundStartedPayload struct {
	RoundNum  int             `json:"round_num"`
	RoundType string          `json:"round_type"`
	RoundName string          `json:"round_name"`
	Duration  int             `json:"duration_seconds"`
	Prompt    string          `json:"prompt,omitempty"`
	CrimeText string          `json:"crime_text,omitempty"`
	Question  QuestionPayload `json:"question,omitempty"`
	ActorID   string          `json:"actor_id,omitempty"`
	ActorName string          `json:"actor_name,omitempty"`
}

// QuestionPayload contains a pub quiz question
type QuestionPayload struct {
	Text    string   `json:"text"`
	Answers []string `json:"answers"`
}

// RoundEndedPayload is sent when a round ends
type RoundEndedPayload struct {
	RoundNum  int          `json:"round_num"`
	RoundType string       `json:"round_type"`
	Results   RoundResults `json:"results"`
}

// RoundResults contains results based on round type
type RoundResults struct {
	CorrectAnswer     int                 `json:"correct_answer,omitempty"`       // pub_quiz
	Ratings           map[string]int      `json:"ratings,omitempty"`              // charades
	VotesBelivable    int                 `json:"votes_believable,omitempty"`     // alibi
	VotesNotBelivable int                 `json:"votes_not_believable,omitempty"` // alibi
	Answers           map[string][]string `json:"answers,omitempty"`              // copycat
}

// ScoresUpdatedPayload is sent with updated scores
type ScoresUpdatedPayload struct {
	Scores []PlayerScore `json:"scores"`
}

// PlayerScore contains a player's current score
type PlayerScore struct {
	PlayerID string `json:"player_id"`
	Name     string `json:"name"`
	Score    int    `json:"score"`
}

// LeaderboardPayload contains final leaderboard
type LeaderboardPayload struct {
	Rankings []LeaderboardEntry `json:"rankings"`
}

// LeaderboardEntry is a single leaderboard entry
type LeaderboardEntry struct {
	Rank  int    `json:"rank"`
	Name  string `json:"name"`
	Score int    `json:"score"`
}

// GameEndedPayload signals the game has ended
type GameEndedPayload struct {
	FinalLeaderboard []LeaderboardEntry `json:"final_leaderboard"`
	HostID           string             `json:"host_id"`
}

// ErrorPayload contains error information
type ErrorPayload struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}
