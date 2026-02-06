// Package grpc provides the gRPC transport layer for the dashboard service.
// It implements the server-side logic defined in the protobuf files and
// acts as an adapter between the gRPC protocol and the Business Logic Layer (BLL).
package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/jassus213/go-board/dashboard/auth"
	"github.com/jassus213/go-board/dashboard/bll"
	"github.com/jassus213/go-board/dashboard/dal"

	"io"
	"log"

	pb "github.com/jassus213/go-board/dashboard/delivery/grpc/gen"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Server implements the generated DashboardServiceServer interface.
// It holds the required dependencies (like the repository) to execute business commands.
type Server struct {
	pb.UnimplementedDashboardServiceServer
	repo dal.DashboardRepository
}

// NewServer creates a new instance of the gRPC Server with the provided data repository.
func NewServer(repo dal.DashboardRepository) *Server {
	return &Server{repo: repo}
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

	authID, _ := ctx.Value("member_id").(string)

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			log.Printf("gRPC Stream read error: %v", err)
			return err
		}

		targetID := in.MemberId
		if authID != "" && authID != targetID {
			log.Printf("Security warning: user %s tried to update score for %s", authID, targetID)
		}

		if in.Dashboard == "" || in.MemberId == "" || authID == "" {
			if sendErr := stream.Send(&pb.UpdateResponse{
				MemberId: authID,
				Error:    "missing dashboard or member_id",
			}); sendErr != nil {
				return sendErr
			}
			continue
		}

		rank, err := bll.ProcessScoreUpdate(ctx, s.repo, bll.IncrementScoreRequest{
			Dashboard: in.Dashboard,
			MemberID:  authID,
			Value:     in.Increment,
		})

		resp := &pb.UpdateResponse{
			MemberId: authID,
		}

		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Rank = rank
		}

		if err := stream.Send(resp); err != nil {
			log.Printf("gRPC Stream send error: %v", err)
			return err
		}
	}
}

// AuthInterceptor returns a gRPC stream interceptor that validates tokens using the provided verifier.
// It expects the token in the "authorization" metadata field (e.g., "Bearer <token>").
func AuthInterceptor(verifier auth.Verifier) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return errors.New("missing metadata")
		}

		values := md["authorization"]
		if len(values) == 0 {
			return errors.New("missing authorization token")
		}

		token := values[0]
		token = strings.TrimPrefix(token, "Bearer ")

		memberID, err := verifier.Verify(ss.Context(), token)
		if err != nil {
			log.Printf("gRPC auth failed: %v", err)
			return err
		}

		wrapped := &authenticatedStream{
			ServerStream: ss,
			memberID:     memberID,
		}

		return handler(srv, wrapped)
	}
}

// authenticatedStream wraps grpc.ServerStream to allow context modification.
type authenticatedStream struct {
	grpc.ServerStream
	memberID string
}

// Context overrides the standard stream context to include the verified memberID.
func (s *authenticatedStream) Context() context.Context {
	return context.WithValue(s.ServerStream.Context(), "member_id", s.memberID)
}
