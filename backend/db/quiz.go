package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// PubQuizQuestion represents a quiz question with answers
type PubQuizQuestion struct {
	ID               string   `json:"id"`
	Text             string   `json:"text"`
	Answers          []string `json:"answers"`
	CorrectAnswerID  int      `json:"correct_answer_idx"`
	GameType         string   `json:"game_type"`
	IsPlayerSpecific bool     `json:"is_player_specific"`
}

// GetRandomPubQuizQuestion returns a random pub quiz question for the given game type
func (db *Database) GetRandomPubQuizQuestion(gameType string) (*PubQuizQuestion, error) {
	var id, text, answersJSON, gType string
	var correctIdx int
	var isPlayerSpecific bool

	err := db.conn.QueryRow(`
		SELECT id, text, answers, correct_answer_idx, game_type, is_player_specific
		FROM pub_quiz_questions
		WHERE game_type = ?
		ORDER BY RANDOM()
		LIMIT 1
	`, gameType).Scan(&id, &text, &answersJSON, &correctIdx, &gType, &isPlayerSpecific)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no pub quiz questions found for game type: %s", gameType)
		}
		return nil, err
	}

	var answers []string
	if err := json.Unmarshal([]byte(answersJSON), &answers); err != nil {
		return nil, fmt.Errorf("failed to parse answers: %w", err)
	}

	return &PubQuizQuestion{
		ID:               id,
		Text:             text,
		Answers:          answers,
		CorrectAnswerID:  correctIdx,
		GameType:         gType,
		IsPlayerSpecific: isPlayerSpecific,
	}, nil
}

// GetPubQuizQuestionByID returns a specific pub quiz question
func (db *Database) GetPubQuizQuestionByID(questionID string) (*PubQuizQuestion, error) {
	var id, text, answersJSON, gType string
	var correctIdx int
	var isPlayerSpecific bool

	err := db.conn.QueryRow(`
		SELECT id, text, answers, correct_answer_idx, game_type, is_player_specific
		FROM pub_quiz_questions
		WHERE id = ?
	`, questionID).Scan(&id, &text, &answersJSON, &correctIdx, &gType, &isPlayerSpecific)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pub quiz question not found: %s", questionID)
		}
		return nil, err
	}

	var answers []string
	if err := json.Unmarshal([]byte(answersJSON), &answers); err != nil {
		return nil, fmt.Errorf("failed to parse answers: %w", err)
	}

	return &PubQuizQuestion{
		ID:               id,
		Text:             text,
		Answers:          answers,
		CorrectAnswerID:  correctIdx,
		GameType:         gType,
		IsPlayerSpecific: isPlayerSpecific,
	}, nil
}
