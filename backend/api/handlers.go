package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"game/db"
	"game/game"
	"game/ws"

	"github.com/gorilla/websocket"
)

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

// HandleCreateGame endpoint
func (s *Server) HandleCreateGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.GameType != "normal" && req.GameType != "spicy" && req.GameType != "unhinged" {
		http.Error(w, "Invalid game type", http.StatusBadRequest)
		return
	}

	if req.RoundCount < 10 || req.RoundCount > 30 {
		req.RoundCount = 20
	}

	code := game.GenerateJoinCode()
	hostID := game.GenerateID()

	g := game.NewGame(code, hostID, game.GameType(req.GameType), req.RoundCount)
	g.AddPlayer(hostID, true)

	s.mutex.Lock()
	s.Games[g.ID] = g

	hub := ws.NewHub(g.ID)
	go hub.Run()
	s.Hubs[g.ID] = hub
	s.mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateGameResponse{
		GameID: g.ID,
		Code:   code,
		HostID: hostID,
	})
}

// HandleJoinGame endpoint
func (s *Server) HandleJoinGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JoinGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	if !game.ValidateJoinCode(req.Code) {
		http.Error(w, "Invalid code format", http.StatusBadRequest)
		return
	}

	s.mutex.RLock()
	var targetGame *game.Game
	for _, g := range s.Games {
		if g.Code == req.Code {
			targetGame = g
			break
		}
	}
	s.mutex.RUnlock()

	if targetGame == nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	if targetGame.Status != game.GameStatusLobby {
		http.Error(w, "Game not accepting new players", http.StatusBadRequest)
		return
	}

	if len(targetGame.Players) >= 16 {
		http.Error(w, "Game is full", http.StatusBadRequest)
		return
	}

	playerID := game.GenerateID()
	targetGame.AddPlayer(playerID, false)

	s.mutex.RLock()
	hub := s.Hubs[targetGame.ID]
	s.mutex.RUnlock()

	if hub != nil {
		payload := ws.PlayerJoinedPayload{
			PlayerID: playerID,
			Count:    len(targetGame.Players),
			Players:  getPlayerInfos(targetGame),
		}
		hub.BroadcastMessage(ws.MsgTypePlayerJoined, payload)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(JoinGameResponse{
		GameID:   targetGame.ID,
		PlayerID: playerID,
		Players:  len(targetGame.Players),
	})
}

