import { useEffect, useState } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { useWebSocket } from '../hooks/useWebSocket'
import './Lobby.css'

interface Player {
  id: string
  name: string
  is_host: boolean
  score: number
}

export default function Lobby() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const gameId = searchParams.get('game')
  const playerId = searchParams.get('player')
  const isHost = localStorage.getItem('isHost') === 'true'

  console.log('Lobby mounted - gameId:', gameId, 'playerId:', playerId, 'isHost:', isHost)

  const { isConnected, sendMessage, onMessage } = useWebSocket(gameId || '', playerId || '')

  const [players, setPlayers] = useState<Player[]>([])
  const [playerName, setPlayerName] = useState('')
  const [nameSaved, setNameSaved] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // Load initial game state and player info
  useEffect(() => {
    if (!gameId || !playerId) {
      console.log('Missing gameId or playerId, navigating home')
      navigate('/')
      return
    }

    console.log('Loading game data for', gameId)
    const loadGame = async () => {
      try {
        const response = await fetch(`/api/games?game_id=${gameId}`)
        if (!response.ok) throw new Error('Failed to load game')
        const data = await response.json()
        setPlayers(data.players || [])

        // Check if this player already has a name
        const thisPlayer = data.players?.find((p: Player) => p.id === playerId)
        if (thisPlayer?.name) {
          setPlayerName(thisPlayer.name)
          setNameSaved(true)
        }
      } catch (err) {
        setError('Failed to load game')
      }
    }

    loadGame()
  }, [gameId, playerId, navigate])

  // Listen for player joined events
  useEffect(() => {
    onMessage('player_joined', (payload) => {
      console.log('Player joined:', payload)
      setPlayers(payload.players || [])
    })
  }, [onMessage])

  // Listen for game started
  useEffect(() => {
    onMessage('game_started', (payload) => {
      console.log('Game started:', payload)
      setTimeout(() => {
        navigate(`/game?game=${gameId}&player=${playerId}`)
      }, 500)
    })
  }, [onMessage, gameId, playerId, navigate])

  const handleSaveName = async () => {
    if (!playerName.trim()) {
      setError('Name cannot be empty')
      return
    }

    setLoading(true)
    setError('')
    try {
      const response = await fetch(
        `/api/players/name?game_id=${gameId}&player_id=${playerId}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: playerName }),
        }
      )

      if (!response.ok) throw new Error('Failed to save name')
      setNameSaved(true)

      // Update the local players list with the new name
      setPlayers((prev) =>
        prev.map((p) =>
          p.id === playerId
            ? { ...p, name: playerName }
            : p
        )
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  const handleStartGame = async () => {
    setLoading(true)
    try {
      const response = await fetch('/api/games/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ game_id: gameId }),
      })

      if (!response.ok) throw new Error('Failed to start game')
      // Navigation will happen via WebSocket game_started event
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  if (!gameId || !playerId) {
    return <div className="lobby"><div className="lobby-container">Invalid session: gameId={gameId}, playerId={playerId}</div></div>
  }

  return (
    <div className="lobby">
      <div className="lobby-container">
        <button className="back-btn" onClick={() => navigate('/')}>← Back</button>
        
        <div className="header">
          <h1>🎮 Game Lobby</h1>
          <p>{isConnected ? 'Connected' : 'Reconnecting...'}</p>
        </div>

        {error && <div className="error">{error}</div>}

        {!nameSaved ? (
          <div className="name-setup">
            <h2>Enter Your Name</h2>
            <div className="form-group">
              <input
                type="text"
                placeholder="Your name"
                value={playerName}
                onChange={(e) => setPlayerName(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handleSaveName()}
                disabled={loading}
                autoFocus
                maxLength={30}
              />
            </div>
            <button
              className="btn btn-primary"
              onClick={handleSaveName}
              disabled={loading || !playerName.trim()}
            >
              {loading ? 'Saving...' : 'Save Name'}
            </button>
          </div>
        ) : (
          <div className="players-and-actions">
            <div className="players-section">
              <h2>Players ({players.filter(p => p.name).length}/{players.length})</h2>
              <ul className="players-list">
                {players.filter(p => p.name).map((player) => (
                  <li key={player.id} className={player.id === playerId ? 'you' : ''}>
                    <span className="player-name">{player.name}</span>
                    {player.is_host && <span className="host-badge">Host</span>}
                    {player.id === playerId && <span className="you-badge">You</span>}
                  </li>
                ))}
              </ul>
              {players.filter(p => !p.name).length > 0 && (
                <p className="setting-name-notice">
                  {players.filter(p => !p.name).length} player(s) setting their name...
                </p>
              )}
            </div>

            <div className="action-section">
              {isHost && (
                <button
                  className="btn btn-primary btn-large"
                  onClick={handleStartGame}
                  disabled={loading || players.length < 2}
                >
                  {loading ? 'Starting...' : '🚀 Start Game'}
                </button>
              )}

              {!isHost && (
                <div className="waiting-message">
                  <p>Waiting for host to start the game...</p>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

