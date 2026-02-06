package bll

import (
	"context"
	"testing"

	"github.com/jassus213/go-board/dashboard/dal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestProcessScoreUpdate(t *testing.T) {
	ctx := context.Background()
	req := IncrementScoreRequest{
		Dashboard: "global_top",
		MemberID:  "player_1",
		Value:     50.0,
	}

	t.Run("success_update_and_get_rank", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)

		repo.EXPECT().
			IncrementMemberScore(ctx, req.Dashboard, req.MemberID, req.Value).
			Return(nil).
			Once()

		repo.EXPECT().
			ViewMemberRank(ctx, req.Dashboard, req.MemberID).
			Return(int64(3), nil).
			Once()

		rank, err := ProcessScoreUpdate(ctx, repo, req)

		assert.NoError(t, err)
		assert.Equal(t, int64(3), rank)
	})

	t.Run("should_stop_if_increment_fails", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)

		repo.EXPECT().
			IncrementMemberScore(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(assert.AnError).
			Once()

		rank, err := ProcessScoreUpdate(ctx, repo, req)

		assert.Error(t, err)
		assert.Equal(t, int64(0), rank)
	})

	t.Run("should_fail_if_rank_query_fails", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)

		repo.EXPECT().
			IncrementMemberScore(ctx, req.Dashboard, req.MemberID, req.Value).
			Return(nil).
			Once()

		repo.EXPECT().
			ViewMemberRank(ctx, req.Dashboard, req.MemberID).
			Return(int64(0), assert.AnError).
			Once()

		rank, err := ProcessScoreUpdate(ctx, repo, req)

		assert.Error(t, err)
		assert.Equal(t, int64(0), rank)
	})
}
