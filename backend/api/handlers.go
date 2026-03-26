package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"game/game"
	"game/logger"
	"game/ws"
)

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

	if !game.IsValidGameType(req.GameType) {
		http.Error(w, "Invalid game type", http.StatusBadRequest)
		return
	}

	req.RoundCount = game.NormalizeRoundCount(req.RoundCount)

	g, code, hostID := s.createAndRegisterGame(req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateGameResponse{
		GameID: g.ID,
		Code:   code,
		HostID: hostID,
	})
}

// createAndRegisterGame creates a new game and registers it in the server
func (s *Server) createAndRegisterGame(req CreateGameRequest) (*game.Game, string, string) {
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

	return g, code, hostID
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

	targetGame, ok := s.findAndValidateGame(req.Code, w)
	if !ok {
		return
	}

	playerID := s.addPlayerToGame(targetGame, w)
	if playerID == "" {
		return
	}

	s.broadcastPlayerJoined(targetGame, playerID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(JoinGameResponse{
		GameID:   targetGame.ID,
		PlayerID: playerID,
		Players:  len(targetGame.Players),
	})
}

// findAndValidateGame finds a game by code and validates it's accepting players
func (s *Server) findAndValidateGame(code string, w http.ResponseWriter) (*game.Game, bool) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if !game.ValidateJoinCode(code) {
		http.Error(w, "Invalid code format", http.StatusBadRequest)
		return nil, false
	}

	s.mutex.RLock()
	var targetGame *game.Game
	for _, g := range s.Games {
		if g.Code == code {
			targetGame = g
			break
		}
	}
	s.mutex.RUnlock()

	if targetGame == nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return nil, false
	}

	if targetGame.Status != game.GameStatusLobby {
		http.Error(w, "Game not accepting new players", http.StatusBadRequest)
		return nil, false
	}

	return targetGame, true
}

// addPlayerToGame adds a new player to a game, returning the player ID
func (s *Server) addPlayerToGame(targetGame *game.Game, w http.ResponseWriter) string {
	if len(targetGame.Players) >= game.MaxPlayersPerGame {
		http.Error(w, "Game is full", http.StatusBadRequest)
		return ""
	}

	playerID := game.GenerateID()
	targetGame.AddPlayer(playerID, false)
	return playerID
}

// broadcastPlayerJoined broadcasts that a player joined
func (s *Server) broadcastPlayerJoined(targetGame *game.Game, playerID string) {
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

	g, ok := s.getGame(gameID, w)
	if !ok {
		return
	}

	if err := g.SetPlayerName(playerID, req.Name); err != nil {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}

	s.broadcastPlayerList(gameID, playerID, g)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// broadcastPlayerList broadcasts the updated player list
func (s *Server) broadcastPlayerList(gameID, playerID string, g *game.Game) {
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

	g, ok := s.getGame(req.GameID, w)
	if !ok {
		return
	}

	if !g.CanStart() {
		http.Error(w, "Cannot start game", http.StatusBadRequest)
		return
	}

	g.Start()
	s.startGameRound(g)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

// startGameRound broadcasts game start and begins the first round
func (s *Server) startGameRound(g *game.Game) {
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
}

// getGame retrieves a game by ID from the server
func (s *Server) getGame(gameID string, w http.ResponseWriter) (*game.Game, bool) {
	s.mutex.RLock()
	g, exists := s.Games[gameID]
	s.mutex.RUnlock()

	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return nil, false
	}
	return g, true
}

// startRound is a helper that starts a round and broadcasts it
func (s *Server) startRound(g *game.Game, hub *ws.Hub) {
	defer func() {
		if r := recover(); r != nil {
			logger.Panic("startRound", "Panic in round handler", r)
		}
	}()

	logger.Debug("startRound", "Starting for game %s", g.ID)
	// Wait a bit for clients to be ready
	<-time.After(1 * time.Second)

	round, err := g.StartNextRound()
	if err != nil {
		s.handleRoundStartError(err, g, hub)
		return
	}

	logger.Debug("startRound", "Round %d started, type: %s", round.RoundNum, round.RoundType)

	// Setup and broadcast round
	s.setupAndBroadcastRound(round, g, hub)

	// Wait for round to complete
	s.waitForRoundCompletion(round)

	// Calculate and broadcast results
	s.calculateAndBroadcastRoundResults(g, round, hub)

	// Advance and potentially start next round
	g.AdvanceRound()
	if g.Status == game.GameStatusInProgress {
		s.startRound(g, hub)
	}
}

// handleRoundStartError handles errors when starting a round
func (s *Server) handleRoundStartError(err error, g *game.Game, hub *ws.Hub) {
	logger.Error("startRound", "Error: %v", err)
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
		logger.Debug("startRound", "Broadcasting leaderboard with %d entries", len(rankings))
		hub.BroadcastMessage(ws.MsgTypeLeaderboard, ws.LeaderboardPayload{
			Rankings: rankings,
		})
	}
}

