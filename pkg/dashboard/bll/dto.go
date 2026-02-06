// Package bll (Business Logic Layer) implements the core functional logic
// for dashboard management using a Functional CQRS approach.
// It coordinates data flow between the delivery layer and the persistence layer.
package bll

import (
	"github.com/jassus213/go-board/dashboard/domain"
)

// AddMemberRequest contains parameters for adding or updating a single member.
type AddMemberRequest struct {
	Dashboard string  // Unique identifier for the leaderboard
	MemberID  string  // Unique identifier for the member
	Score     float64 // Initial or new score value
}

// BatchAddRequest provides a collection of records for high-performance bulk updates.
type BatchAddRequest struct {
	Dashboard string                   // Unique identifier for the target leaderboard
	Members   []domain.DashboardRecord // List of members to be added or updated
}

// IncrementScoreRequest holds data for relative score adjustments.
type IncrementScoreRequest struct {
	Dashboard string  // Unique identifier for the leaderboard
	MemberID  string  // Unique identifier for the member
	Value     float64 // Amount to add to the existing score (can be negative)
}

// GetTopRequest defines filtering criteria for retrieving leaderboard standings.
type GetTopRequest struct {
	Dashboard string // Unique identifier for the leaderboard
	Limit     int64  // Maximum number of top records to return
}

// GetRankRequest specifies criteria for looking up a single member's standing.
type GetRankRequest struct {
	Dashboard string // Unique identifier for the leaderboard
	MemberID  string // Unique identifier for the member to look up
}
