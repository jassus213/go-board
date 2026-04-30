package rest

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jassus213/go-board/dashboard/auth"
	"github.com/jassus213/go-board/dashboard/core"
	"github.com/jassus213/go-board/dashboard/core/dto"
	"github.com/jassus213/go-board/dashboard/core/interfaces"
	"github.com/jassus213/go-board/dashboard/delivery/problem"
	"github.com/jassus213/go-board/dashboard/delivery/ws"
)

type Params struct {
	Hub      *ws.Hub
	UseCase  interfaces.DashboardUseCase
	Verifier auth.Verifier
	Config   Config
}

type handlers struct {
	params     *Params
	corsConfig ws.CORSConfig
}

func NewRouter(p *Params) *gin.Engine {
	if p == nil {
		return gin.New()
	}

	cfg := p.Config.normalized()
	h := &handlers{
		params: p,
		corsConfig: ws.CORSConfig{
			AllowedOrigins:   cfg.CORSAllowedOrigins,
			AllowCredentials: cfg.CORSAllowCredentials,
		},
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	if cfg.EnableWebSocket {
		router.GET("/ws", h.websocket)
	} else {
		router.GET("/ws", h.websocketDisabled)
	}

	router.GET("/health", h.health)

	v1 := router.Group(cfg.APIPrefix)
	v1.Use(authMiddleware(p.Verifier))
	v1.POST("/dashboards/:dashboard/members/:member_id/increment", h.incrementScore)
	v1.GET("/dashboards/:dashboard/members/:member_id/rank", h.getMemberRank)
	v1.GET("/dashboards/:dashboard/top", h.getTopMembers)
	v1.GET("/dashboards/:dashboard/stats", h.getDashboardStats)

	return router
}

func (h *handlers) websocket(c *gin.Context) {
	ws.ServeWs(h.params.Hub, h.params.UseCase, h.params.Verifier, h.corsConfig, c.Writer, c.Request)
}

func (h *handlers) websocketDisabled(c *gin.Context) {
	pd := problem.FromError(errors.New("websocket mode is disabled"), http.StatusServiceUnavailable, c.Request.URL.Path)
	problem.WriteHTTP(c.Writer, &pd)
}

func (h *handlers) health(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}

func (h *handlers) incrementScore(c *gin.Context) {
	var body struct {
		Increment *float64 `json:"increment"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Increment == nil {
		writeProblem(
			c,
			fmt.Errorf("%w: increment is required", core.ErrInvalidArgument),
			http.StatusBadRequest,
		)
		return
	}

	authMemberID := c.GetString("member_id")
	requestedMemberID := c.Param("member_id")
	if requestedMemberID != authMemberID {
		log.Printf("Security warning: user %s tried to update score for %s", authMemberID, requestedMemberID)
	}

	rank, err := h.params.UseCase.ProcessScoreUpdate(c.Request.Context(), dto.IncrementScoreRequest{
		Dashboard: c.Param("dashboard"),
		MemberID:  authMemberID,
		Value:     *body.Increment,
	})
	if err != nil {
		writeProblem(c, err, http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{"member_id": authMemberID, "rank": rank})
}

func (h *handlers) getMemberRank(c *gin.Context) {
	authMemberID := c.GetString("member_id")
	requestedMemberID := c.Param("member_id")
	if requestedMemberID != authMemberID {
		log.Printf("Security warning: user %s tried to fetch rank for %s", authMemberID, requestedMemberID)
	}

	rank, err := h.params.UseCase.GetMemberRankHandler(c.Request.Context(), dto.GetRankRequest{
		Dashboard: c.Param("dashboard"),
		MemberID:  authMemberID,
	})
	if err != nil {
		writeProblem(c, err, http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{"member_id": authMemberID, "rank": rank})
}

func (h *handlers) getTopMembers(c *gin.Context) {
	limit := int64(10)
	if limitValue := c.Query("limit"); limitValue != "" {
		parsedLimit, err := strconv.ParseInt(limitValue, 10, 64)
		if err != nil {
			writeProblem(
				c,
				fmt.Errorf("%w: invalid limit query parameter", core.ErrInvalidArgument),
				http.StatusBadRequest,
			)
			return
		}
		limit = parsedLimit
	}

	top, err := h.params.UseCase.GetTopMembersHandler(c.Request.Context(), dto.GetTopRequest{
		Dashboard: c.Param("dashboard"),
		Limit:     limit,
	})
	if err != nil {
		writeProblem(c, err, http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": top})
}

func (h *handlers) getDashboardStats(c *gin.Context) {
	total, err := h.params.UseCase.GetDashboardStatsHandler(c.Request.Context(), c.Param("dashboard"))
	if err != nil {
		writeProblem(c, err, http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{"total_members": total})
}

func authMiddleware(verifier auth.Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")
		if token == "" {
			writeProblem(c, auth.ErrEmptyToken, http.StatusUnauthorized)
			return
		}

		memberID, err := verifier.Verify(c.Request.Context(), token)
		if err != nil {
			writeProblem(c, err, http.StatusForbidden)
			return
		}

		c.Set("member_id", memberID)
		c.Next()
	}
}

func writeProblem(c *gin.Context, err error, fallbackStatus int) {
	pd := problem.FromError(err, fallbackStatus, c.Request.URL.Path)
	problem.WriteHTTP(c.Writer, &pd)
	c.Abort()
}
