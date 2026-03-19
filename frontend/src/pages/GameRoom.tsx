import { useSearchParams, useNavigate } from 'react-router-dom'
import { useEffect, useState } from 'react'
import confetti from 'canvas-confetti'
import { useWebSocket } from '../hooks/useWebSocket'
import './GameRoom.css'

interface PubQuizRound {
  round_num: number
  round_type: string
  duration_seconds: number
  question: {
    text: string
    answers: string[]
  }
}

interface PlayerScore {
  player_id: string
  name: string
  score: number
}

interface LeaderboardEntry {
  rank: number
  name: string
  score: number
}

export default function GameRoom() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const gameId = searchParams.get('game')
  const playerId = searchParams.get('player')
  const isHost = localStorage.getItem('isHost') === 'true'

  const [isCheckingGameStatus, setIsCheckingGameStatus] = useState(true)
  const [validatedGameId, setValidatedGameId] = useState<string>('')
  const [validatedPlayerId, setValidatedPlayerId] = useState<string>('')

  // Only establish WebSocket after game is validated
  const { isConnected, sendMessage, onMessage } = useWebSocket(validatedGameId, validatedPlayerId)

  const [currentRound, setCurrentRound] = useState<PubQuizRound | null>(null)
  const [timeLeft, setTimeLeft] = useState<number>(0)
  const [selectedAnswer, setSelectedAnswer] = useState<number | null>(null)
  const [hasAnswered, setHasAnswered] = useState(false)
  const [scores, setScores] = useState<PlayerScore[]>([])
  const [gameStarted, setGameStarted] = useState(false)
  const [roundEnded, setRoundEnded] = useState(false)
  const [gameEnded, setGameEnded] = useState(false)
  const [finalLeaderboard, setFinalLeaderboard] = useState<LeaderboardEntry[]>([])
  const [connectionError, setConnectionError] = useState<string>('')
  const [showIntro, setShowIntro] = useState(false)
  const [isAnswerCorrect, setIsAnswerCorrect] = useState<boolean | null>(null)
  const [showAnswerResult, setShowAnswerResult] = useState(false)

  // Check if game is valid before establishing WebSocket
  useEffect(() => {
    if (!gameId || !playerId) {
      setIsCheckingGameStatus(false)
      return
    }

    const checkGameStatus = async () => {
      try {
        // Set a timeout for the fetch request (5 seconds)
        const controller = new AbortController()
        const timeoutId = setTimeout(() => controller.abort(), 5000)

        const response = await fetch(`/api/games?game_id=${gameId}`, {
          signal: controller.signal,
        })
        clearTimeout(timeoutId)
        
        // If game doesn't exist (404) or any other error, redirect home
        if (!response.ok) {
          navigate('/')
          return
        }

        const gameData = await response.json()
        
        // If game is finished, redirect to home
        if (gameData.status === 'Finished') {
          navigate('/')
          return
        }

        // Game is valid - now establish the WebSocket connection
        setValidatedGameId(gameId)
        setValidatedPlayerId(playerId)
        setIsCheckingGameStatus(false)
      } catch (error) {
        console.error('Failed to check game status:', error)
        // On error, redirect home to be safe
        navigate('/')
      }
    }

    checkGameStatus()
  }, [gameId, playerId, navigate])

  // Listen for game_started message
  useEffect(() => {
    // Game is started if we navigated here from Lobby
    setGameStarted(true)
    
    onMessage('game_started', (payload) => {
      console.log('Game started:', payload)
      setGameStarted(true)
      
      // Reset all game state for new game (important for when host clicks Play Again)
      setGameEnded(false)
      setCurrentRound(null)
      setScores([])
      setFinalLeaderboard([])
      setTimeLeft(0)
      setSelectedAnswer(null)
      setHasAnswered(false)
      setRoundEnded(false)
      setShowIntro(false)
      setIsAnswerCorrect(null)
      setShowAnswerResult(false)
    })
  }, [onMessage])

  // Handle round started
  useEffect(() => {
    onMessage('round_started', (payload) => {
      console.log('Round started:', payload)
      console.log('Question:', payload.question)
      setCurrentRound(payload)
      setTimeLeft(payload.duration_seconds)
      setSelectedAnswer(null)
      setHasAnswered(false)
      setRoundEnded(false)
      setShowIntro(true)
      setIsAnswerCorrect(null)
      setShowAnswerResult(false)
    })
  }, [onMessage])

  // Handle intro screen auto-dismiss (3 seconds)
  useEffect(() => {
    if (!showIntro) return

    const introTimer = setTimeout(() => {
      setShowIntro(false)
    }, 3000)

    return () => clearTimeout(introTimer)
  }, [showIntro])

  // Handle answer submission acknowledgement
  useEffect(() => {
    onMessage('submission_ack', (payload) => {
      console.log('Submission ack:', payload)
      if (payload.round_num === currentRound?.round_num) {
        setIsAnswerCorrect(payload.is_correct)
        setShowAnswerResult(true)
      }
    })
  }, [onMessage, currentRound])

  // Handle round ended
  useEffect(() => {
    onMessage('round_ended', (payload) => {
      console.log('Round ended:', payload)
      setRoundEnded(true)
      // Feedback was already shown via submission_ack
    })
  }, [onMessage])

  // Handle scores updated
  useEffect(() => {
    onMessage('scores_updated', (payload) => {
      console.log('Scores updated:', payload)
      setScores(payload.scores || [])
    })
  }, [onMessage])

  // Handle leaderboard/game end
  useEffect(() => {
    onMessage('leaderboard', (payload) => {
      console.log('Game ended - leaderboard:', payload)
      setFinalLeaderboard(payload.rankings || [])
      setGameEnded(true)
      // No auto-redirect - user can now choose to go back or start new game
    })
  }, [onMessage, navigate])

  // Trigger confetti on game end
  useEffect(() => {
    if (!gameEnded) return

    // Fire confetti celebration
    confetti({
      particleCount: 100,
      spread: 70,
      origin: { y: 0.6 },
      colors: ['#EC4899', '#A855F7', '#06B6D4', '#22C55E', '#F97316'],
    })

    // Second burst
    setTimeout(() => {
      confetti({
        particleCount: 80,
        spread: 60,
        origin: { x: 0.1, y: 0.5 },
        colors: ['#EC4899', '#A855F7', '#06B6D4'],
      })
    }, 200)

    // Third burst
    setTimeout(() => {
      confetti({
        particleCount: 80,
        spread: 60,
        origin: { x: 0.9, y: 0.5 },
        colors: ['#22C55E', '#F97316', '#06B6D4'],
      })
    }, 400)
  }, [gameEnded])

  // Timer effect - only count down after intro screen is dismissed
  // Timer keeps counting even after player answers
  useEffect(() => {
    if (!currentRound || timeLeft <= 0 || showIntro) return

    const timer = setTimeout(() => {
      setTimeLeft(timeLeft - 1)
    }, 1000)

    return () => clearTimeout(timer)
  }, [timeLeft, currentRound, showIntro])

  // Connection timeout
  useEffect(() => {
    if (isConnected || !gameId || !playerId) return

    const timeout = setTimeout(() => {
      setConnectionError('Failed to connect to game server. Please refresh and try again.')
    }, 5000)

    return () => clearTimeout(timeout)
  }, [isConnected, gameId, playerId])

  // Handle answer selection
  const handleAnswerClick = (answerIdx: number) => {
    console.log(`[GameRoom] Answer clicked: ${answerIdx}, hasAnswered: ${hasAnswered}, currentRound: ${currentRound?.round_num}, timeLeft: ${timeLeft}`)
    if (hasAnswered || !currentRound || timeLeft <= 0) {
      console.warn(`[GameRoom] Answer click ignored. hasAnswered=${hasAnswered}, currentRound=${!!currentRound}, timeLeft=${timeLeft}`)
      return
    }

    console.log(`[GameRoom] Submitting answer ${answerIdx} for round ${currentRound.round_num}`)
    setSelectedAnswer(answerIdx)
    setHasAnswered(true)

    // Send answer via WebSocket
    sendMessage('player_submission', {
      round_num: currentRound.round_num,
      answer_idx: answerIdx,
    })
  }

  // Handle starting a new game (reset game, keep same players/lobby)
  const handleStartNewGame = async () => {
    try {
      const response = await fetch('/api/games/reset', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ game_id: gameId }),
      })

      if (response.ok) {
        // Reset UI state
        setGameEnded(false)
        setCurrentRound(null)
        setScores([])
        setFinalLeaderboard([])
        setTimeLeft(0)
        setSelectedAnswer(null)
        setHasAnswered(false)
        setRoundEnded(false)
        setShowIntro(false)
        setIsAnswerCorrect(null)
        setShowAnswerResult(false)
        setCorrectAnswer(null)
        setCorrectAnswer(null)
      }
    } catch (error) {
      console.error('Failed to start new game:', error)
    }
  }

  // Handle going back to home
  const handleBackToHome = () => {
    navigate('/')
  }

  if (!gameId || !playerId) {
    return <div className="game-room">Invalid game session</div>
  }

  // While checking game status, don't render anything (prevents blank screen)
  if (isCheckingGameStatus) {
    return null
  }

  if (connectionError) {
    return (
      <div className="game-room">
        <div className="error-container">
          <h2>Connection Error</h2>
          <p>{connectionError}</p>
          <button onClick={handleBackToHome}>Back to Home</button>
        </div>
      </div>
    )
  }

  if (!isConnected) {
    return (
      <div className="game-room">
        <div className="loading-container">
          <h2>Connecting to game...</h2>
          <div className="spinner"></div>
        </div>
      </div>
    )
  }

  if (!gameStarted) {
    return (
      <div className="game-room">
        <div className="loading-container">
          <h2>Waiting for game to start...</h2>
          <div className="spinner"></div>
        </div>
      </div>
    )
  }

  if (gameEnded) {
    return (
      <div className="game-room leaderboard-view">
        <div className="leaderboard-container">
          <h1>🏆 Game Over!</h1>
          <div className="leaderboard">
            {finalLeaderboard.map((entry, idx) => (
              <div 
                key={idx} 
                className={`leaderboard-entry rank-${entry.rank} ${
                  entry.rank === 1 ? 'winner' : ''
                }`}
              >
                <span className="rank-number">#{entry.rank}</span>
                <span className="rank-medal">
                  {entry.rank === 1 && '🥇'}
                  {entry.rank === 2 && '🥈'}
                  {entry.rank === 3 && '🥉'}
                </span>
                <span className="rank-name">{entry.name}</span>
                <span className="rank-score">{entry.score} points</span>
              </div>
            ))}
          </div>
          <div className="game-end-actions">
            {isHost && (
              <button className="btn btn-primary" onClick={handleStartNewGame}>
                🔄 Play Again
              </button>
            )}
            <button className="btn btn-secondary" onClick={handleBackToHome}>
              ← Back to Home
            </button>
          </div>
        </div>
      </div>
    )
  }

  if (!currentRound) {
    return <div className="game-room">Loading round...</div>
  }

  // Show intro screen for 5 seconds
  if (showIntro && currentRound) {
    return (
      <div className="game-room pub-quiz">
        <div className="intro-screen">
          <div className="intro-content">
            <h1 className="round-number">Round {currentRound.round_num}</h1>
            <p className="round-type">{currentRound.round_type === 'pub_quiz' ? '🎯 Pub Quiz' : currentRound.round_type}</p>
          </div>
        </div>
      </div>
    )
  }

  if (!currentRound) {
    return <div className="game-room">Loading round...</div>
  }

  if (currentRound.round_type === 'pub_quiz') {
    return (
      <div className="game-room pub-quiz">
        <div className="timer">
          {timeLeft}
        </div>

        <div className="question-container">
          <h2 className="question-text">{currentRound.question.text}</h2>

          <div className="answers-grid">
            {currentRound.question.answers.map((answer, idx) => {
              let btnClass = 'answer-btn'
              if (showAnswerResult && selectedAnswer === idx) {
                btnClass += isAnswerCorrect ? ' correct-answer' : ' incorrect-answer'
              } else if (selectedAnswer === idx) {
                btnClass += ' selected'
              }
              if (hasAnswered) {
                btnClass += ' disabled'
              }
              
              return (
                <button
                  key={idx}
                  className={btnClass}
                  onClick={() => handleAnswerClick(idx)}
                  disabled={hasAnswered}
                >
                  {String.fromCharCode(65 + idx)}) {answer}
                  {showAnswerResult && selectedAnswer === idx && (
                    isAnswerCorrect && (
                      <svg className="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                        <polyline points="20 6 9 17 4 12"></polyline>
                      </svg>
                    )
                  )}
                  {showAnswerResult && selectedAnswer === idx && (
                    !isAnswerCorrect && (
                      <svg className="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                        <line x1="18" y1="6" x2="6" y2="18"></line>
                        <line x1="6" y1="6" x2="18" y2="18"></line>
                      </svg>
                    )
                  )}
                </button>
              )
            })}
          </div>

          {hasAnswered && !showAnswerResult && (
            <p className="answered-message">✓ Answer submitted! Waiting for others...</p>
          )}
          
          {showAnswerResult && isAnswerCorrect !== null && (
            <div className={`answer-result ${isAnswerCorrect ? 'correct' : 'incorrect'}`}>
              <svg className="result-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                {isAnswerCorrect ? (
                  <polyline points="20 6 9 17 4 12"></polyline>
                ) : (
                  <>
                    <line x1="18" y1="6" x2="6" y2="18"></line>
                    <line x1="6" y1="6" x2="18" y2="18"></line>
                  </>
                )}
              </svg>
              <p className="result-text">{isAnswerCorrect ? 'Correct!' : 'Incorrect'}</p>
            </div>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className="game-room">
      <h1>🎮 {currentRound.round_type}</h1>
      <p>Round {currentRound.round_num} coming soon...</p>
    </div>
  )
}
