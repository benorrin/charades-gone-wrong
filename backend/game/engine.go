package game

import (
	"math/rand"
	"time"
)

// Game constants
const (
	MaxPlayersPerGame = 16
	MinRoundCount     = 10
	MaxRoundCount     = 30
	RoundIntroDelay   = 3 * time.Second
	DefaultRoundCount = 20
)

// IsValidGameType checks if the game type is valid
func IsValidGameType(gameType string) bool {
	validTypes := []string{"normal", "spicy", "unhinged"}
	for _, valid := range validTypes {
		if gameType == valid {
			return true
		}
	}
	return false
}

// NormalizeRoundCount ensures round count is within valid range
func NormalizeRoundCount(roundCount int) int {
	if roundCount < MinRoundCount || roundCount > MaxRoundCount {
		return DefaultRoundCount
	}
	return roundCount
}

// NewGame creates a new game
func NewGame(code string, hostID string, gameType GameType, roundCount int) *Game {
	return &Game{
		ID:           GenerateID(),
		Code:         code,
		HostID:       hostID,
		GameType:     gameType,
		RoundCount:   roundCount,
		CurrentRound: 0,
		Status:       GameStatusLobby,
		Players:      make(map[string]*Player),
		Rounds:       make([]*Round, 0),
		CreatedAt:    time.Now(),
	}
}

// AddPlayer adds a player to the game
func (g *Game) AddPlayer(playerID string, isHost bool) *Player {
	player := &Player{
		ID:       playerID,
		GameID:   g.ID,
		IsHost:   isHost,
		Score:    0,
		JoinedAt: time.Now(),
	}
	g.Players[playerID] = player
	return player
}

// SetPlayerName updates a player's name
func (g *Game) SetPlayerName(playerID string, name string) error {
	if player, exists := g.Players[playerID]; exists {
		player.Name = name
		return nil
	}
	return ErrPlayerNotFound
}

// CanStart checks if the game can be started (host + at least 2 players)
func (g *Game) CanStart() bool {
	if g.Status != GameStatusLobby || len(g.Players) < 2 {
		return false
	}
	return true
}

// Start begins the game
func (g *Game) Start() error {
	if !g.CanStart() {
		return ErrCannotStartGame
	}
	g.Status = GameStatusInProgress
	g.CurrentRound = 1
	return nil
}

// StartNextRound starts the next round and returns the round details
func (g *Game) StartNextRound() (*Round, error) {
	if err := g.validateRoundStart(); err != nil {
		return nil, err
	}

	roundType := g.GetNextRoundType()
	round := g.createRound(roundType)

	if err := g.setupRoundByType(round); err != nil {
		return nil, err
	}

	g.Rounds = append(g.Rounds, round)
	return round, nil
}

// validateRoundStart checks if a round can be started
func (g *Game) validateRoundStart() error {
	if g.Status != GameStatusInProgress {
		return ErrInvalidGameStatus
	}

	if g.CurrentRound >= g.RoundCount {
		g.Status = GameStatusFinished
		return ErrGameFinished
	}

	return nil
}

// createRound creates a new round with basic info
func (g *Game) createRound(roundType RoundType) *Round {
	return &Round{
		ID:        GenerateID(),
		GameID:    g.ID,
		RoundNum:  g.CurrentRound,
		RoundType: roundType,
		StartTime: time.Now(),
	}
}

// setupRoundByType configures round-specific properties based on type
func (g *Game) setupRoundByType(round *Round) error {
	switch round.RoundType {
	case RoundTypePubQuiz:
		g.setupPubQuizRound(round)
	case RoundTypeCharades:
		return g.setupCharadesRound(round)
	case RoundTypeAlibi:
		return g.setupAlibiRound(round)
	case RoundTypeCopycat:
		g.setupCopycatRound(round)
	}
	return nil
}

// setupPubQuizRound configures a pub quiz round
func (g *Game) setupPubQuizRound(round *Round) {
	round.Duration = 10 * time.Second
	round.Prompt = "Next question loading..."
}

// setupCharadesRound configures a charades round
func (g *Game) setupCharadesRound(round *Round) error {
	round.Duration = 30 * time.Second
	actor := g.GetRandomPlayer()
	if actor == nil {
		return ErrPlayerNotFound
	}
	round.ActorID = actor.ID
	round.Prompt = "Act out the prompt!"
	return nil
}

// setupAlibiRound configures an alibi round
func (g *Game) setupAlibiRound(round *Round) error {
	round.Duration = 60 * time.Second
	suspect := g.GetRandomPlayer()
	if suspect == nil {
		return ErrPlayerNotFound
	}
	round.ActorID = suspect.ID
	round.CrimeText = "The crime is..."
	return nil
}

// setupCopycatRound configures a copycat round
func (g *Game) setupCopycatRound(round *Round) {
	round.Duration = 10 * time.Second
	round.Prompt = "Write your answer..."
}

// AdvanceRound moves to the next round
func (g *Game) AdvanceRound() {
	if g.CurrentRound < g.RoundCount {
		g.CurrentRound++
	} else {
		g.Status = GameStatusFinished
	}
}

// Reset resets the game state while keeping all players
func (g *Game) Reset() {
	g.Status = GameStatusInProgress
	g.CurrentRound = 1

	// Reset all player scores to 0
	for _, player := range g.Players {
		player.Score = 0
	}
}

// GetNextRoundType returns the next round type (avoiding consecutive repeats)
func (g *Game) GetNextRoundType() RoundType {
	// TODO: For now, only pub quiz rounds for testing
	return RoundTypePubQuiz

	/*
		roundTypes := []RoundType{
			RoundTypePubQuiz,
			RoundTypeCharades,
			RoundTypeAlibi,
			RoundTypeCopycat,
		}

		// Filter out last round type if it exists
		available := roundTypes
		if g.LastRoundType != "" {
			filtered := make([]RoundType, 0)
			for _, rt := range roundTypes {
				if rt != g.LastRoundType {
					filtered = append(filtered, rt)
				}
			}
			available = filtered
		}

		// Pick random from available
		chosen := available[rand.Intn(len(available))]
		g.LastRoundType = chosen
		return chosen
	*/
}

// GetRandomPlayer returns a random player (for charades/alibi actor selection)
func (g *Game) GetRandomPlayer() *Player {
	if len(g.Players) == 0 {
		return nil
	}

	players := make([]*Player, 0, len(g.Players))
	for _, p := range g.Players {
		players = append(players, p)
	}

	return players[rand.Intn(len(players))]
}

// GetPlayerByID retrieves a player by ID
func (g *Game) GetPlayerByID(playerID string) *Player {
	return g.Players[playerID]
}

// GetLeaderboard returns sorted players by score (highest first)
func (g *Game) GetLeaderboard() []*Player {
	players := g.buildPlayerList()
	g.sortPlayersByScore(players)
	return players
}

// buildPlayerList converts players map to a slice
func (g *Game) buildPlayerList() []*Player {
	players := make([]*Player, 0, len(g.Players))
	for _, p := range g.Players {
		players = append(players, p)
	}
	return players
}

// sortPlayersByScore sorts players by score in descending order
func (g *Game) sortPlayersByScore(players []*Player) {
	// Simple bubble sort (for small player counts, this is fine)
	for i := 0; i < len(players); i++ {
		for j := i + 1; j < len(players); j++ {
			if players[j].Score > players[i].Score {
				players[i], players[j] = players[j], players[i]
			}
		}
	}
}
