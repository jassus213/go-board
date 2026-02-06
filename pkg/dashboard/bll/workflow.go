package bll

import (
	"context"

	"github.com/jassus213/GoBoard/dashboard/dal"
)

// ProcessScoreUpdate is a composite business operation (Workflow) that performs
// two sequential actions: it increments a member's score and then retrieves
// their updated rank within the specified dashboard.
//
// This is a high-level function designed to provide immediate feedback
// to clients (like WebSockets or gRPC streams) after a score change.
//
// Returns the new 1-based rank of the member or an error if either
// the increment or the rank retrieval fails.
func ProcessScoreUpdate(ctx context.Context, repo dal.DashboardRepository, req IncrementScoreRequest) (int64, error) {
	if err := IncrementScoreHandler(ctx, repo, req); err != nil {
		return 0, err
	}

	rank, err := GetMemberRankHandler(ctx, repo, GetRankRequest{
		Dashboard: req.Dashboard,
		MemberID:  req.MemberID,
	})

	if err != nil {
		return 0, err
	}

	return rank, nil
}
