package ws

import (
	"encoding/json"
	"time"

	"game/logger"

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
	logger.Debug("Hub", "SetRoundAnswer: Stored answer %d for round %d", correctAnswer, roundNum)
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
			logger.Panic("Hub", "Game %s panic recovered", r)
		}
	}()

	for {
		select {
		case client := <-h.Register:
			h.handleRegister(client)
		case client := <-h.Unregister:
			h.handleUnregister(client)
		case msg := <-h.Broadcast:
			h.handleBroadcast(msg)
		case <-h.done:
			h.closeAllClients()
			return
		}
	}
}

// handleRegister adds a client to the hub
func (h *Hub) handleRegister(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.Clients[client] = true
	logger.Info("Hub", "Client %s registered for game %s", client.PlayerID, h.GameID)
}

// handleUnregister removes a client from the hub
func (h *Hub) handleUnregister(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if _, ok := h.Clients[client]; ok {
		delete(h.Clients, client)
		close(client.Send)
	}
	logger.Info("Hub", "Client %s unregistered from game %s", client.PlayerID, h.GameID)
}

// handleBroadcast sends a message to all connected clients
func (h *Hub) handleBroadcast(msg *Message) {
	clients := h.getConnectedClients()
	logger.Debug("Hub", "Broadcasting to %d clients, msg type: %s", len(clients), msg.Type)

	for _, client := range clients {
		h.sendToClient(client, msg)
	}
}

// getConnectedClients returns a snapshot of all connected clients
func (h *Hub) getConnectedClients() []*Client {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	clients := make([]*Client, 0, len(h.Clients))
	for client := range h.Clients {
		clients = append(clients, client)
	}
	return clients
}

// sendToClient sends a message to a client, unregistering if the channel is full
func (h *Hub) sendToClient(client *Client, msg *Message) {
	select {
	case client.Send <- msg:
		// Successfully sent
	default:
		// Client's send channel is full or closed, unregister them
		logger.Debug("Hub", "Client %s send channel full/closed (msg type: %s), unregistering", client.PlayerID, msg.Type)
		go func(c *Client) {
			h.Unregister <- c
		}(client)
	}
}

// closeAllClients closes the send channel for all connected clients
func (h *Hub) closeAllClients() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	for client := range h.Clients {
		close(client.Send)
	}
}

// BroadcastMessage broadcasts a message to all connected clients
func (h *Hub) BroadcastMessage(msgType MessageType, payload interface{}) {
	logger.Debug("BroadcastMessage", "Queuing broadcast of type %s for game %s", msgType, h.GameID)
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
			logger.Panic("ReadPump", "Player %s panic recovered", r)
		}
		hub.Unregister <- c
		c.Conn.Close()
	}()

	c.setupReadDeadlines()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("ReadPump", "Unexpected close for player %s: %v", c.PlayerID, err)
			}
			return
		}

		logger.Debug("ReadPump", "Received raw message from player %s: %s", c.PlayerID, string(message))

		if !c.handleMessage(message, hub) {
			return
		}
	}
}

// setupReadDeadlines configures the WebSocket read deadline and pong handler
func (c *Client) setupReadDeadlines() {
	// Set read deadline to 5 minutes to avoid timeout during long games
	c.Conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	c.Conn.SetPongHandler(func(string) error {
		// Extend deadline on each pong
		c.Conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		return nil
	})
}

// handleMessage processes a received message
func (c *Client) handleMessage(data []byte, hub *Hub) bool {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		logger.Error("ReadPump", "Failed to unmarshal message for player %s: %v", c.PlayerID, err)
		return true
	}

	logger.Debug("ReadPump", "Received message type: %s from player %s", msg.Type, c.PlayerID)

	if msg.Type == MsgTypePlayerSubmission {
		return c.handlePlayerSubmission(msg, hub)
	}

	return true
}

// handlePlayerSubmission processes a player submission message
func (c *Client) handlePlayerSubmission(msg Message, hub *Hub) bool {
	logger.Debug("ReadPump", "Processing player submission from %s", c.PlayerID)

	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		logger.Error("ReadPump", "Invalid submission payload for player %s", c.PlayerID)
		return true
	}

	submission := c.parseSubmissionPayload(payload)
	isCorrect := c.validateAndStoreSubmission(submission, hub)

	ackMsg := c.buildSubmissionAckMessage(submission, isCorrect, hub)
	return c.sendSubmissionAck(ackMsg, hub)
}

