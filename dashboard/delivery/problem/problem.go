package problem

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/jassus213/go-board/dashboard/auth"
	"github.com/jassus213/go-board/dashboard/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultProblemType = "about:blank"

// FromError maps service errors to ProblemDetails.
func FromError(err error, fallbackStatus int, instance string) core.ProblemDetails {
	if err == nil {
		return core.ProblemDetails{
			Type:     defaultProblemType,
			Title:    http.StatusText(fallbackStatus),
			Status:   fallbackStatus,
			Instance: instance,
		}
	}

	switch {
	case errors.Is(err, auth.ErrEmptyToken):
		return core.ProblemDetails{
			Type:     "urn:goboard:auth:missing-token",
			Title:    "Unauthorized",
			Status:   http.StatusUnauthorized,
			Detail:   "authentication token is required",
			Instance: instance,
			Code:     "auth_missing_token",
		}
	case errors.Is(err, auth.ErrInvalidToken), errors.Is(err, auth.ErrTokenExpired), errors.Is(err, auth.ErrInvalidSigningMethod):
		return core.ProblemDetails{
			Type:     "urn:goboard:auth:invalid-token",
			Title:    "Forbidden",
			Status:   http.StatusForbidden,
			Detail:   "authentication token is invalid",
			Instance: instance,
			Code:     "auth_invalid_token",
		}
	case errors.Is(err, core.ErrMemberNotFound):
		return core.ProblemDetails{
			Type:     "urn:goboard:dashboard:member-not-found",
			Title:    "Not Found",
			Status:   http.StatusNotFound,
			Detail:   err.Error(),
			Instance: instance,
			Code:     "member_not_found",
		}
	case errors.Is(err, core.ErrInvalidArgument):
		detail := err.Error()
		detail = strings.TrimPrefix(detail, core.ErrInvalidArgument.Error()+": ")
		return core.ProblemDetails{
			Type:     "urn:goboard:request:invalid-argument",
			Title:    "Bad Request",
			Status:   http.StatusBadRequest,
			Detail:   detail,
			Instance: instance,
			Code:     "invalid_argument",
		}
	case errors.Is(err, context.DeadlineExceeded):
		return core.ProblemDetails{
			Type:     "urn:goboard:request:timeout",
			Title:    "Gateway Timeout",
			Status:   http.StatusGatewayTimeout,
			Detail:   "request timed out",
			Instance: instance,
			Code:     "timeout",
		}
	default:
		return core.ProblemDetails{
			Type:     defaultProblemType,
			Title:    http.StatusText(fallbackStatus),
			Status:   fallbackStatus,
			Detail:   err.Error(),
			Instance: instance,
			Code:     "internal_error",
		}
	}
}

// WriteHTTP writes problem details using RFC7807 media type.
func WriteHTTP(w http.ResponseWriter, pd *core.ProblemDetails) {
	if pd == nil {
		return
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(pd.Status)
	if err := json.NewEncoder(w).Encode(pd); err != nil {
		log.Printf("failed to encode problem details response: %v", err)
	}
}

// GRPCCode maps HTTP status to the nearest gRPC code.
func GRPCCode(httpStatus int) codes.Code {
	switch httpStatus {
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusGatewayTimeout:
		return codes.DeadlineExceeded
	default:
		return codes.Internal
	}
}

// ToGRPCStatus creates a gRPC status from ProblemDetails.
func ToGRPCStatus(pd *core.ProblemDetails) *status.Status {
	if pd == nil {
		return status.New(codes.Internal, http.StatusText(http.StatusInternalServerError))
	}

	msg := pd.Detail
	if msg == "" {
		msg = pd.Title
	}
	return status.New(GRPCCode(pd.Status), msg)
}
