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
