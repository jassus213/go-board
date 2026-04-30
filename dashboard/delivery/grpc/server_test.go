package grpc

import (
	"context"
	"io"
	"testing"

	"github.com/jassus213/go-board/dashboard/core/usecase"
	pb "github.com/jassus213/go-board/dashboard/delivery/grpc/gen"
	"github.com/jassus213/go-board/dashboard/repo/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
