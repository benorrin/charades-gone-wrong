import { useNavigate } from 'react-router-dom'
import './HomePage.css'

export default function HomePage() {
  const navigate = useNavigate()

  return (
    <div className="home-page">
      <div className="container">
        <h1 className="title">🎭 Charades Gone Wrong</h1>
        <p className="subtitle">A party game of ridiculous challenges and hilarious moments</p>

        <div className="buttons-grid">
          <button className="btn btn-primary" onClick={() => navigate('/create')}>
            ✨ Create Game
          </button>
          <button className="btn btn-secondary" onClick={() => navigate('/join')}>
            🎮 Join Game
          </button>
        </div>

        <details className="how-to-play">
          <summary>📖 How to Play</summary>
          <div className="how-to-content">
            <h3>Game Modes</h3>
            <ul>
              <li><strong>Pub Quiz:</strong> Answer multiple choice questions based on general knowledge and player facts. Speed matters!</li>
              <li><strong>Charades Gone Wrong:</strong> Act out prompts while others rate your performance (Rubbish, Good, Great, Funny, Cringe)</li>
              <li><strong>Alibi:</strong> If you're the suspect, improvise an alibi for a ridiculous crime. Others vote if it's believable</li>
              <li><strong>Copycat:</strong> Everyone writes their answer to a prompt. Matching answers = everyone drinks (0 points). Be unique!</li>
            </ul>
            <h3>Scoring</h3>
            <ul>
              <li>Correct answers + speed bonuses</li>
              <li>Performance ratings and truthfulness votes earn points</li>
              <li>After all rounds, the highest score wins!</li>
            </ul>
          </div>
        </details>
      </div>
    </div>
  )
}
