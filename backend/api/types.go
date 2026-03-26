package api

import (
	"net/http"
	"sync"

	"game/db"
	"game/game"
	"game/ws"

	"github.com/gorilla/websocket"
)

// CreateGameRequest is sent when creating a new game
type CreateGameRequest struct {
	GameType   string `json:"game_type"`
	RoundCount int    `json:"round_count"`
}

// CreateGameResponse is returned when a game is created
type CreateGameResponse struct {
	GameID string `json:"game_id"`
	Code   string `json:"code"`
	HostID string `json:"host_id"`
}

// JoinGameRequest is sent when joining an existing game
type JoinGameRequest struct {
	Code string `json:"code"`
}

// JoinGameResponse is returned when a player joins a game
type JoinGameResponse struct {
	GameID   string `json:"game_id"`
	PlayerID string `json:"player_id"`
	Players  int    `json:"player_count"`
}

// SetPlayerNameRequest is sent when setting a player's name
type SetPlayerNameRequest struct {
	Name string `json:"name"`
}

// StartGameRequest is sent when starting a game
type StartGameRequest struct {
	GameID string `json:"game_id"`
}

// StatusResponse is a generic success response
type StatusResponse struct {
	Status string `json:"status"`
}

// Server holds all dependencies for API handlers
type Server struct {
	DB       *db.Database
	Games    map[string]*game.Game
	Hubs     map[string]*ws.Hub
	mutex    sync.RWMutex
	upgrader websocket.Upgrader
}

// NewServer creates a new API server
func NewServer(database *db.Database) *Server {
	return &Server{
		DB:    database,
		Games: make(map[string]*game.Game),
		Hubs:  make(map[string]*ws.Hub),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}
}
