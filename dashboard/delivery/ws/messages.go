package ws

// InboundMessage represents the JSON payload sent by the client to the server via WebSocket.
// It encapsulates a command to modify a specific member's score on a dashboard.
type InboundMessage struct {
	// Dashboard is the unique identifier of the leaderboard to update.
	Dashboard string `json:"dashboard"`
	// MemberID is the unique identifier of the user or participant.
	MemberID string `json:"member_id"`
	// Increment is the delta to add to the current score (can be negative to decrease).
	Increment float64 `json:"increment"`
}

// OutboundMessage represents the server's response sent back to the client.
// It contains the real-time result of the operation, including the new rank.
type OutboundMessage struct {
	// Rank is the member's new 1-based position in the leaderboard.
	Rank int64 `json:"rank"`
	// Score is the member's updated total score.
	Score float64 `json:"score"`
	// Error contains a description if the operation failed; otherwise, it is empty and omitted from the JSON.
	Error string `json:"error,omitempty"`
}
