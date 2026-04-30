package usecase

import (
	"context"
	"testing"

	"github.com/jassus213/go-board/dashboard/core"
	"github.com/jassus213/go-board/dashboard/core/dto"
	"github.com/jassus213/go-board/dashboard/core/entity"
	"github.com/jassus213/go-board/dashboard/repo/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAddMemberHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("should_fail_if_dashboard_is_empty", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)
		uc := New(repo)

		req := dto.AddMemberRequest{Dashboard: "", MemberID: "u1", Score: 10}

		err := uc.AddMemberHandler(ctx, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dashboard and member ID are required")
	})

	t.Run("should_call_repo_on_success", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)
		uc := New(repo)

		req := dto.AddMemberRequest{Dashboard: "db1", MemberID: "u1", Score: 100}

		repo.EXPECT().
			AddMemberToDashboard(ctx, "db1", "u1", float64(100)).
			Return(nil)

		err := uc.AddMemberHandler(ctx, req)
		assert.NoError(t, err)
	})
}

func TestBatchAddHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("should_return_nil_without_calling_repo_if_empty", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)
		uc := New(repo)

		req := dto.BatchAddRequest{Dashboard: "db1", Members: []entity.DashboardRecord{}}

		err := uc.BatchAddHandler(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("should_call_batch_add_on_repo", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)
		uc := New(repo)

		mems := []entity.DashboardRecord{{ID: "u1", Score: 10}}
		req := dto.BatchAddRequest{Dashboard: "db1", Members: mems}

		repo.EXPECT().
			AddMembersBatch(ctx, "db1", mems).
			Return(nil)

		err := uc.BatchAddHandler(ctx, req)
		assert.NoError(t, err)
	})
}

func TestIncrementScoreHandler(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewDashboardRepository(t)
	uc := New(repo)

	req := dto.IncrementScoreRequest{Dashboard: "db1", MemberID: "u1", Value: 5.5}

	repo.EXPECT().
		IncrementMemberScore(ctx, "db1", "u1", 5.5).
		Return(nil)

	err := uc.IncrementScoreHandler(ctx, req)
	assert.NoError(t, err)
}

func TestRemoveMemberHandler(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewDashboardRepository(t)
	uc := New(repo)

	repo.EXPECT().
		RemoveMemberFromDashboard(ctx, "db1", "u1").
		Return(nil)

	err := uc.RemoveMemberHandler(ctx, "db1", "u1")
	assert.NoError(t, err)
}

func TestDeleteDashboardHandler(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewDashboardRepository(t)
	uc := New(repo)

	repo.EXPECT().
		DeleteDashboard(ctx, "db1").
		Return(nil)

	err := uc.DeleteDashboardHandler(ctx, "db1")
	assert.NoError(t, err)
}

func TestBatchIncrementHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("should_skip_repo_if_empty", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)
		uc := New(repo)

		err := uc.BatchIncrementHandler(ctx, dto.BatchAddRequest{Members: nil})
		assert.NoError(t, err)
	})

	t.Run("should_call_increment_batch", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)
		uc := New(repo)

		mems := []entity.DashboardRecord{{ID: "u1", Score: 1}}

		repo.EXPECT().
			IncrementMembersBatch(ctx, "db1", mems).
			Return(nil)

		err := uc.BatchIncrementHandler(ctx, dto.BatchAddRequest{Dashboard: "db1", Members: mems})
		assert.NoError(t, err)
	})
}

func TestGetTopMembersHandler(t *testing.T) {
	ctx := context.Background()
	dashboard := "season_1"

	t.Run("should_use_provided_limit", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)
		uc := New(repo)

		expectedRecords := []entity.DashboardRecord{
			{ID: "user1", Score: 100, Rank: 1},
		}

		repo.EXPECT().
			GetTopMembers(ctx, dashboard, int64(5)).
			Return(expectedRecords, nil)

		res, err := uc.GetTopMembersHandler(ctx, dto.GetTopRequest{
			Dashboard: dashboard,
			Limit:     5,
		})

		assert.NoError(t, err)
		assert.Equal(t, expectedRecords, res)
	})

	t.Run("should_default_to_10_if_limit_is_zero_or_less", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)
		uc := New(repo)

		repo.EXPECT().
			GetTopMembers(ctx, dashboard, int64(10)).
			Return(nil, nil)

		_, err := uc.GetTopMembersHandler(ctx, dto.GetTopRequest{
			Dashboard: dashboard,
			Limit:     0, // или -1
		})

		assert.NoError(t, err)
	})
}

