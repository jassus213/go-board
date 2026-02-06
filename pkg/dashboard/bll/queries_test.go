package bll

import (
	"context"
	"testing"

	"github.com/jassus213/go-board/dashboard/dal/mocks"
	"github.com/jassus213/go-board/dashboard/domain"
	"github.com/stretchr/testify/assert"
)

func TestGetTopMembersHandler(t *testing.T) {
	ctx := context.Background()
	dashboard := "season_1"

	t.Run("should_use_provided_limit", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)
		expectedRecords := []domain.DashboardRecord{
			{ID: "user1", Score: 100, Rank: 1},
		}

		repo.EXPECT().
			GetTopMembers(ctx, dashboard, int64(5)).
			Return(expectedRecords, nil)

		res, err := GetTopMembersHandler(ctx, repo, GetTopRequest{
			Dashboard: dashboard,
			Limit:     5,
		})

		assert.NoError(t, err)
		assert.Equal(t, expectedRecords, res)
	})

	t.Run("should_default_to_10_if_limit_is_zero_or_less", func(t *testing.T) {
		repo := mocks.NewDashboardRepository(t)

		repo.EXPECT().
			GetTopMembers(ctx, dashboard, int64(10)).
			Return(nil, nil)

		_, err := GetTopMembersHandler(ctx, repo, GetTopRequest{
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

		rank, err := GetMemberRankHandler(ctx, repo, GetRankRequest{
			Dashboard: "db",
			MemberID:  "user1",
		})

		assert.NoError(t, err)
		assert.Equal(t, int64(42), rank)
	})

	t.Run("not_found", func(t *testing.T) {
		repo.EXPECT().
			ViewMemberRank(ctx, "db", "ghost").
			Return(int64(0), domain.ErrMemberNotFound)

		rank, err := GetMemberRankHandler(ctx, repo, GetRankRequest{
			Dashboard: "db",
			MemberID:  "ghost",
		})

		assert.ErrorIs(t, err, domain.ErrMemberNotFound)
		assert.Equal(t, int64(0), rank)
	})
}

func TestGetDashboardStatsHandler(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewDashboardRepository(t)

	t.Run("should_return_total_count", func(t *testing.T) {
		repo.EXPECT().
			GetTotalMembers(ctx, "global").
			Return(int64(100500), nil)

		count, err := GetDashboardStatsHandler(ctx, repo, "global")

		assert.NoError(t, err)
		assert.Equal(t, int64(100500), count)
	})
}
