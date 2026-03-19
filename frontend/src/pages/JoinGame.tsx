import { useNavigate } from 'react-router-dom'
import { useState } from 'react'
import './JoinGame.css'

export default function JoinGame() {
  const navigate = useNavigate()
  const [code, setCode] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleJoinGame = async () => {
    if (!code.trim()) {
      setError('Please enter a game code')
      return
    }

    setLoading(true)
    setError('')
    try {
      const response = await fetch('/api/games/join', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code: code.toUpperCase() }),
      })

      if (!response.ok) {
        if (response.status === 404) {
          throw new Error('Game not found')
        }
        throw new Error('Failed to join game')
      }

      const data = await response.json()
      
      // Store in localStorage
      localStorage.setItem('gameId', data.game_id)
      localStorage.setItem('playerId', data.player_id)
      localStorage.setItem('isHost', 'false')

      // Navigate to lobby
      navigate(`/lobby?game=${data.game_id}&player=${data.player_id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="join-game">
      <button className="back-btn" onClick={() => navigate('/')}>← Back</button>

      <div className="container">
        <h2>Join Game</h2>
        <p>Enter the code from the host</p>

        {error && <div className="error">{error}</div>}

        <div className="form-group">
          <input
            type="text"
            placeholder="Game Code (e.g., ABC123)"
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase())}
            onKeyPress={(e) => e.key === 'Enter' && handleJoinGame()}
            disabled={loading}
            maxLength={6}
            autoFocus
            className="code-input"
          />
        </div>

        <button
          className="btn btn-primary"
          onClick={handleJoinGame}
          disabled={loading || code.length !== 6}
        >
          {loading ? 'Joining...' : 'Join Game'}
        </button>
      </div>
    </div>
  )
}