func TestGetMemberRankHandler(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewDashboardRepository(t)

	t.Run("success", func(t *testing.T) {
		repo.EXPECT().
			ViewMemberRank(ctx, "db", "user1").
			Return(int64(42), nil)
		uc := New(repo)

		rank, err := uc.GetMemberRankHandler(ctx, dto.GetRankRequest{
			Dashboard: "db",
			MemberID:  "user1",
		})

		assert.NoError(t, err)
		assert.Equal(t, int64(42), rank)
	})

	t.Run("not_found", func(t *testing.T) {
		repo.EXPECT().
			ViewMemberRank(ctx, "db", "ghost").
			Return(int64(0), core.ErrMemberNotFound)

		uc := New(repo)

		rank, err := uc.GetMemberRankHandler(ctx, dto.GetRankRequest{
			Dashboard: "db",
			MemberID:  "ghost",
		})

		assert.ErrorIs(t, err, core.ErrMemberNotFound)
		assert.Equal(t, int64(0), rank)
	})
}

func TestGetDashboardStatsHandler(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewDashboardRepository(t)
	uc := New(repo)

	t.Run("should_return_total_count", func(t *testing.T) {
		repo.EXPECT().
			GetTotalMembers(ctx, "global").
			Return(int64(100500), nil)

		count, err := uc.GetDashboardStatsHandler(ctx, "global")

		assert.NoError(t, err)
		assert.Equal(t, int64(100500), count)
	})
}

func TestProcessScoreUpdate(t *testing.T) {
	ctx := context.Background()
	req := dto.IncrementScoreRequest{
		Dashboard: "global_top",
		MemberID:  "player_1",
		Value:     50.0,
	}

	t.Run("success_update_and_get_rank", func(t *testing.T) {
		assertProcessScoreUpdateSuccess(t, ctx, req)
	})

	t.Run("should_stop_if_increment_fails", func(t *testing.T) {
		assertProcessScoreUpdateIncrementError(t, ctx, req)
	})

	t.Run("should_fail_if_rank_query_fails", func(t *testing.T) {
		assertProcessScoreUpdateRankError(t, ctx, req)
	})
}

func assertProcessScoreUpdateSuccess(t *testing.T, ctx context.Context, req dto.IncrementScoreRequest) {
	t.Helper()

	repo := mocks.NewDashboardRepository(t)
	uc := New(repo)

	repo.EXPECT().
		IncrementMemberScore(ctx, req.Dashboard, req.MemberID, req.Value).
		Return(nil).
		Once()

	repo.EXPECT().
		ViewMemberRank(ctx, req.Dashboard, req.MemberID).
		Return(int64(3), nil).
		Once()

	rank, err := uc.ProcessScoreUpdate(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), rank)
}

func assertProcessScoreUpdateIncrementError(t *testing.T, ctx context.Context, req dto.IncrementScoreRequest) {
	t.Helper()

	repo := mocks.NewDashboardRepository(t)
	uc := New(repo)

	repo.EXPECT().
		IncrementMemberScore(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(assert.AnError).
		Once()

	rank, err := uc.ProcessScoreUpdate(ctx, req)
	assert.Error(t, err)
	assert.Equal(t, int64(0), rank)
}

func assertProcessScoreUpdateRankError(t *testing.T, ctx context.Context, req dto.IncrementScoreRequest) {
	t.Helper()

	repo := mocks.NewDashboardRepository(t)
	uc := New(repo)

	repo.EXPECT().
		IncrementMemberScore(ctx, req.Dashboard, req.MemberID, req.Value).
		Return(nil).
		Once()

	repo.EXPECT().
		ViewMemberRank(ctx, req.Dashboard, req.MemberID).
		Return(int64(0), assert.AnError).
		Once()

	rank, err := uc.ProcessScoreUpdate(ctx, req)
	assert.Error(t, err)
	assert.Equal(t, int64(0), rank)
}
