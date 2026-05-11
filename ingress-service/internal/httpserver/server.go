package httpserver

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"yunxiao-ingress-service/internal/config"
	"yunxiao-ingress-service/internal/model"
	"yunxiao-ingress-service/internal/mq"
	"yunxiao-ingress-service/internal/repository"
	"yunxiao-ingress-service/internal/yunxiao/codeup"
	"yunxiao-ingress-service/internal/yunxiao/flow"
	"yunxiao-ingress-service/internal/yunxiao/projex"
)

// Server 封装接入服务的 HTTP 路由和外部依赖。
type Server struct {
	cfg       config.Config
	repo      *repository.EventRepository
	publisher mq.Publisher
	router    *gin.Engine
}

// New 创建接入服务实例并注册 HTTP 路由。
func New(cfg config.Config, repo *repository.EventRepository, publisher mq.Publisher) *Server {
	server := &Server{cfg: cfg, repo: repo, publisher: publisher}
	router := gin.Default()
	router.GET("/healthz", server.healthz)
	router.POST("/webhooks/yunxiao/codeup", server.handleCodeupWebhook)
	router.POST("/webhooks/yunxiao/projex", server.handleProjexWebhook)
	router.POST("/webhooks/yunxiao/flow", server.handleFlowWebhook)
	server.router = router
	return server
}

// Run 启动 HTTP 服务。
func (s *Server) Run() error {
	return s.router.Run(s.cfg.HTTPAddr)
}

func (s *Server) healthz(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleCodeupWebhook 接收 Codeup 合并请求 webhook。
func (s *Server) handleCodeupWebhook(ctx *gin.Context) {
	if !codeup.VerifyToken(ctx.Request, s.cfg.CodeupToken) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid codeup token"})
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "read body"})
		return
	}

	payload, err := codeup.PayloadFromJSON(body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid codeup json"})
		return
	}

	traceID := "trace_" + uuid.NewString()
	raw := model.RawEventMeta{
		Source:          "yunxiao.codeup",
		EventType:       ctx.Request.Header.Get("Codeup-Event"),
		ExternalEventID: ctx.Request.Header.Get("X-Codeup-Delivery"),
	}
	rawID, rawPayloadID, err := s.repo.SaveRawEvent(ctx.Request.Context(), raw, ctx.Request.Header, body, true, traceID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "save raw event"})
		return
	}

	event, err := codeup.Normalize(payload, rawPayloadID, body, traceID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.persistAndPublish(ctx, event, rawID)
}

// handleProjexWebhook 接收 Projex 自动化规则 webhook。
func (s *Server) handleProjexWebhook(ctx *gin.Context) {
	if !projex.VerifySignature(ctx.Request, s.cfg.ProjexSecret) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid projex signature"})
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "read body"})
		return
	}

	payload, err := projex.PayloadFromJSON(body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid projex json"})
		return
	}

	eventType := ctx.Query("event_type")
	traceID := "trace_" + uuid.NewString()
	raw := model.RawEventMeta{
		Source:          "yunxiao.projex",
		EventType:       eventType,
		ExternalEventID: payload.ID,
	}
	rawID, rawPayloadID, err := s.repo.SaveRawEvent(ctx.Request.Context(), raw, ctx.Request.Header, body, true, traceID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "save raw event"})
		return
	}

	event, err := projex.Normalize(payload, eventType, rawPayloadID, body, traceID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.persistAndPublish(ctx, event, rawID)
}

// handleFlowWebhook 接收 Flow 流水线通知 webhook。
func (s *Server) handleFlowWebhook(ctx *gin.Context) {
	if !flow.VerifyToken(ctx.Request, s.cfg.FlowToken) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid flow token"})
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "read body"})
		return
	}

	payload, err := flow.PayloadFromJSON(body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid flow json"})
		return
	}

	traceID := "trace_" + uuid.NewString()
	raw := model.RawEventMeta{
		Source:          "yunxiao.flow",
		EventType:       payload.Event + "." + payload.Action,
		ExternalEventID: payload.Task.PipelineID + ":" + payload.Task.BuildNumber + ":" + payload.Task.StatusCode,
	}
	rawID, rawPayloadID, err := s.repo.SaveRawEvent(ctx.Request.Context(), raw, ctx.Request.Header, body, true, traceID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "save raw event"})
		return
	}

	event, err := flow.Normalize(payload, rawPayloadID, body, traceID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.persistAndPublish(ctx, event, rawID)
}

// persistAndPublish 保存标准事件并发布到 RabbitMQ。
func (s *Server) persistAndPublish(ctx *gin.Context, event model.NormalizedEvent, rawID int64) {
	if err := s.repo.SaveNormalizedEvent(ctx.Request.Context(), event, rawID); err != nil {
		if errors.Is(err, repository.ErrDuplicateEvent) {
			ctx.JSON(http.StatusOK, gin.H{"event_id": event.EventID, "status": "duplicate"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "save normalized event"})
		return
	}

	if err := s.publisher.Publish(ctx.Request.Context(), event); err != nil {
		_ = s.repo.MarkPublishFailed(ctx.Request.Context(), event.EventID)
		ctx.JSON(http.StatusAccepted, gin.H{"event_id": event.EventID, "status": "stored_publish_failed"})
		return
	}

	if err := s.repo.MarkPublished(ctx.Request.Context(), event.EventID); err != nil {
		ctx.JSON(http.StatusAccepted, gin.H{"event_id": event.EventID, "status": "published_mark_failed"})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"event_id": event.EventID, "status": "published"})
}
