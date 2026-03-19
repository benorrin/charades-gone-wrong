import { useNavigate } from 'react-router-dom'
import { useState } from 'react'
import './GameSetup.css'

export default function GameSetup() {
  const navigate = useNavigate()
  const [gameType, setGameType] = useState('normal')
  const [roundCount, setRoundCount] = useState(20)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [joinCode, setJoinCode] = useState('')
  const [gameId, setGameId] = useState('')
  const [hostId, setHostId] = useState('')

  const handleCreateGame = async () => {
    setLoading(true)
    setError('')
    try {
      const response = await fetch('/api/games/create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          game_type: gameType,
          round_count: roundCount,
        }),
      })

      if (!response.ok) throw new Error('Failed to create game')

      const data = await response.json()
      setJoinCode(data.code)
      setGameId(data.game_id)
      setHostId(data.host_id)
      
      // Store in localStorage for use in lobby
      localStorage.setItem('gameId', data.game_id)
      localStorage.setItem('playerId', data.host_id)
      localStorage.setItem('isHost', 'true')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  if (joinCode) {
    return (
      <div className="game-setup">
        <button className="back-btn" onClick={() => {
          setJoinCode('')
          setError('')
        }}>← Back</button>

        <div className="container">
          <h2>Game Created! 🎉</h2>
          <p>Share this code with your friends:</p>

          <div className="join-code-card">
            <div className="join-code">{joinCode}</div>
            <button className="copy-btn" onClick={() => {
              navigator.clipboard.writeText(joinCode)
              alert('Code copied!')
            }}>
              📋 Copy Code
            </button>
          </div>

          <div className="game-settings">
            <p><strong>Game Type:</strong> {gameType}</p>
            <p><strong>Total Rounds:</strong> {roundCount}</p>
          </div>

          <button className="btn btn-primary" onClick={() => {
            navigate(`/lobby?game=${gameId}&player=${hostId}`)
          }}>
            Go to Lobby →
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="game-setup">
      <button className="back-btn" onClick={() => navigate('/')}>← Back</button>

      <div className="container">
        <h2>Create Game</h2>

        {error && <div className="error">{error}</div>}

        <form onSubmit={(e) => {
          e.preventDefault()
          handleCreateGame()
        }}>
          <div className="form-group">
            <label>Game Type</label>
            <select 
              value={gameType} 
              onChange={(e) => setGameType(e.target.value)}
              disabled={loading}
            >
              <option value="normal">Normal</option>
              <option value="spicy">Spicy</option>
              <option value="unhinged">Unhinged</option>
            </select>
          </div>

          <div className="form-group">
            <label>Number of Rounds: {roundCount}</label>
            <input 
              type="range" 
              min="10" 
              max="30" 
              value={roundCount}
              onChange={(e) => setRoundCount(parseInt(e.target.value))}
              disabled={loading}
            />
            <p className="hint">10-30 rounds recommended</p>
          </div>

          <button 
            className="btn btn-primary"
            type="submit"
            disabled={loading}
          >
            {loading ? 'Creating...' : 'Create Game'}
          </button>
        </form>
      </div>
    </div>
  )
}
