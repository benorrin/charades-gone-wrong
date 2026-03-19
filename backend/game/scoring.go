package game

import "time"

// PubQuizScoreWithTimestamp calculates points for pub quiz based on:
// - Whether the answer was correct
// - When the submission was received vs when the round started
//
// Formula: BasePoints * (TimeRemaining / TotalTime)
// This rewards faster correct answers with higher points
func PubQuizScoreWithTimestamp(correct bool, roundStartTime time.Time, submissionTime int64, roundDuration time.Duration) int {
	if !correct {
		return 0
	}

	roundStartMs := roundStartTime.UnixMilli()
	roundDurationMs := roundDuration.Milliseconds()
	timeElapsedMs := submissionTime - roundStartMs
	timeRemainingMs := roundDurationMs - timeElapsedMs

	// Ensure time remaining doesn't go negative
	if timeRemainingMs < 0 {
		timeRemainingMs = 0
	}

	// Calculate points: base 1000 points scaled by speed
	basePoints := 1000
	timeMultiplier := float64(timeRemainingMs) / float64(roundDurationMs)
	points := int(float64(basePoints) * timeMultiplier)

	return points
}

// PubQuizScore calculates points for pub quiz question
// timeTaken is in seconds, returns base points + speed bonus
func PubQuizScore(correct bool, timeTaken int) int {
	if !correct {
		return 0
	}

	// 10 seconds for time limit
	speedBonus := 10 - timeTaken
	if speedBonus < 1 {
		speedBonus = 1
	}

	basePoints := 10
	return basePoints + speedBonus
}

// CharadesScore calculates points for charades round
// ratings is a map of rating to count
func CharadesScore(ratings map[string]int, responseSec int) int {
	ratingPoints := map[string]int{
		"rubbish": 1,
		"good":    3,
		"great":   5,
		"funny":   4,
		"cringe":  2,
	}

	totalPoints := 0
	for rating, count := range ratings {
		if pts, exists := ratingPoints[rating]; exists {
			totalPoints += pts * count
		}
	}

	// Speed multiplier (30 second limit)
	speedMultiplier := 1.0
	if responseSec < 10 {
		speedMultiplier = 1.5
	} else if responseSec < 20 {
		speedMultiplier = 1.2
	}

	return int(float64(totalPoints) * speedMultiplier)
}

// AlibiScore calculates points for alibi round
// believableVotes is count of "believable" votes
func AlibiScore(believableVotes int) int {
	return believableVotes * 2
}

// CopycatScore calculates points for copycat round
// isMatching indicates if player's answer matched others
func CopycatScore(isMatching bool) int {
	if isMatching {
		return 0
	}
	return 5
}

// RoundDurations returns the duration for each round type
func RoundDurations() map[RoundType]time.Duration {
	return map[RoundType]time.Duration{
		RoundTypePubQuiz:  10 * time.Second,
		RoundTypeCharades: 30 * time.Second,
		RoundTypeAlibi:    60 * time.Second,
		RoundTypeCopycat:  10 * time.Second,
	}
}

// RoundDuration returns the duration for a specific round type
func RoundDuration(rt RoundType) time.Duration {
	durations := RoundDurations()
	if d, exists := durations[rt]; exists {
		return d
	}
	return 10 * time.Second // default
}
