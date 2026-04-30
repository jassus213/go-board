// Package grpc provides the gRPC transport layer for the dashboard service.
// It implements the server-side logic defined in the protobuf files and
// acts as an adapter between the gRPC protocol and the Business Logic Layer (BLL).
package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/jassus213/go-board/dashboard/auth"
	"github.com/jassus213/go-board/dashboard/core"
	"github.com/jassus213/go-board/dashboard/core/dto"
	"github.com/jassus213/go-board/dashboard/core/interfaces"
	pb "github.com/jassus213/go-board/dashboard/delivery/grpc/gen"
	"github.com/jassus213/go-board/dashboard/delivery/problem"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

// memberIDKey is the context key for storing authenticated member ID.
const memberIDKey contextKey = "member_id"

// Server implements the generated DashboardServiceServer interface.
// It holds the required dependencies (like the repository) to execute business commands.
type Server struct {
	pb.UnimplementedDashboardServiceServer
	uc interfaces.DashboardUseCase
}

// NewServer creates a new instance of the gRPC Server with the provided data repository.
func NewServer(uc interfaces.DashboardUseCase) *Server {
	return &Server{uc: uc}
}

// StreamUpdates handles a bidirectional stream of score updates.
//
// Workflow:
// 1. Continuously receives `UpdateRequest` messages from the client.
// 2. Validates the input (Dashboard and MemberID presence).
// 3. Calls the BLL `ProcessScoreUpdate` to atomically increment the score and retrieve the new rank.
// 4. Sends an `UpdateResponse` back to the client with the new rank or an error message.
//
// The loop terminates when the client closes the stream (io.EOF) or a transport error occurs.
func (s *Server) StreamUpdates(stream pb.DashboardService_StreamUpdatesServer) error {
	ctx := stream.Context()

	authID, _ := ctx.Value(memberIDKey).(string)

	for {
		in, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			log.Printf("gRPC Stream read error: %v", err)
			return err
		}

		if err := s.handleUpdateRequest(ctx, stream, authID, in); err != nil {
			log.Printf("gRPC Stream send error: %v", err)
			return err
		}
	}
}

// IncrementScore handles unary score increment and returns updated rank.
func (s *Server) IncrementScore(ctx context.Context, in *pb.IncrementScoreRequest) (*pb.IncrementScoreResponse, error) {
	authID := getAuthMemberID(ctx)
	if in.Dashboard == "" || authID == "" {
		err := fmt.Errorf("%w: missing dashboard or member_id", core.ErrInvalidArgument)
		pd := problem.FromError(err, http.StatusBadRequest, "/dashboard.DashboardService/IncrementScore")
		return nil, withProblemDetails(&pd)
	}

	if in.MemberId != "" && in.MemberId != authID {
		log.Printf("Security warning: user %s tried to update score for %s", authID, in.MemberId)
	}

	rank, err := s.uc.ProcessScoreUpdate(ctx, dto.IncrementScoreRequest{
		Dashboard: in.Dashboard,
		MemberID:  authID,
		Value:     in.Increment,
	})
	if err != nil {
		pd := problem.FromError(err, http.StatusInternalServerError, "/dashboard.DashboardService/IncrementScore")
		return nil, withProblemDetails(&pd)
	}

	return &pb.IncrementScoreResponse{
		MemberId: authID,
		Rank:     rank,
	}, nil
}

// GetMemberRank returns rank for authenticated member.
func (s *Server) GetMemberRank(ctx context.Context, in *pb.GetMemberRankRequest) (*pb.GetMemberRankResponse, error) {
	authID := getAuthMemberID(ctx)
	if in.Dashboard == "" || authID == "" {
		err := fmt.Errorf("%w: missing dashboard or member_id", core.ErrInvalidArgument)
		pd := problem.FromError(err, http.StatusBadRequest, "/dashboard.DashboardService/GetMemberRank")
		return nil, withProblemDetails(&pd)
	}

	if in.MemberId != "" && in.MemberId != authID {
		log.Printf("Security warning: user %s tried to fetch rank for %s", authID, in.MemberId)
	}

	rank, err := s.uc.GetMemberRankHandler(ctx, dto.GetRankRequest{
		Dashboard: in.Dashboard,
		MemberID:  authID,
	})
	if err != nil {
		pd := problem.FromError(err, http.StatusInternalServerError, "/dashboard.DashboardService/GetMemberRank")
		return nil, withProblemDetails(&pd)
	}

	return &pb.GetMemberRankResponse{
		MemberId: authID,
		Rank:     rank,
	}, nil
}

// GetTopMembers returns top N records from dashboard.
func (s *Server) GetTopMembers(ctx context.Context, in *pb.GetTopMembersRequest) (*pb.GetTopMembersResponse, error) {
	if in.Dashboard == "" {
		err := fmt.Errorf("%w: missing dashboard", core.ErrInvalidArgument)
		pd := problem.FromError(err, http.StatusBadRequest, "/dashboard.DashboardService/GetTopMembers")
		return nil, withProblemDetails(&pd)
	}

	records, err := s.uc.GetTopMembersHandler(ctx, dto.GetTopRequest{
		Dashboard: in.Dashboard,
		Limit:     in.Limit,
	})
	if err != nil {
		pd := problem.FromError(err, http.StatusInternalServerError, "/dashboard.DashboardService/GetTopMembers")
		return nil, withProblemDetails(&pd)
	}

	members := make([]*pb.MemberRecord, 0, len(records))
	for _, item := range records {
		members = append(members, &pb.MemberRecord{
			MemberId: item.ID,
			Rank:     item.Rank,
			Score:    item.Score,
		})
	}

	return &pb.GetTopMembersResponse{Members: members}, nil
}

