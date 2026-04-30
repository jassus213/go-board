// Package redis provides a Redis-based implementation of the dashboard repository.
// It uses Sorted Sets (ZSET) to maintain member rankings and scores efficiently.
package redis

import (
	"context"
	"errors"

	"github.com/jassus213/go-board/dashboard/core"
	"github.com/jassus213/go-board/dashboard/core/entity"
	goredis "github.com/redis/go-redis/v9"
)

// DashboardRedisRepository implements domain.DashboardRepository using Redis.
// It supports namespacing via a prefix to avoid key collisions.
type DashboardRedisRepository struct {
	client *goredis.Client
	prefix string
}

// NewDashboardRedisRepository creates a new instance of DashboardRedisRepository.
// The prefix is prepended to all dashboard keys (e.g., "prefix:dashboard_name").
func NewDashboardRedisRepository(client *goredis.Client, prefix string) *DashboardRedisRepository {
	return &DashboardRedisRepository{
		client: client,
		prefix: prefix,
	}
}

// AddMembersBatch adds multiple members to a dashboard in a single Redis operation.
// This is more efficient than calling AddMemberToDashboard multiple times.
// If a member already exists, their score is updated.
func (r *DashboardRedisRepository) AddMembersBatch(ctx context.Context, dashboard string, members []entity.DashboardRecord) error {
	if len(members) == 0 {
		return nil
	}

	zMembers := make([]goredis.Z, len(members))
	for i, m := range members {
		zMembers[i] = goredis.Z{
			Score:  m.Score,
			Member: m.ID,
		}
	}

	return r.client.ZAdd(ctx, r.prefix+dashboard, zMembers...).Err()
}

// AddMemberToDashboard adds a single member with a score to the specified dashboard.
// If the member already exists, their score is overwritten.
func (r *DashboardRedisRepository) AddMemberToDashboard(ctx context.Context, dashboard, member string, score float64) error {
	return r.client.ZAdd(ctx, r.prefix+dashboard, goredis.Z{Score: score, Member: member}).Err()
}

// RemoveMemberFromDashboard removes a member from the specified dashboard.
// If the member does not exist, no error is returned.
func (r *DashboardRedisRepository) RemoveMemberFromDashboard(ctx context.Context, dashboard, member string) error {
	return r.client.ZRem(ctx, r.prefix+dashboard, member).Err()
}

// ViewMemberRank returns the 1-based rank of a member based on descending scores.
// It returns domain.ErrMemberNotFound if the member does not exist in the dashboard.
func (r *DashboardRedisRepository) ViewMemberRank(ctx context.Context, dashboard string, memberId string) (int64, error) {
	rank, err := r.client.ZRevRank(ctx, r.prefix+dashboard, memberId).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return 0, core.ErrMemberNotFound
		}
		return 0, err
	}

	return rank + 1, nil
}

// GetTopMembers retrieves the top N members from the dashboard, sorted by score descending.
// If top is 10, it returns the top 10 members (ranks 1 to 10).
func (r *DashboardRedisRepository) GetTopMembers(ctx context.Context, dashboard string, top int64) ([]entity.DashboardRecord, error) {
	if top <= 0 {
		return nil, nil
	}

	result, err := r.client.ZRevRangeWithScores(ctx, r.prefix+dashboard, 0, top-1).Result()
	if err != nil {
		return nil, err
	}

	records := make([]entity.DashboardRecord, len(result))
	for i, memberData := range result {
		records[i] = entity.DashboardRecord{
			ID:    memberData.Member.(string),
			Score: memberData.Score,
			Rank:  int64(i + 1),
		}
	}

	return records, nil
}

// IncrementMemberScore adds the given value to the existing score of a member.
// If the member does not exist, they are added to the dashboard with the increment as their initial score.
func (r *DashboardRedisRepository) IncrementMemberScore(ctx context.Context, dashboard, member string, increment float64) error {
	return r.client.ZIncrBy(ctx, r.prefix+dashboard, increment, member).Err()
}

// GetTotalMembers returns the total number of participants in the specified dashboard.
func (r *DashboardRedisRepository) GetTotalMembers(ctx context.Context, dashboard string) (int64, error) {
	return r.client.ZCard(ctx, r.prefix+dashboard).Result()
}

// DeleteDashboard removes the entire dashboard and all associated member data from Redis.
func (r *DashboardRedisRepository) DeleteDashboard(ctx context.Context, dashboard string) error {
	return r.client.Del(ctx, r.prefix+dashboard).Err()
}

// IncrementMembersBatch updates multiple members' scores by their respective increments using Redis Pipelining.
// Each member's score is increased by the value provided in the DashboardRecord.Score field.
// If a member doesn't exist, they are added with the increment as their initial score.
func (r *DashboardRedisRepository) IncrementMembersBatch(ctx context.Context, dashboard string, increments []entity.DashboardRecord) error {
	if len(increments) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()

	for _, m := range increments {
		pipe.ZIncrBy(ctx, r.prefix+dashboard, m.Score, m.ID)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