// HandleSetPlayerName endpoint
func (s *Server) HandleSetPlayerName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	gameID := r.URL.Query().Get("game_id")
	playerID := r.URL.Query().Get("player_id")

	if gameID == "" || playerID == "" {
		http.Error(w, "Missing game_id or player_id", http.StatusBadRequest)
		return
	}

	var req SetPlayerNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) < 1 || len(req.Name) > 50 {
		http.Error(w, "Name must be 1-50 characters", http.StatusBadRequest)
		return
	}

	s.mutex.RLock()
	g, exists := s.Games[gameID]
	s.mutex.RUnlock()

	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	if err := g.SetPlayerName(playerID, req.Name); err != nil {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}

	// Broadcast updated player list to all connected clients
	s.mutex.RLock()
	hub := s.Hubs[gameID]
	s.mutex.RUnlock()

	if hub != nil {
		payload := ws.PlayerJoinedPayload{
			PlayerID: playerID,
			Count:    len(g.Players),
			Players:  getPlayerInfos(g),
		}
		hub.BroadcastMessage(ws.MsgTypePlayerJoined, payload)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleStartGame endpoint
func (s *Server) HandleStartGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StartGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	s.mutex.RLock()
	g, exists := s.Games[req.GameID]
	s.mutex.RUnlock()

	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	if !g.CanStart() {
		http.Error(w, "Cannot start game", http.StatusBadRequest)
		return
	}

	g.Start()

	s.mutex.RLock()
	hub := s.Hubs[g.ID]
	s.mutex.RUnlock()

	if hub != nil {
		payload := ws.GameStartedPayload{
			RoundCount:   g.RoundCount,
			GameType:     string(g.GameType),
			TotalPlayers: len(g.Players),
		}
		hub.BroadcastMessage(ws.MsgTypeGameStarted, payload)

		// Start first round
		go s.startRound(g, hub)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

// startRound is a helper that starts a round and broadcasts it
func (s *Server) startRound(g *game.Game, hub *ws.Hub) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[startRound] PANIC: %v", r)
		}
	}()

	log.Printf("[startRound] Starting for game %s", g.ID)
	// Wait a bit for clients to be ready
	<-time.After(1 * time.Second)

	round, err := g.StartNextRound()
	if err != nil {
		log.Printf("[startRound] Error: %v", err)
		if err == game.ErrGameFinished {
			// Game ended, broadcast leaderboard
			leaderboard := g.GetLeaderboard()
			rankings := make([]ws.LeaderboardEntry, len(leaderboard))
			for i, p := range leaderboard {
				rankings[i] = ws.LeaderboardEntry{
					Rank:  i + 1,
					Name:  p.Name,
					Score: p.Score,
				}
			}
			log.Printf("[startRound] Broadcasting leaderboard with %d entries", len(rankings))
			hub.BroadcastMessage(ws.MsgTypeLeaderboard, ws.LeaderboardPayload{
				Rankings: rankings,
			})
		}
		return
	}
	log.Printf("[startRound] Round %d started, type: %s", round.RoundNum, round.RoundType)

	// Get player name for actor
	actorName := ""
	if round.ActorID != "" {
		if actor := g.GetPlayerByID(round.ActorID); actor != nil {
			actorName = actor.Name
		}
	}

	// Load question for pub quiz rounds
	var question *ws.QuestionPayload
	var correctAnswerIdx int
	if round.RoundType == game.RoundTypePubQuiz {
		q, err := s.DB.GetRandomPubQuizQuestion(string(g.GameType))
		if err != nil {
			log.Printf("Failed to load question: %v", err)
		} else {
			round.QuestionID = q.ID
			correctAnswerIdx = q.CorrectAnswerID
			question = &ws.QuestionPayload{
				Text:    q.Text,
				Answers: q.Answers,
			}
		}
	}

	// Broadcast round started
	var questionPayload ws.QuestionPayload
	if question != nil {
		questionPayload = *question
	}
	roundPayload := ws.RoundStartedPayload{
		RoundNum:  round.RoundNum,
		RoundType: string(round.RoundType),
		Duration:  int(round.Duration.Seconds()),
		Prompt:    round.Prompt,
		CrimeText: round.CrimeText,
		ActorID:   round.ActorID,
		ActorName: actorName,
		Question:  questionPayload,
	}

	// Store the correct answer in the hub for pub_quiz rounds (for submission checking)
	if round.RoundType == game.RoundTypePubQuiz && correctAnswerIdx >= 0 {
		hub.SetRoundAnswer(round.RoundNum, correctAnswerIdx)
		log.Printf("[startRound] Stored correct answer %d for round %d", correctAnswerIdx, round.RoundNum)
	}

	log.Printf("[startRound] Broadcasting round started message")
	hub.BroadcastMessage(ws.MsgTypeRoundStarted, roundPayload)
	log.Printf("[startRound] Round message broadcast complete, waiting %d seconds or all players answered", int(round.Duration.Seconds()))

	// Create a done channel to stop the goroutine when we're done waiting
	doneChan := make(chan struct{})

	// Start a goroutine to check for all players answering
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[startRound] PANIC in answer check goroutine for round %d: %v", round.RoundNum, r)
			}
		}()

		// Just wait for the done signal and exit
		<-doneChan
		log.Printf("[startRound] Answer check goroutine stopping for round %d", round.RoundNum)
	}()

	// Wait for intro screen duration (3 seconds) - frontend shows round info during this time
	introDelay := 3 * time.Second
	log.Printf("[startRound] Round %d: waiting %d seconds for intro screen", round.RoundNum, int(introDelay.Seconds()))
	<-time.After(introDelay)

	// Wait for round duration (10 seconds for pub_quiz)
	log.Printf("[startRound] Round %d: intro complete, waiting %d seconds for question answers", round.RoundNum, int(round.Duration.Seconds()))
	<-time.After(round.Duration)
	log.Printf("[startRound] Round %d: duration expired", round.RoundNum)

	// Signal the answer check goroutine to stop
	log.Printf("[startRound] Round %d: closing doneChan to stop answer check goroutine", round.RoundNum)
	close(doneChan)

	// Calculate scores
	log.Printf("[startRound] About to calculate scores for round %d", round.RoundNum)
	s.calculateRoundScores(g, round, hub)
	log.Printf("[startRound] Scores calculated, about to broadcast round ended")

	// End round
	hub.BroadcastMessage(ws.MsgTypeRoundEnded, ws.RoundEndedPayload{
		RoundNum:  round.RoundNum,
		RoundType: string(round.RoundType),
		Results:   ws.RoundResults{},
	})
	log.Printf("[startRound] Round ended broadcasted, about to broadcast scores")

	// Broadcast updated scores
	s.broadcastScores(g, hub)
	log.Printf("[startRound] Scores broadcasted, about to advance round")

	// Advance and start next round
	g.AdvanceRound()
	log.Printf("[startRound] Advanced to round %d, game status: %v", g.CurrentRound, g.Status)
	if g.Status == game.GameStatusInProgress {
		log.Printf("[startRound] Starting next round recursively")
		s.startRound(g, hub)
	}
}

