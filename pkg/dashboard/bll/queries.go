package bll

import (
	"context"

	"github.com/jassus213/go-board/dashboard/dal"
	"github.com/jassus213/go-board/dashboard/domain"
)

// GetTopMembersHandler retrieves the highest-ranking members based on the requested limit.
// If no Limit is specified (<= 0), it defaults to retrieving the top 10 members.
func GetTopMembersHandler(ctx context.Context, repo dal.DashboardRepository, req GetTopRequest) ([]domain.DashboardRecord, error) {
	if req.Limit <= 0 {
		req.Limit = 10
	}
	return repo.GetTopMembers(ctx, req.Dashboard, req.Limit)
}

// GetMemberRankHandler retrieves the current 1-based rank of a member.
// It returns domain.ErrMemberNotFound if the member is not part of the specified dashboard.
func GetMemberRankHandler(ctx context.Context, repo dal.DashboardRepository, req GetRankRequest) (int64, error) {
	return repo.ViewMemberRank(ctx, req.Dashboard, req.MemberID)
}

// GetDashboardStatsHandler returns the current total number of participants in a dashboard.
func GetDashboardStatsHandler(ctx context.Context, repo dal.DashboardRepository, dashboard string) (int64, error) {
	return repo.GetTotalMembers(ctx, dashboard)
}
