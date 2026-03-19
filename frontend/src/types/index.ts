export type GameType = 'normal' | 'spicy' | 'unhinged'
export type RoundType = 'pub_quiz' | 'charades' | 'alibi' | 'copycat'
export type GameStatus = 'lobby' | 'in_progress' | 'finished'

export interface Game {
  id: string
  code: string
  host_id: string
  game_type: GameType
  round_count: number
  current_round: number
  status: GameStatus
  created_at: string
}

export interface Player {
  id: string
  game_id: string
  name?: string
  is_host: boolean
  created_at: string
}

export interface PubQuizQuestion {
  id: string
  text: string
  answers: string[]
  correct_answer_idx: number
  game_type: GameType
  is_player_specific: boolean
}

export interface Round {
  id: string
  game_id: string
  round_num: number
  round_type: RoundType
  prompt?: string
  crime_text?: string
  question_id?: string
  actor_player_id?: string
}

export interface PlayerScore {
  id: string
  game_id: string
  player_id: string
  round_num: number
  round_type: RoundType
  points: number
}
