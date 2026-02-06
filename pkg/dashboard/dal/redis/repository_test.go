package redis

import (
	"context"
	"testing"

	"github.com/jassus213/GoBoard/dashboard/domain"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type DashboardTestSuite struct {
	suite.Suite
	mockRedis *miniredis.Miniredis
	repo      *DashboardRedisRepository
	ctx       context.Context
}

func (s *DashboardTestSuite) SetupTest() {
	mr, err := miniredis.Run()
	assert.NoError(s.T(), err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	s.mockRedis = mr
	s.repo = NewDashboardRedisRepository(client, "test:")
	s.ctx = context.Background()
}

func (s *DashboardTestSuite) TearDownTest() {
	s.mockRedis.Close()
}

func TestDashboardTestSuite(t *testing.T) {
	suite.Run(t, new(DashboardTestSuite))
}

func (s *DashboardTestSuite) TestAddAndGetTotal() {
	err := s.repo.AddMemberToDashboard(s.ctx, "leaderboard", "user1", 100)
	assert.NoError(s.T(), err)

	count, err := s.repo.GetTotalMembers(s.ctx, "leaderboard")
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1), count)
}

func (s *DashboardTestSuite) TestAddMembersBatch() {
	members := []domain.DashboardRecord{
		{ID: "u1", Score: 10},
		{ID: "u2", Score: 20},
		{ID: "u3", Score: 30},
	}

	err := s.repo.AddMembersBatch(s.ctx, "batch_db", members)
	assert.NoError(s.T(), err)

	count, _ := s.repo.GetTotalMembers(s.ctx, "batch_db")
	assert.Equal(s.T(), int64(3), count)
}

func (s *DashboardTestSuite) TestViewMemberRank() {
	_ = s.repo.AddMemberToDashboard(s.ctx, "ranks", "loser", 10)
	_ = s.repo.AddMemberToDashboard(s.ctx, "ranks", "winner", 100)
	_ = s.repo.AddMemberToDashboard(s.ctx, "ranks", "middle", 50)

	rank, err := s.repo.ViewMemberRank(s.ctx, "ranks", "winner")
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1), rank)

	rank, err = s.repo.ViewMemberRank(s.ctx, "ranks", "loser")
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(3), rank)

	_, err = s.repo.ViewMemberRank(s.ctx, "ranks", "ghost")
	assert.ErrorIs(s.T(), err, domain.ErrMemberNotFound)
}

func (s *DashboardTestSuite) TestGetTopMembers() {
	_ = s.repo.AddMemberToDashboard(s.ctx, "top", "u1", 10)
	_ = s.repo.AddMemberToDashboard(s.ctx, "top", "u2", 30)
	_ = s.repo.AddMemberToDashboard(s.ctx, "top", "u3", 20)

	top, err := s.repo.GetTopMembers(s.ctx, "top", 2)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), top, 2)

	assert.Equal(s.T(), "u2", top[0].ID)
	assert.Equal(s.T(), int64(1), top[0].Rank)
	assert.Equal(s.T(), float64(30), top[0].Score)

	assert.Equal(s.T(), "u3", top[1].ID)
	assert.Equal(s.T(), int64(2), top[1].Rank)
}

func (s *DashboardTestSuite) TestIncrementScore() {
	_ = s.repo.AddMemberToDashboard(s.ctx, "inc", "player", 50)

	err := s.repo.IncrementMemberScore(s.ctx, "inc", "player", 15.5)
	assert.NoError(s.T(), err)

	top, _ := s.repo.GetTopMembers(s.ctx, "inc", 1)
	assert.Equal(s.T(), 65.5, top[0].Score)
}

func (s *DashboardTestSuite) TestDeleteDashboard() {
	_ = s.repo.AddMemberToDashboard(s.ctx, "to_delete", "u1", 10)

	err := s.repo.DeleteDashboard(s.ctx, "to_delete")
	assert.NoError(s.T(), err)

	count, _ := s.repo.GetTotalMembers(s.ctx, "to_delete")
	assert.Equal(s.T(), int64(0), count)
}

func (s *DashboardTestSuite) TestRemoveMember() {
	_ = s.repo.AddMemberToDashboard(s.ctx, "rem", "u1", 10)
	_ = s.repo.AddMemberToDashboard(s.ctx, "rem", "u2", 20)

	err := s.repo.RemoveMemberFromDashboard(s.ctx, "rem", "u1")
	assert.NoError(s.T(), err)

	count, _ := s.repo.GetTotalMembers(s.ctx, "rem")
	assert.Equal(s.T(), int64(1), count)
}
