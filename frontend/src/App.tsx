import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'
import HomePage from './pages/HomePage'
import GameSetup from './pages/GameSetup'
import JoinGame from './pages/JoinGame'
import Lobby from './pages/Lobby'
import GameRoom from './pages/GameRoom'
import './App.css'

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/create" element={<GameSetup />} />
        <Route path="/join" element={<JoinGame />} />
        <Route path="/lobby" element={<Lobby />} />
        <Route path="/game" element={<GameRoom />} />
      </Routes>
    </Router>
  )
}

export default App
