// Package dal (Data Access Layer) defines the abstractions for data persistence.
// It provides interfaces to interact with various storage backends for dashboard management.
package dal

import (
	"context"

	"github.com/jassus213/go-board/dashboard/domain"
)

// DashboardRepository defines the set of operations required to manage dashboards.
// It abstracts the underlying storage (e.g., Redis, PostgreSQL) and focuses on
// leaderboards, rankings, and member score management.
type DashboardRepository interface {
	// AddMemberToDashboard adds a member with a specific score to a dashboard.
	// If the member already exists, their score will be overwritten.
	AddMemberToDashboard(ctx context.Context, dashboard, member string, score float64) error

	// AddMembersBatch performs a bulk update of multiple members in a single operation.
	// This is the preferred method for high-volume data ingestion.
	AddMembersBatch(ctx context.Context, dashboard string, members []domain.DashboardRecord) error

	// RemoveMemberFromDashboard deletes a specific member from the dashboard.
	// If the member is not found, the operation should return nil (no-op).
	RemoveMemberFromDashboard(ctx context.Context, dashboard, member string) error

	// GetTopMembers retrieves the leading members of a dashboard, sorted by score in descending order.
	// The 'top' parameter specifies the number of records to return (e.g., top 10).
	GetTopMembers(ctx context.Context, dashboard string, top int64) ([]domain.DashboardRecord, error)

	// ViewMemberRank returns the current position (rank) of a member in the dashboard.
	// The rank is typically 1-based and calculated based on descending scores.
	// Should return domain.ErrMemberNotFound if the member does not exist.
	ViewMemberRank(ctx context.Context, dashboard string, memberId string) (int64, error)

	// IncrementMemberScore modifies a member's current score by adding the increment value.
	// If the member does not exist, they will be created with the increment as their initial score.
	IncrementMemberScore(ctx context.Context, dashboard, member string, increment float64) error

	// GetTotalMembers returns the total count of unique members currently in the dashboard.
	GetTotalMembers(ctx context.Context, dashboard string) (int64, error)

	// DeleteDashboard removes the entire dashboard and all its associated data from the storage.
	DeleteDashboard(ctx context.Context, dashboard string) error

	// IncrementMembersBatch performs a bulk increment of multiple members' scores in a single operation.
	// It uses Redis Pipelining to minimize network round-trips.
	// This is highly efficient for processing streams of score updates.
	IncrementMembersBatch(ctx context.Context, dashboard string, increments []domain.DashboardRecord) error
}
