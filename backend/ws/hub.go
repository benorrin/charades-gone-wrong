package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// NewHub creates a new WebSocket hub for a game
func NewHub(gameID string) *Hub {
	return &Hub{
		GameID:       gameID,
		Clients:      make(map[*Client]bool),
		Broadcast:    make(chan *Message, 256),
		Register:     make(chan *Client),
		Unregister:   make(chan *Client),
		Submissions:  make(map[int]map[string]*PlayerSubmissionPayload),
		RoundAnswers: make(map[int]int),
		done:         make(chan struct{}),
	}
}

// SetRoundAnswer stores the correct answer for a round
func (h *Hub) SetRoundAnswer(roundNum int, correctAnswer int) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.RoundAnswers[roundNum] = correctAnswer
}

// Helper to get map keys for logging
func getMapKeys(m map[int]int) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Run starts the hub's broadcaster
func (h *Hub) Run() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Hub PANIC] Game %s: %v", h.GameID, r)
		}
	}()

	for {
		select {
		case client := <-h.Register:
			h.mutex.Lock()
			h.Clients[client] = true
			h.mutex.Unlock()
			log.Printf("[Hub] Client %s registered for game %s", client.PlayerID, h.GameID)

		case client := <-h.Unregister:
			h.mutex.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}
			h.mutex.Unlock()
			log.Printf("[Hub] Client %s unregistered from game %s", client.PlayerID, h.GameID)

		case msg := <-h.Broadcast:
			h.mutex.RLock()
			clients := make([]*Client, 0, len(h.Clients))
			for client := range h.Clients {
				clients = append(clients, client)
			}
			h.mutex.RUnlock()
			log.Printf("[Hub] Broadcasting to %d clients, msg type: %s", len(clients), msg.Type)

			for _, client := range clients {
				select {
				case client.Send <- msg:
					// Successfully sent
				default:
					// Client's send channel is full or closed, unregister them
					log.Printf("[Hub] Client %s send channel full/closed (msg type: %s), unregistering", client.PlayerID, msg.Type)
					go func(c *Client) {
						h.Unregister <- c
					}(client)
				}
			}

		case <-h.done:
			h.mutex.Lock()
			for client := range h.Clients {
				close(client.Send)
			}
			h.mutex.Unlock()
			return
		}
	}
}

// BroadcastMessage broadcasts a message to all connected clients
func (h *Hub) BroadcastMessage(msgType MessageType, payload interface{}) {
	log.Printf("[BroadcastMessage] Queuing broadcast of type %s for game %s", msgType, h.GameID)
	msg := &Message{
		Type:    msgType,
		GameID:  h.GameID,
		Payload: payload,
	}
	h.Broadcast <- msg
}

// Close stops the hub and closes all client connections
func (h *Hub) Close() {
	close(h.done)
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.Clients)
}

