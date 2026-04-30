package grpc

import (
	"context"
	"io"
	"testing"

	"github.com/jassus213/go-board/dashboard/core"
	"github.com/jassus213/go-board/dashboard/core/entity"
	"github.com/jassus213/go-board/dashboard/core/usecase"
	pb "github.com/jassus213/go-board/dashboard/delivery/grpc/gen"
	"github.com/jassus213/go-board/dashboard/repo/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockStream struct {
	pb.DashboardService_StreamUpdatesServer
	mock.Mock
	ctx context.Context
}

func (m *mockStream) Recv() (*pb.UpdateRequest, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.UpdateRequest), args.Error(1)
}

func (m *mockStream) Send(resp *pb.UpdateResponse) error {
	return m.Called(resp).Error(0)
}

func (m *mockStream) Context() context.Context {
	return m.ctx
}

func TestStreamUpdates(t *testing.T) {
	repo := mocks.NewDashboardRepository(t)
	uc := usecase.New(repo)
	srv := NewServer(uc)

	t.Run("success_stream_cycle", func(t *testing.T) {
		testAuthID := "user_1"
		ctx := context.WithValue(context.Background(), memberIDKey, testAuthID)

		stream := &mockStream{ctx: ctx}

		req := &pb.UpdateRequest{
			Dashboard: "test_db",
			MemberId:  "user_1",
			Increment: 10.5,
		}

		stream.On("Recv").Return(req, nil).Once()
		stream.On("Recv").Return(nil, io.EOF).Once()

		repo.EXPECT().
			IncrementMemberScore(mock.Anything, "test_db", "user_1", 10.5).
			Return(nil).
			Once()

		repo.EXPECT().
			ViewMemberRank(mock.Anything, "test_db", "user_1").
			Return(int64(1), nil).
			Once()

		stream.On("Send", mock.MatchedBy(func(r *pb.UpdateResponse) bool {
			return r.Rank == 1 && r.MemberId == testAuthID && r.Problem == nil
		})).Return(nil).Once()

		err := srv.StreamUpdates(stream)

		assert.NoError(t, err)
		stream.AssertExpectations(t)
	})

	t.Run("fail_on_unauthorized_context", func(t *testing.T) {
		stream := &mockStream{ctx: context.Background()}
		req := &pb.UpdateRequest{Dashboard: "db", MemberId: "u1"}

		stream.On("Recv").Return(req, nil).Once()
		stream.On("Recv").Return(nil, io.EOF).Once()

		stream.On("Send", mock.MatchedBy(func(r *pb.UpdateResponse) bool {
			return r.Problem != nil &&
				r.Problem.Code == "invalid_argument" &&
				r.Problem.Status == 400 &&
				r.Problem.Detail == "missing dashboard or member_id"
		})).Return(nil).Once()

		err := srv.StreamUpdates(stream)
		assert.NoError(t, err)
	})
}

func TestStreamUpdates_Errors(t *testing.T) {
	repo := mocks.NewDashboardRepository(t)
	uc := usecase.New(repo)
	srv := NewServer(uc)

	t.Run("recv_error_stops_stream", func(t *testing.T) {
		stream := &mockStream{ctx: context.Background()}
		stream.On("Recv").Return(nil, assert.AnError).Once()

		err := srv.StreamUpdates(stream)
		assert.Error(t, err)
	})

	t.Run("send_error_stops_stream", func(t *testing.T) {
		testAuthID := "user_1"
		ctx := context.WithValue(context.Background(), memberIDKey, testAuthID)
		stream := &mockStream{ctx: ctx}

		req := &pb.UpdateRequest{
			Dashboard: "test_db",
			MemberId:  "user_1",
			Increment: 2,
		}

		stream.On("Recv").Return(req, nil).Once()
		repo.EXPECT().
			IncrementMemberScore(mock.Anything, "test_db", "user_1", 2.0).
			Return(nil).
			Once()
		repo.EXPECT().
			ViewMemberRank(mock.Anything, "test_db", "user_1").
			Return(int64(3), nil).
			Once()
		stream.On("Send", mock.AnythingOfType("*gen.UpdateResponse")).Return(assert.AnError).Once()

		err := srv.StreamUpdates(stream)
		assert.Error(t, err)
	})

	t.Run("usecase_error_is_returned_in_response", func(t *testing.T) {
		testAuthID := "user_1"
		ctx := context.WithValue(context.Background(), memberIDKey, testAuthID)
		stream := &mockStream{ctx: ctx}

		req := &pb.UpdateRequest{
			Dashboard: "test_db",
			MemberId:  "user_1",
			Increment: 1,
		}

		stream.On("Recv").Return(req, nil).Once()
		stream.On("Recv").Return(nil, io.EOF).Once()
		repo.EXPECT().
			IncrementMemberScore(mock.Anything, "test_db", "user_1", 1.0).
			Return(assert.AnError).
			Once()
		stream.On("Send", mock.MatchedBy(func(r *pb.UpdateResponse) bool {
			return r.MemberId == testAuthID &&
				r.Rank == 0 &&
				r.Problem != nil &&
				r.Problem.Code == "internal_error" &&
				r.Problem.Status == 500
		})).Return(nil).Once()

		err := srv.StreamUpdates(stream)
		assert.NoError(t, err)
	})
}