// setupAndBroadcastRound loads question data and broadcasts round to players
func (s *Server) setupAndBroadcastRound(round *game.Round, g *game.Game, hub *ws.Hub) {
	// Get player name for actor
	actorName := s.getActorName(round, g)

	// Load question for pub quiz rounds and store correct answer in hub
	question := s.loadQuestionIfNeeded(round, g, hub)

	// Build and broadcast round payload
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

	logger.Debug("startRound", "Broadcasting round started message")
	hub.BroadcastMessage(ws.MsgTypeRoundStarted, roundPayload)
}

// getActorName retrieves the actor's name for the current round
func (s *Server) getActorName(round *game.Round, g *game.Game) string {
	if round.ActorID != "" {
		if actor := g.GetPlayerByID(round.ActorID); actor != nil {
			return actor.Name
		}
	}
	return ""
}

// loadQuestionIfNeeded loads a question for pub quiz rounds
func (s *Server) loadQuestionIfNeeded(round *game.Round, g *game.Game, hub *ws.Hub) *ws.QuestionPayload {
	if round.RoundType != game.RoundTypePubQuiz {
		return nil
	}

	q, err := s.DB.GetRandomPubQuizQuestion(string(g.GameType))
	if err != nil {
		logger.Error("startRound", "Failed to load question: %v", err)
		return nil
	}

	round.QuestionID = q.ID
	correctAnswerIdx := q.CorrectAnswerID

	logger.Debug("startRound", "Loaded question %s (type: %s), correct answer index: %d", q.ID, g.GameType, correctAnswerIdx)

	// Store the correct answer in hub for WebSocket validation of submissions
	if correctAnswerIdx >= 0 {
		hub.SetRoundAnswer(round.RoundNum, correctAnswerIdx)
		logger.Debug("startRound", "Stored correct answer %d for round %d in hub", correctAnswerIdx, round.RoundNum)
	} else {
		logger.Error("startRound", "Invalid correct answer index %d for question %s", correctAnswerIdx, q.ID)
	}

	return &ws.QuestionPayload{
		Text:    q.Text,
		Answers: q.Answers,
	}
}

// waitForRoundCompletion waits for the round duration with intro delay
func (s *Server) waitForRoundCompletion(round *game.Round) {
	// Wait for intro screen duration - frontend shows round info during this time
	logger.Debug("startRound", "Round %d: waiting %d seconds for intro screen", round.RoundNum, int(game.RoundIntroDelay.Seconds()))
	<-time.After(game.RoundIntroDelay)

	// Wait for round duration
	logger.Debug("startRound", "Round %d: intro complete, waiting %d seconds for answers", round.RoundNum, int(round.Duration.Seconds()))
	<-time.After(round.Duration)
	logger.Debug("startRound", "Round %d: duration expired", round.RoundNum)
}