// parseSubmissionPayload extracts submission data from the payload
func (c *Client) parseSubmissionPayload(payload map[string]interface{}) PlayerSubmissionPayload {
	var submission PlayerSubmissionPayload
	submission.PlayerID = c.PlayerID

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

	// Capture server-side submission timestamp in milliseconds
	submission.SubmissionTime = time.Now().UnixMilli()
	return submission
}

// validateAndStoreSubmission validates and stores a player submission
func (c *Client) validateAndStoreSubmission(submission PlayerSubmissionPayload, hub *Hub) bool {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()

	// Store submission
	if _, exists := hub.Submissions[submission.RoundNum]; !exists {
		hub.Submissions[submission.RoundNum] = make(map[string]*PlayerSubmissionPayload)
	}
	hub.Submissions[submission.RoundNum][c.PlayerID] = &submission

	// Check if answer is correct (for pub_quiz rounds)
	isCorrect := false
	if correctAnswer, exists := hub.RoundAnswers[submission.RoundNum]; exists {
		logger.Debug("ReadPump", "Found correct answer %d for round %d, player answered %d", correctAnswer, submission.RoundNum, submission.AnswerIdx)
		isCorrect = submission.AnswerIdx == correctAnswer
	} else {
		logger.Debug("ReadPump", "No correct answer stored for round %d. Available rounds: %v", submission.RoundNum, getMapKeys(hub.RoundAnswers))
	}

	logger.Debug("ReadPump", "Player %s submitted answer for round %d at %d ms, isCorrect: %v", c.PlayerID, submission.RoundNum, submission.SubmissionTime, isCorrect)
	return isCorrect
}

// buildSubmissionAckMessage creates a submission acknowledgement message
func (c *Client) buildSubmissionAckMessage(submission PlayerSubmissionPayload, isCorrect bool, hub *Hub) *Message {
	return &Message{
		Type:   MsgTypeSubmissionAck,
		GameID: hub.GameID,
		Payload: SubmissionAckPayload{
			RoundNum:  submission.RoundNum,
			IsCorrect: isCorrect,
			AnswerIdx: submission.AnswerIdx,
		},
	}
}

// sendSubmissionAck sends a submission acknowledgement to the client
func (c *Client) sendSubmissionAck(ackMsg *Message, hub *Hub) bool {
	select {
	case c.Send <- ackMsg:
		logger.Debug("ReadPump", "Sent submission ack to player %s (correct: %v)", c.PlayerID, ackMsg.Payload.(SubmissionAckPayload).IsCorrect)
		return true
	case <-hub.done:
		return false
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	defer c.Conn.Close()
	defer func() {
		if r := recover(); r != nil {
			logger.Panic("WritePump", "Player %s panic recovered", r)
		}
	}()

	msgCount := 0
	for msg := range c.Send {
		msgCount++
		if err := c.sendMessage(msg, msgCount); err != nil {
			return
		}
	}
	logger.Debug("WritePump", "Send channel closed for player %s after %d messages", c.PlayerID, msgCount)
}

// sendMessage sends a single message to the WebSocket connection
func (c *Client) sendMessage(msg *Message, msgNum int) error {
	logger.Debug("WritePump", "Player %s processing message #%d (type: %s)", c.PlayerID, msgNum, msg.Type)
	c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	w, err := c.Conn.NextWriter(websocket.TextMessage)
	if err != nil {
		logger.Error("WritePump", "NextWriter error for player %s at msg #%d: %v", c.PlayerID, msgNum, err)
		return err
	}
	defer w.Close()

	if err := json.NewEncoder(w).Encode(msg); err != nil {
		logger.Error("WritePump", "Encode error for player %s at msg #%d: %v", c.PlayerID, msgNum, err)
		return err
	}

	logger.Debug("WritePump", "Player %s successfully sent message #%d", c.PlayerID, msgNum)
	return nil
}