// GetRoundSubmissions returns all submissions for a given round
func (h *Hub) GetRoundSubmissions(roundNum int) map[string]*PlayerSubmissionPayload {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if submissions, exists := h.Submissions[roundNum]; exists {
		// Return a copy to avoid race conditions
		result := make(map[string]*PlayerSubmissionPayload)
		for k, v := range submissions {
			result[k] = v
		}
		return result
	}
	return make(map[string]*PlayerSubmissionPayload)
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump(hub *Hub) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ReadPump PANIC] Player %s: %v", c.PlayerID, r)
		}
		hub.Unregister <- c
		c.Conn.Close()
	}()

	// Set read deadline to 5 minutes to avoid timeout during long games
	c.Conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	c.Conn.SetPongHandler(func(string) error {
		// Extend deadline on each pong
		c.Conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[ReadPump] Unexpected close for player %s: %v", c.PlayerID, err)
			}
			return
		}

		log.Printf("[ReadPump] Received raw message from player %s: %s", c.PlayerID, string(message))

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[ReadPump] Failed to unmarshal message for player %s: %v", c.PlayerID, err)
			continue
		}

		log.Printf("[ReadPump] Received message type: %s from player %s", msg.Type, c.PlayerID)

		// Handle player submissions
		if msg.Type == MsgTypePlayerSubmission {
			log.Printf("[ReadPump] Processing player submission from %s", c.PlayerID)
			payload, ok := msg.Payload.(map[string]interface{})
			if !ok {
				log.Printf("[ReadPump] Invalid submission payload for player %s", c.PlayerID)
				continue
			}

			// Extract submission data
			var submission PlayerSubmissionPayload
			if roundNum, ok := payload["round_num"].(float64); ok {
				submission.RoundNum = int(roundNum)
			}
			if answerIdx, ok := payload["answer_idx"].(float64); ok {
				submission.AnswerIdx = int(answerIdx)
			}
			if rating, ok := payload["rating"].(string); ok {
				submission.Rating = rating
			}
			if voteBelievable, ok := payload["vote_believable"].(bool); ok {
				submission.VoteBelievable = voteBelievable
			}
			if answer, ok := payload["answer"].(string); ok {
				submission.Answer = answer
			}

			submission.PlayerID = c.PlayerID
			// Capture server-side submission timestamp in milliseconds
			submission.SubmissionTime = time.Now().UnixMilli()

			// Store submission
			hub.mutex.Lock()
			if _, exists := hub.Submissions[submission.RoundNum]; !exists {
				hub.Submissions[submission.RoundNum] = make(map[string]*PlayerSubmissionPayload)
			}
			hub.Submissions[submission.RoundNum][c.PlayerID] = &submission

			// Check if answer is correct (for pub_quiz rounds)
			isCorrect := false
			if correctAnswer, exists := hub.RoundAnswers[submission.RoundNum]; exists {
				log.Printf("[ReadPump] Found correct answer %d for round %d, player answered %d", correctAnswer, submission.RoundNum, submission.AnswerIdx)
				isCorrect = submission.AnswerIdx == correctAnswer
			} else {
				log.Printf("[ReadPump] No correct answer stored for round %d. Available rounds: %v", submission.RoundNum, getMapKeys(hub.RoundAnswers))
			}
			hub.mutex.Unlock()

			log.Printf("[ReadPump] Player %s submitted answer for round %d at %d ms", c.PlayerID, submission.RoundNum, submission.SubmissionTime)

			// Send submission acknowledgement to this player
			ackMsg := &Message{
				Type:   MsgTypeSubmissionAck,
				GameID: hub.GameID,
				Payload: SubmissionAckPayload{
					RoundNum:  submission.RoundNum,
					IsCorrect: isCorrect,
					AnswerIdx: submission.AnswerIdx,
				},
			}
			select {
			case c.Send <- ackMsg:
				log.Printf("[ReadPump] Sent submission ack to player %s (correct: %v)", c.PlayerID, isCorrect)
			case <-hub.done:
				return
			}
		}
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	defer c.Conn.Close()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[WritePump PANIC] Player %s: %v", c.PlayerID, r)
		}
	}()

	msgCount := 0
	for msg := range c.Send {
		msgCount++
		log.Printf("[WritePump] Player %s processing message #%d (type: %s)", c.PlayerID, msgCount, msg.Type)
		c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

		w, err := c.Conn.NextWriter(websocket.TextMessage)
		if err != nil {
			log.Printf("[WritePump] NextWriter error for player %s at msg #%d: %v", c.PlayerID, msgCount, err)
			return
		}

		if err := json.NewEncoder(w).Encode(msg); err != nil {
			log.Printf("[WritePump] Encode error for player %s at msg #%d: %v", c.PlayerID, msgCount, err)
			w.Close()
			return
		}

		if err := w.Close(); err != nil {
			log.Printf("[WritePump] Close error for player %s at msg #%d: %v", c.PlayerID, msgCount, err)
			return
		}
		log.Printf("[WritePump] Player %s successfully sent message #%d", c.PlayerID, msgCount)
	}
	log.Printf("[WritePump] Send channel closed for player %s after %d messages", c.PlayerID, msgCount)
}