// HandleResetGame endpoint - resets the game while keeping all players
func (s *Server) HandleResetGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		GameID string `json:"game_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	gameID := req.GameID
	if gameID == "" {
		http.Error(w, "Game ID required", http.StatusBadRequest)
		return
	}

	s.mutex.RLock()
	g, gameExists := s.Games[gameID]
	hub, hubExists := s.Hubs[gameID]
	s.mutex.RUnlock()

	if !gameExists || !hubExists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	// Reset game state but keep all players
	g.Status = game.GameStatusInProgress
	g.CurrentRound = 1

	// Reset all player scores to 0
	for _, player := range g.Players {
		player.Score = 0
	}

	log.Printf("[HandleResetGame] Game %s reset - %d players kept", gameID, len(g.Players))

	// Broadcast game started message to all connected clients
	hub.BroadcastMessage(ws.MsgTypeGameStarted, ws.GameStartedPayload{
		RoundCount:   g.RoundCount,
		GameType:     string(g.GameType),
		TotalPlayers: len(g.Players),
	})

	// Start first round after a short delay
	go func() {
		time.Sleep(1 * time.Second)
		s.startRound(g, hub)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleGetGame endpoint
func (s *Server) HandleGetGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	gameID := r.URL.Query().Get("game_id")
	if gameID == "" {
		http.Error(w, "Missing game_id", http.StatusBadRequest)
		return
	}

	s.mutex.RLock()
	g, exists := s.Games[gameID]
	s.mutex.RUnlock()

	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":            g.ID,
		"code":          g.Code,
		"status":        g.Status,
		"game_type":     g.GameType,
		"round_count":   g.RoundCount,
		"current_round": g.CurrentRound,
		"players":       getPlayerInfos(g),
	})
}

// Helper function
func getPlayerInfos(g *game.Game) []ws.PlayerInfo {
	infos := make([]ws.PlayerInfo, 0, len(g.Players))
	for _, p := range g.Players {
		infos = append(infos, ws.PlayerInfo{
			ID:     p.ID,
			Name:   p.Name,
			IsHost: p.IsHost,
			Score:  p.Score,
		})
	}
	return infos
}

// HandleWebSocket handles WebSocket connections
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	gameID := r.URL.Query().Get("game_id")
	playerID := r.URL.Query().Get("player_id")

	if gameID == "" || playerID == "" {
		http.Error(w, "Missing game_id or player_id", http.StatusBadRequest)
		return
	}

	s.mutex.RLock()
	game, gameExists := s.Games[gameID]
	hub, hubExists := s.Hubs[gameID]
	s.mutex.RUnlock()

	if !gameExists || !hubExists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	// Verify player exists in game
	if _, exists := game.Players[playerID]; !exists {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create client and register with hub
	client := &ws.Client{
		GameID:   gameID,
		PlayerID: playerID,
		Conn:     conn,
		Send:     make(chan *ws.Message, 256),
	}

	hub.Register <- client

	// Start read and write pumps
	go client.ReadPump(hub)
	go client.WritePump()
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/games/create", s.HandleCreateGame)
	mux.HandleFunc("POST /api/games/join", s.HandleJoinGame)
	mux.HandleFunc("POST /api/players/name", s.HandleSetPlayerName)
	mux.HandleFunc("POST /api/games/start", s.HandleStartGame)
	mux.HandleFunc("POST /api/games/reset", s.HandleResetGame)
	mux.HandleFunc("GET /api/games", s.HandleGetGame)
	mux.HandleFunc("GET /ws", s.HandleWebSocket)
}

// calculateRoundScores calculates and updates player scores based on round results
func (s *Server) calculateRoundScores(g *game.Game, round *game.Round, hub *ws.Hub) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[calculateRoundScores PANIC] Round %d: %v", round.RoundNum, r)
		}
	}()

	log.Printf("[calculateRoundScores] Getting submissions for round %d", round.RoundNum)
	submissions := hub.GetRoundSubmissions(round.RoundNum)
	log.Printf("[calculateRoundScores] Got %d submissions", len(submissions))

	switch round.RoundType {
	case game.RoundTypePubQuiz:
		log.Printf("[calculateRoundScores] Processing pub quiz with question ID: %s", round.QuestionID)
		if round.QuestionID == "" {
			log.Println("[calculateRoundScores] No question ID for pub quiz round")
			return
		}

		log.Printf("[calculateRoundScores] Querying database for question %s", round.QuestionID)
		q, err := s.DB.GetPubQuizQuestionByID(round.QuestionID)
		if err != nil {
			log.Printf("[calculateRoundScores] Failed to get question: %v", err)
			return
		}
		log.Printf("[calculateRoundScores] Got question: %s, correct answer: %d", q.Text, q.CorrectAnswerID)

		// Score pub quiz answers with time-based multiplier
		for playerID, submission := range submissions {
			if player, exists := g.Players[playerID]; exists {
				// Calculate score using the scoring function
				points := game.PubQuizScoreWithTimestamp(
					submission.AnswerIdx == q.CorrectAnswerID,
					round.StartTime,
					submission.SubmissionTime,
					round.Duration,
				)

				if submission.AnswerIdx == q.CorrectAnswerID {
					// Log correct answers with timing details
					roundStartMs := round.StartTime.UnixMilli()
					timeElapsedMs := submission.SubmissionTime - roundStartMs
					roundDurationMs := round.Duration.Milliseconds()
					timeRemainingMs := roundDurationMs - timeElapsedMs
					if timeRemainingMs < 0 {
						timeRemainingMs = 0
					}
					timeMultiplier := float64(timeRemainingMs) / float64(roundDurationMs)

					log.Printf("[calculateRoundScores] Player %s: CORRECT, time: %dms / %dms, multiplier: %.2f, points: %d",
						player.Name, timeElapsedMs, roundDurationMs, timeMultiplier, points)
				} else {
					log.Printf("[calculateRoundScores] Player %s: INCORRECT (chose %d, correct %d), 0 points",
						player.Name, submission.AnswerIdx, q.CorrectAnswerID)
				}

				player.Score += points
			}
		}
		log.Printf("[calculateRoundScores] Pub quiz scoring complete")
	}
}

// broadcastScores broadcasts the current scores to all players
func (s *Server) broadcastScores(g *game.Game, hub *ws.Hub) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[broadcastScores PANIC]: %v", r)
		}
	}()

	log.Printf("[broadcastScores] Building scores for %d players", len(g.Players))
	scores := make([]ws.PlayerScore, 0, len(g.Players))
	for _, player := range g.Players {
		scores = append(scores, ws.PlayerScore{
			PlayerID: player.ID,
			Name:     player.Name,
			Score:    player.Score,
		})
	}

	log.Printf("[broadcastScores] Broadcasting %d scores", len(scores))
	hub.BroadcastMessage(ws.MsgTypeScoresUpdated, ws.ScoresUpdatedPayload{
		Scores: scores,
	})
	log.Printf("[broadcastScores] Scores broadcast complete")
}