// calculateAndBroadcastRoundResults scores the round and broadcasts results
func (s *Server) calculateAndBroadcastRoundResults(g *game.Game, round *game.Round, hub *ws.Hub) {
	logger.Debug("startRound", "Calculating scores for round %d", round.RoundNum)
	s.calculateRoundScores(g, round, hub)

	// End round
	hub.BroadcastMessage(ws.MsgTypeRoundEnded, ws.RoundEndedPayload{
		RoundNum:  round.RoundNum,
		RoundType: string(round.RoundType),
		Results:   ws.RoundResults{},
	})
	logger.Debug("startRound", "Round ended broadcasted")

	// Broadcast updated scores
	s.broadcastScores(g, hub)
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

	g, hub, ok := s.getGameAndHub(gameID, w)
	if !ok {
		return
	}

	g.Reset()
	logger.Info("HandleResetGame", "Game %s reset - %d players kept", g.ID, len(g.Players))
	s.broadcastGameReset(g, hub)
	s.startGameRound(g)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// getGameAndHub retrieves both the game and its hub
func (s *Server) getGameAndHub(gameID string, w http.ResponseWriter) (*game.Game, *ws.Hub, bool) {
	s.mutex.RLock()
	g, gameExists := s.Games[gameID]
	hub, hubExists := s.Hubs[gameID]
	s.mutex.RUnlock()

	if !gameExists || !hubExists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return nil, nil, false
	}
	return g, hub, true
}

// broadcastGameReset broadcasts game reset to all players
func (s *Server) broadcastGameReset(g *game.Game, hub *ws.Hub) {
	hub.BroadcastMessage(ws.MsgTypeGameStarted, ws.GameStartedPayload{
		RoundCount:   g.RoundCount,
		GameType:     string(g.GameType),
		TotalPlayers: len(g.Players),
	})
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
		logger.ErrorWithErr("HandleWebSocket", "WebSocket upgrade error", err)
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
			logger.Panic("calculateRoundScores", "Panic in scoring", r)
		}
	}()

	logger.Debug("calculateRoundScores", "Getting submissions for round %d", round.RoundNum)
	submissions := hub.GetRoundSubmissions(round.RoundNum)
	logger.Debug("calculateRoundScores", "Got %d submissions", len(submissions))

	switch round.RoundType {
	case game.RoundTypePubQuiz:
		s.calculatePubQuizScores(g, round, submissions)
	}
}

// calculatePubQuizScores scores pub quiz round answers with time-based multiplier
func (s *Server) calculatePubQuizScores(g *game.Game, round *game.Round, submissions map[string]*ws.PlayerSubmissionPayload) {
	logger.Debug("calculateRoundScores", "Processing pub quiz with question ID: %s", round.QuestionID)
	if round.QuestionID == "" {
		logger.Error("calculateRoundScores", "No question ID for pub quiz round")
		return
	}

	logger.Debug("calculateRoundScores", "Querying database for question %s", round.QuestionID)
	q, err := s.DB.GetPubQuizQuestionByID(round.QuestionID)
	if err != nil {
		logger.ErrorWithErr("calculateRoundScores", "Failed to get question", err)
		return
	}
	logger.Debug("calculateRoundScores", "Got question: %s, correct answer: %d", q.Text, q.CorrectAnswerID)

	// Score all pub quiz submissions using centralized scoring logic
	game.ScorePubQuizRound(g, round, submissions, q)

	// Log scoring summary
	for playerID, submission := range submissions {
		if player, exists := g.Players[playerID]; exists {
			isCorrect := game.ValidatePubQuizAnswer(submission.AnswerIdx, q.CorrectAnswerID)
			if isCorrect {
				logger.Debug("calculateRoundScores", "Player %s answered correctly, score: %d", player.Name, player.Score)
			}
		}
	}
	logger.Debug("calculateRoundScores", "Pub quiz scoring complete")
}

// broadcastScores broadcasts the current scores to all players
func (s *Server) broadcastScores(g *game.Game, hub *ws.Hub) {
	defer func() {
		if r := recover(); r != nil {
			logger.Panic("broadcastScores", "Panic in broadcast", r)
		}
	}()

	logger.Debug("broadcastScores", "Building scores for %d players", len(g.Players))
	scores := make([]ws.PlayerScore, 0, len(g.Players))
	for _, player := range g.Players {
		scores = append(scores, ws.PlayerScore{
			PlayerID: player.ID,
			Name:     player.Name,
			Score:    player.Score,
		})
	}

	logger.Debug("broadcastScores", "Broadcasting %d scores", len(scores))
	hub.BroadcastMessage(ws.MsgTypeScoresUpdated, ws.ScoresUpdatedPayload{
		Scores: scores,
	})
	logger.Debug("broadcastScores", "Scores broadcast complete")
}
