package bll

import (
	"context"
	"testing"

	"github.com/jassus213/GoBoard/dashboard/dal/mocks"
	"github.com/jassus213/GoBoard/dashboard/domain"
	"github.com/stretchr/testify/assert"
)

func TestAddMemberHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("should_fail_if_dashboard_is_empty", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)
		req := AddMemberRequest{Dashboard: "", MemberID: "u1", Score: 10}

		err := AddMemberHandler(ctx, repo, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dashboard and member ID are required")
	})

	t.Run("should_call_repo_on_success", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)
		req := AddMemberRequest{Dashboard: "db1", MemberID: "u1", Score: 100}

		repo.EXPECT().
			AddMemberToDashboard(ctx, "db1", "u1", float64(100)).
			Return(nil)

		err := AddMemberHandler(ctx, repo, req)
		assert.NoError(t, err)
	})
}

func TestBatchAddHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("should_return_nil_without_calling_repo_if_empty", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)
		req := BatchAddRequest{Dashboard: "db1", Members: []domain.DashboardRecord{}}

		err := BatchAddHandler(ctx, repo, req)
		assert.NoError(t, err)
	})

	t.Run("should_call_batch_add_on_repo", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)
		mems := []domain.DashboardRecord{{ID: "u1", Score: 10}}
		req := BatchAddRequest{Dashboard: "db1", Members: mems}

		repo.EXPECT().
			AddMembersBatch(ctx, "db1", mems).
			Return(nil)

		err := BatchAddHandler(ctx, repo, req)
		assert.NoError(t, err)
	})
}

func TestIncrementScoreHandler(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewDashboardRepository(t)
	req := IncrementScoreRequest{Dashboard: "db1", MemberID: "u1", Value: 5.5}

	repo.EXPECT().
		IncrementMemberScore(ctx, "db1", "u1", 5.5).
		Return(nil)

	err := IncrementScoreHandler(ctx, repo, req)
	assert.NoError(t, err)
}

func TestRemoveMemberHandler(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewDashboardRepository(t)

	repo.EXPECT().
		RemoveMemberFromDashboard(ctx, "db1", "u1").
		Return(nil)

	err := RemoveMemberHandler(ctx, repo, "db1", "u1")
	assert.NoError(t, err)
}

func TestDeleteDashboardHandler(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewDashboardRepository(t)

	repo.EXPECT().
		DeleteDashboard(ctx, "db1").
		Return(nil)

	err := DeleteDashboardHandler(ctx, repo, "db1")
	assert.NoError(t, err)
}

func TestBatchIncrementHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("should_skip_repo_if_empty", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)
		err := BatchIncrementHandler(ctx, repo, BatchAddRequest{Members: nil})
		assert.NoError(t, err)
	})

	t.Run("should_call_increment_batch", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)
		mems := []domain.DashboardRecord{{ID: "u1", Score: 1}}

		repo.EXPECT().
			IncrementMembersBatch(ctx, "db1", mems).
			Return(nil)

		err := BatchIncrementHandler(ctx, repo, BatchAddRequest{Dashboard: "db1", Members: mems})
		assert.NoError(t, err)
	})
}
