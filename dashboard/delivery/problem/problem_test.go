package problem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/jassus213/go-board/dashboard/auth"
	"github.com/jassus213/go-board/dashboard/core"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

type failingResponseWriter struct {
	header http.Header
	status int
}

func (f *failingResponseWriter) Header() http.Header {
	if f.header == nil {
		f.header = make(http.Header)
	}
	return f.header
}

func (f *failingResponseWriter) WriteHeader(statusCode int) {
	f.status = statusCode
}

func (f *failingResponseWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestFromError(t *testing.T) {
	instance := "/api/v1/resource"

	t.Run("nil_error_uses_fallback_status", func(t *testing.T) {
		pd := FromError(nil, http.StatusTeapot, instance)
		assert.Equal(t, http.StatusTeapot, pd.Status)
		assert.Equal(t, "I'm a teapot", pd.Title)
		assert.Equal(t, instance, pd.Instance)
		assert.Equal(t, "about:blank", pd.Type)
	})

	t.Run("maps_auth_missing_token", func(t *testing.T) {
		pd := FromError(auth.ErrEmptyToken, http.StatusInternalServerError, instance)
		assert.Equal(t, http.StatusUnauthorized, pd.Status)
		assert.Equal(t, "auth_missing_token", pd.Code)
	})

	t.Run("maps_auth_invalid_variants", func(t *testing.T) {
		for _, err := range []error{auth.ErrInvalidToken, auth.ErrTokenExpired, auth.ErrInvalidSigningMethod} {
			pd := FromError(err, http.StatusInternalServerError, instance)
			assert.Equal(t, http.StatusForbidden, pd.Status)
			assert.Equal(t, "auth_invalid_token", pd.Code)
		}
	})

	t.Run("maps_member_not_found", func(t *testing.T) {
		pd := FromError(core.ErrMemberNotFound, http.StatusInternalServerError, instance)
		assert.Equal(t, http.StatusNotFound, pd.Status)
		assert.Equal(t, "member_not_found", pd.Code)
	})

	t.Run("maps_invalid_argument_and_trims_prefix", func(t *testing.T) {
		err := fmt.Errorf("%w: bad limit", core.ErrInvalidArgument)
		pd := FromError(err, http.StatusInternalServerError, instance)
		assert.Equal(t, http.StatusBadRequest, pd.Status)
		assert.Equal(t, "invalid_argument", pd.Code)
		assert.Equal(t, "bad limit", pd.Detail)
	})

	t.Run("maps_timeout", func(t *testing.T) {
		pd := FromError(context.DeadlineExceeded, http.StatusInternalServerError, instance)
		assert.Equal(t, http.StatusGatewayTimeout, pd.Status)
		assert.Equal(t, "timeout", pd.Code)
	})

	t.Run("fallback_internal_error", func(t *testing.T) {
		pd := FromError(errors.New("boom"), http.StatusInternalServerError, instance)
		assert.Equal(t, http.StatusInternalServerError, pd.Status)
		assert.Equal(t, "internal_error", pd.Code)
		assert.Equal(t, "boom", pd.Detail)
	})
}

func TestWriteHTTP(t *testing.T) {
	t.Run("nil_problem_is_noop", func(t *testing.T) {
		w := &failingResponseWriter{}
		WriteHTTP(w, nil)
		assert.Equal(t, 0, w.status)
	})

	t.Run("writes_problem_payload", func(t *testing.T) {
		rec := &failingResponseWriter{}
		pd := &core.ProblemDetails{
			Type:   "urn:test",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "wrong input",
			Code:   "invalid_argument",
		}
		WriteHTTP(rec, pd)
		assert.Equal(t, http.StatusBadRequest, rec.status)
		assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
	})

	t.Run("encode_error_path_is_safe", func(t *testing.T) {
		rec := &failingResponseWriter{}
		pd := &core.ProblemDetails{Status: http.StatusInternalServerError}
		WriteHTTP(rec, pd)
		assert.Equal(t, http.StatusInternalServerError, rec.status)
	})
}

func TestGRPCCode(t *testing.T) {
	assert.Equal(t, codes.InvalidArgument, GRPCCode(http.StatusBadRequest))
	assert.Equal(t, codes.Unauthenticated, GRPCCode(http.StatusUnauthorized))
	assert.Equal(t, codes.PermissionDenied, GRPCCode(http.StatusForbidden))
	assert.Equal(t, codes.NotFound, GRPCCode(http.StatusNotFound))
	assert.Equal(t, codes.DeadlineExceeded, GRPCCode(http.StatusGatewayTimeout))
	assert.Equal(t, codes.Internal, GRPCCode(http.StatusConflict))
}

func TestToGRPCStatus(t *testing.T) {
	t.Run("nil_problem_returns_internal_default", func(t *testing.T) {
		st := ToGRPCStatus(nil)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Equal(t, "Internal Server Error", st.Message())
	})

	t.Run("uses_title_when_detail_is_empty", func(t *testing.T) {
		st := ToGRPCStatus(&core.ProblemDetails{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
		})
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Equal(t, "Bad Request", st.Message())
	})

	t.Run("uses_detail_when_present", func(t *testing.T) {
		st := ToGRPCStatus(&core.ProblemDetails{
			Status: http.StatusForbidden,
			Title:  "Forbidden",
			Detail: "token invalid",
		})
		assert.Equal(t, codes.PermissionDenied, st.Code())
		assert.Equal(t, "token invalid", st.Message())
	})
}

func TestWriteHTTP_EncodesJSONShape(t *testing.T) {
	rec := newTestRecorder()
	pd := &core.ProblemDetails{
		Type:     "urn:test:type",
		Title:    "Bad Request",
		Status:   http.StatusBadRequest,
		Detail:   "detail text",
		Instance: "/instance",
		Code:     "invalid_argument",
	}

	WriteHTTP(rec, pd)

	var body map[string]any
	assert.NoError(t, json.Unmarshal(rec.body, &body))
	assert.Equal(t, "urn:test:type", body["type"])
	assert.EqualValues(t, http.StatusBadRequest, body["status"])
}

type testRecorder struct {
	header http.Header
	status int
	body   []byte
}

func newTestRecorder() *testRecorder {
	return &testRecorder{header: make(http.Header)}
}

func (r *testRecorder) Header() http.Header  { return r.header }
func (r *testRecorder) WriteHeader(code int) { r.status = code }
func (r *testRecorder) Write(p []byte) (int, error) {
	r.body = append(r.body, p...)
	return len(p), nil
}