// GetDashboardStats returns dashboard total members.
func (s *Server) GetDashboardStats(ctx context.Context, in *pb.GetDashboardStatsRequest) (*pb.GetDashboardStatsResponse, error) {
	if in.Dashboard == "" {
		err := fmt.Errorf("%w: missing dashboard", core.ErrInvalidArgument)
		pd := problem.FromError(err, http.StatusBadRequest, "/dashboard.DashboardService/GetDashboardStats")
		return nil, withProblemDetails(&pd)
	}

	total, err := s.uc.GetDashboardStatsHandler(ctx, in.Dashboard)
	if err != nil {
		pd := problem.FromError(err, http.StatusInternalServerError, "/dashboard.DashboardService/GetDashboardStats")
		return nil, withProblemDetails(&pd)
	}

	return &pb.GetDashboardStatsResponse{TotalMembers: total}, nil
}

func (s *Server) handleUpdateRequest(
	ctx context.Context,
	stream pb.DashboardService_StreamUpdatesServer,
	authID string,
	in *pb.UpdateRequest,
) error {
	targetID := in.MemberId
	if authID != "" && authID != targetID {
		log.Printf("Security warning: user %s tried to update score for %s", authID, targetID)
	}

	if in.Dashboard == "" || in.MemberId == "" || authID == "" {
		err := fmt.Errorf("%w: missing dashboard or member_id", core.ErrInvalidArgument)
		pd := problem.FromError(err, http.StatusBadRequest, "/dashboard.DashboardService/StreamUpdates")
		return stream.Send(&pb.UpdateResponse{
			MemberId: authID,
			Problem:  toProtoProblem(&pd),
		})
	}

	rank, err := s.uc.ProcessScoreUpdate(ctx, dto.IncrementScoreRequest{
		Dashboard: in.Dashboard,
		MemberID:  authID,
		Value:     in.Increment,
	})

	resp := &pb.UpdateResponse{MemberId: authID}
	if err != nil {
		pd := problem.FromError(err, http.StatusInternalServerError, "/dashboard.DashboardService/StreamUpdates")
		resp.Problem = toProtoProblem(&pd)
	} else {
		resp.Rank = rank
	}

	return stream.Send(resp)
}

// AuthInterceptor returns a gRPC stream interceptor that validates tokens using the provided verifier.
// It expects the token in the "authorization" metadata field (e.g., "Bearer <token>").
func AuthInterceptor(verifier auth.Verifier) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		token, err := tokenFromMetadata(ss.Context())
		if err != nil {
			pd := problem.FromError(err, http.StatusUnauthorized, info.FullMethod)
			return withProblemDetails(&pd)
		}

		memberID, err := verifier.Verify(ss.Context(), token)
		if err != nil {
			log.Printf("gRPC auth failed: %v", err)
			pd := problem.FromError(err, http.StatusForbidden, info.FullMethod)
			return withProblemDetails(&pd)
		}

		wrapped := &authenticatedStream{
			ServerStream: ss,
			memberID:     memberID,
		}

		return handler(srv, wrapped)
	}
}

// AuthUnaryInterceptor validates token and enriches context with member ID.
func AuthUnaryInterceptor(verifier auth.Verifier) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		token, err := tokenFromMetadata(ctx)
		if err != nil {
			pd := problem.FromError(err, http.StatusUnauthorized, info.FullMethod)
			return nil, withProblemDetails(&pd)
		}

		memberID, err := verifier.Verify(ctx, token)
		if err != nil {
			log.Printf("gRPC auth failed: %v", err)
			pd := problem.FromError(err, http.StatusForbidden, info.FullMethod)
			return nil, withProblemDetails(&pd)
		}

		return handler(context.WithValue(ctx, memberIDKey, memberID), req)
	}
}

// authenticatedStream wraps grpc.ServerStream to allow context modification.
type authenticatedStream struct {
	grpc.ServerStream
	memberID string
}

// Context overrides the standard stream context to include the verified memberID.
func (s *authenticatedStream) Context() context.Context {
	return context.WithValue(s.ServerStream.Context(), memberIDKey, s.memberID)
}

func getAuthMemberID(ctx context.Context) string {
	authID, _ := ctx.Value(memberIDKey).(string)
	return authID
}

func tokenFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", auth.ErrEmptyToken
	}

	values := md["authorization"]
	if len(values) == 0 {
		return "", auth.ErrEmptyToken
	}

	token := strings.TrimPrefix(values[0], "Bearer ")
	return token, nil
}

func toProtoProblem(pd *core.ProblemDetails) *pb.ProblemDetails {
	if pd == nil {
		return nil
	}

	return &pb.ProblemDetails{
		Type:     pd.Type,
		Title:    pd.Title,
		Status:   int32(pd.Status),
		Detail:   pd.Detail,
		Instance: pd.Instance,
		Code:     pd.Code,
	}
}

func withProblemDetails(pd *core.ProblemDetails) error {
	st := problem.ToGRPCStatus(pd)
	stWithDetails, err := st.WithDetails(toProtoProblem(pd))
	if err != nil {
		return st.Err()
	}
	return stWithDetails.Err()
}
