package bll

import (
	"context"
	"errors"

	"github.com/jassus213/go-board/dashboard/dal"
)

// AddMemberHandler processes the addition or overwrite of a single dashboard member.
// It validates that both Dashboard and MemberID are provided before persisting data.
func AddMemberHandler(ctx context.Context, repo dal.DashboardRepository, req AddMemberRequest) error {
	if req.Dashboard == "" || req.MemberID == "" {
		return errors.New("dashboard and member ID are required")
	}

	return repo.AddMemberToDashboard(ctx, req.Dashboard, req.MemberID, req.Score)
}

// BatchAddHandler handles large-scale member ingestion using optimized repository methods.
// If the member slice is empty, the operation returns successfully without calling the repository.
func BatchAddHandler(ctx context.Context, repo dal.DashboardRepository, req BatchAddRequest) error {
	if len(req.Members) == 0 {
		return nil
	}
	return repo.AddMembersBatch(ctx, req.Dashboard, req.Members)
}

// IncrementScoreHandler adjusts a member's score by a specific delta.
// If the member does not exist, they are created with the increment value as their starting score.
func IncrementScoreHandler(ctx context.Context, repo dal.DashboardRepository, req IncrementScoreRequest) error {
	return repo.IncrementMemberScore(ctx, req.Dashboard, req.MemberID, req.Value)
}

// RemoveMemberHandler deletes a specific member from a dashboard.
func RemoveMemberHandler(ctx context.Context, repo dal.DashboardRepository, dashboard, memberID string) error {
	return repo.RemoveMemberFromDashboard(ctx, dashboard, memberID)
}

// DeleteDashboardHandler completely removes a dashboard and all its associated data.
func DeleteDashboardHandler(ctx context.Context, repo dal.DashboardRepository, dashboard string) error {
	return repo.DeleteDashboard(ctx, dashboard)
}

// BatchIncrementHandler processes a stream of increments.
func BatchIncrementHandler(ctx context.Context, repo dal.DashboardRepository, req BatchAddRequest) error {
	if len(req.Members) == 0 {
		return nil
	}

	return repo.IncrementMembersBatch(ctx, req.Dashboard, req.Members)
}
