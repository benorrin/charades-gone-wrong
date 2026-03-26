package game

import "errors"

// Error definitions
var (
	ErrPlayerNotFound      = errors.New("player not found")
	ErrCannotStartGame     = errors.New("cannot start game: need at least 2 players")
	ErrGameNotInProgress   = errors.New("game is not in progress")
	ErrInvalidAnswer       = errors.New("invalid answer index")
	ErrGameFinished        = errors.New("game has finished")
	ErrInvalidGameStatus   = errors.New("invalid game status")
	ErrDuplicatePlayerName = errors.New("player name already taken in this lobby")
)
