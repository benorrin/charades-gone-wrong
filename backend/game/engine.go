package game

import (
	"math/rand"
	"time"
)

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
	if g.Status != GameStatusInProgress {
		return nil, ErrInvalidGameStatus
	}

	if g.CurrentRound >= g.RoundCount {
		g.Status = GameStatusFinished
		return nil, ErrGameFinished
	}

	roundType := g.GetNextRoundType()

	round := &Round{
		ID:        GenerateID(),
		GameID:    g.ID,
		RoundNum:  g.CurrentRound,
		RoundType: roundType,
		StartTime: time.Now(),
	}

	// Set duration and content based on round type
	switch roundType {
	case RoundTypePubQuiz:
		round.Duration = 10 * time.Second
		// Prompt would be the question text
		round.Prompt = "Next question loading..."
	case RoundTypeCharades:
		round.Duration = 30 * time.Second
		actor := g.GetRandomPlayer()
		if actor == nil {
			return nil, ErrPlayerNotFound
		}
		round.ActorID = actor.ID
		round.Prompt = "Act out the prompt!"
	case RoundTypeAlibi:
		round.Duration = 60 * time.Second
		suspect := g.GetRandomPlayer()
		if suspect == nil {
			return nil, ErrPlayerNotFound
		}
		round.ActorID = suspect.ID
		round.CrimeText = "The crime is..." // placeholder
	case RoundTypeCopycat:
		round.Duration = 10 * time.Second
		round.Prompt = "Write your answer..."
	}

	g.Rounds = append(g.Rounds, round)
	return round, nil
}

// AdvanceRound moves to the next round
func (g *Game) AdvanceRound() {
	if g.CurrentRound < g.RoundCount {
		g.CurrentRound++
	} else {
		g.Status = GameStatusFinished
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
	players := make([]*Player, 0, len(g.Players))
	for _, p := range g.Players {
		players = append(players, p)
	}

	// Simple bubble sort (for small player counts, this is fine)
	for i := 0; i < len(players); i++ {
		for j := i + 1; j < len(players); j++ {
			if players[j].Score > players[i].Score {
				players[i], players[j] = players[j], players[i]
			}
		}
	}

	return players
}

// GenerateID generates a unique ID
func GenerateID() string {
	return time.Now().Format("20060102150405") + "_" + randString(8)
}

// Helper function for random string
func randString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