func TestUnaryMethods(t *testing.T) {
	repo := mocks.NewDashboardRepository(t)
	uc := usecase.New(repo)
	srv := NewServer(uc)
	ctx := context.WithValue(context.Background(), memberIDKey, "user_1")

	t.Run("increment_score_success", func(t *testing.T) {
		repo.EXPECT().
			IncrementMemberScore(mock.Anything, "games", "user_1", 10.0).
			Return(nil).
			Once()
		repo.EXPECT().
			ViewMemberRank(mock.Anything, "games", "user_1").
			Return(int64(2), nil).
			Once()

		resp, err := srv.IncrementScore(ctx, &pb.IncrementScoreRequest{
			Dashboard: "games",
			MemberId:  "other_user",
			Increment: 10,
		})
		assert.NoError(t, err)
		assert.Equal(t, int64(2), resp.Rank)
		assert.Equal(t, "user_1", resp.MemberId)
	})

	t.Run("get_top_members_success", func(t *testing.T) {
		repo.EXPECT().
			GetTopMembers(mock.Anything, "games", int64(3)).
			Return([]entity.DashboardRecord{
				{ID: "u1", Rank: 1, Score: 10},
				{ID: "u2", Rank: 2, Score: 8},
			}, nil).
			Once()

		resp, err := srv.GetTopMembers(ctx, &pb.GetTopMembersRequest{
			Dashboard: "games",
			Limit:     3,
		})
		assert.NoError(t, err)
		if assert.Len(t, resp.Members, 2) {
			assert.Equal(t, "u1", resp.Members[0].MemberId)
			assert.Equal(t, int64(1), resp.Members[0].Rank)
		}
	})

	t.Run("invalid_argument_error", func(t *testing.T) {
		_, err := srv.GetDashboardStats(ctx, &pb.GetDashboardStatsRequest{})
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

func TestGetMemberRank(t *testing.T) {
	repo := mocks.NewDashboardRepository(t)
	uc := usecase.New(repo)
	srv := NewServer(uc)
	ctx := context.WithValue(context.Background(), memberIDKey, "auth_user")

	t.Run("success_uses_authenticated_member", func(t *testing.T) {
		repo.EXPECT().
			ViewMemberRank(mock.Anything, "games", "auth_user").
			Return(int64(4), nil).
			Once()

		resp, err := srv.GetMemberRank(ctx, &pb.GetMemberRankRequest{
			Dashboard: "games",
			MemberId:  "other_user",
		})
		assert.NoError(t, err)
		assert.Equal(t, "auth_user", resp.MemberId)
		assert.Equal(t, int64(4), resp.Rank)
	})

	t.Run("missing_dashboard_returns_invalid_argument", func(t *testing.T) {
		_, err := srv.GetMemberRank(ctx, &pb.GetMemberRankRequest{})
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("usecase_error_returns_internal", func(t *testing.T) {
		repo.EXPECT().
			ViewMemberRank(mock.Anything, "games", "auth_user").
			Return(int64(0), assert.AnError).
			Once()

		_, err := srv.GetMemberRank(ctx, &pb.GetMemberRankRequest{Dashboard: "games"})
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})
}

func TestTopAndStatsErrorPaths(t *testing.T) {
	repo := mocks.NewDashboardRepository(t)
	uc := usecase.New(repo)
	srv := NewServer(uc)
	ctx := context.WithValue(context.Background(), memberIDKey, "auth_user")

	t.Run("get_top_members_usecase_error", func(t *testing.T) {
		repo.EXPECT().
			GetTopMembers(mock.Anything, "games", int64(10)).
			Return(nil, assert.AnError).
			Once()

		_, err := srv.GetTopMembers(ctx, &pb.GetTopMembersRequest{
			Dashboard: "games",
			Limit:     10,
		})
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})

	t.Run("get_top_members_invalid_argument", func(t *testing.T) {
		_, err := srv.GetTopMembers(ctx, &pb.GetTopMembersRequest{})
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("get_dashboard_stats_success", func(t *testing.T) {
		repo.EXPECT().
			GetTotalMembers(mock.Anything, "games").
			Return(int64(99), nil).
			Once()

		resp, err := srv.GetDashboardStats(ctx, &pb.GetDashboardStatsRequest{Dashboard: "games"})
		assert.NoError(t, err)
		assert.Equal(t, int64(99), resp.TotalMembers)
	})

	t.Run("get_dashboard_stats_usecase_error", func(t *testing.T) {
		repo.EXPECT().
			GetTotalMembers(mock.Anything, "games").
			Return(int64(0), assert.AnError).
			Once()

		_, err := srv.GetDashboardStats(ctx, &pb.GetDashboardStatsRequest{Dashboard: "games"})
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})
}

func TestProblemHelpers(t *testing.T) {
	t.Run("to_proto_problem_nil", func(t *testing.T) {
		assert.Nil(t, toProtoProblem(nil))
	})

	t.Run("with_problem_details_returns_grpc_error", func(t *testing.T) {
		err := withProblemDetails(&core.ProblemDetails{
			Status: 400,
			Title:  "Bad Request",
			Detail: "invalid input",
			Code:   "invalid_argument",
		})
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}
