package usecase

import (
	"context"
	"errors"

	"github.com/jassus213/go-board/dashboard/core/dto"
	"github.com/jassus213/go-board/dashboard/core/entity"
	"github.com/jassus213/go-board/dashboard/core/interfaces"
)

var _ interfaces.DashboardUseCase = (*BoardUseCase)(nil)

type BoardUseCase struct {
	repo interfaces.DashboardRepository
}

func New(repo interfaces.DashboardRepository) *BoardUseCase {
	return &BoardUseCase{repo: repo}
}

// AddMemberHandler processes the addition or overwrite of a single dashboard member.
// It validates that both Dashboard and MemberID are provided before persisting data.
func (u *BoardUseCase) AddMemberHandler(ctx context.Context, req dto.AddMemberRequest) error {
	if req.Dashboard == "" || req.MemberID == "" {
		return errors.New("dashboard and member ID are required")
	}

	return u.repo.AddMemberToDashboard(ctx, req.Dashboard, req.MemberID, req.Score)
}

// BatchAddHandler handles large-scale member ingestion using optimized repository methods.
// If the member slice is empty, the operation returns successfully without calling the repository.
func (u *BoardUseCase) BatchAddHandler(ctx context.Context, req dto.BatchAddRequest) error {
	if len(req.Members) == 0 {
		return nil
	}

	return u.repo.AddMembersBatch(ctx, req.Dashboard, req.Members)
}

// IncrementScoreHandler adjusts a member's score by a specific delta.
// If the member does not exist, they are created with the increment value as their starting score.
func (u *BoardUseCase) IncrementScoreHandler(ctx context.Context, req dto.IncrementScoreRequest) error {
	return u.repo.IncrementMemberScore(ctx, req.Dashboard, req.MemberID, req.Value)
}

// RemoveMemberHandler deletes a specific member from a dashboard.
func (u *BoardUseCase) RemoveMemberHandler(ctx context.Context, dashboard, memberID string) error {
	return u.repo.RemoveMemberFromDashboard(ctx, dashboard, memberID)
}

// DeleteDashboardHandler completely removes a dashboard and all its associated data.
func (u *BoardUseCase) DeleteDashboardHandler(ctx context.Context, dashboard string) error {
	return u.repo.DeleteDashboard(ctx, dashboard)
}

// BatchIncrementHandler processes a stream of increments.
func (u *BoardUseCase) BatchIncrementHandler(ctx context.Context, req dto.BatchAddRequest) error {
	if len(req.Members) == 0 {
		return nil
	}

	return u.repo.IncrementMembersBatch(ctx, req.Dashboard, req.Members)
}

// GetTopMembersHandler retrieves the highest-ranking members based on the requested limit.
// If no Limit is specified (<= 0), it defaults to retrieving the top 10 members.
func (u *BoardUseCase) GetTopMembersHandler(ctx context.Context, req dto.GetTopRequest) ([]entity.DashboardRecord, error) {
	if req.Limit <= 0 {
		req.Limit = 10
	}
	return u.repo.GetTopMembers(ctx, req.Dashboard, req.Limit)
}

// GetMemberRankHandler retrieves the current 1-based rank of a member.
// It returns domain.ErrMemberNotFound if the member is not part of the specified dashboard.
func (u *BoardUseCase) GetMemberRankHandler(ctx context.Context, req dto.GetRankRequest) (int64, error) {
	return u.repo.ViewMemberRank(ctx, req.Dashboard, req.MemberID)
}

// GetDashboardStatsHandler returns the current total number of participants in a dashboard.
func (u *BoardUseCase) GetDashboardStatsHandler(ctx context.Context, dashboard string) (int64, error) {
	return u.repo.GetTotalMembers(ctx, dashboard)
}

// ProcessScoreUpdate is a composite business operation (Workflow) that performs
// two sequential actions: it increments a member's score and then retrieves
// their updated rank within the specified dashboard.
//
// This is a high-level function designed to provide immediate feedback
// to clients (like WebSockets or gRPC streams) after a score change.
//
// Returns the new 1-based rank of the member or an error if either
// the increment or the rank retrieval fails.
func (u *BoardUseCase) ProcessScoreUpdate(ctx context.Context, req dto.IncrementScoreRequest) (int64, error) {
	if err := u.IncrementScoreHandler(ctx, req); err != nil {
		return 0, err
	}

	rank, err := u.GetMemberRankHandler(ctx, dto.GetRankRequest{
		Dashboard: req.Dashboard,
		MemberID:  req.MemberID,
	})

	if err != nil {
		return 0, err
	}

	return rank, nil
}
