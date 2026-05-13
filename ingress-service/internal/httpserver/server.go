package httpserver

import (
	"context"
	"errors"
	"io"
	"log"
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
	repo      eventRepository
	publisher mq.Publisher
	router    *gin.Engine
}

type eventRepository interface {
	SaveRawEvent(ctx context.Context, meta model.RawEventMeta, headers map[string][]string, payload []byte, signatureValid bool, traceID string) (int64, string, error)
	UpdateRawEventMeta(ctx context.Context, rawEventID int64, meta model.RawEventMeta) error
	MarkRawEventInvalidPayload(ctx context.Context, rawEventID int64, processError string) error
	MarkRawEventIgnored(ctx context.Context, rawEventID int64, processError string) error
	MarkRawEventNormalized(ctx context.Context, rawEventID int64, normalizedEventID string) error
	MarkRawEventFailed(ctx context.Context, rawEventID int64, processError string) error
	SaveNormalizedEvent(ctx context.Context, event model.NormalizedEvent, rawEventID int64) error
	FindNormalizedEventIDByDedupKey(ctx context.Context, dedupKey string) (string, error)
	MarkPublished(ctx context.Context, eventID string) error
	MarkPublishFailed(ctx context.Context, eventID string) error
}

// New 创建接入服务实例并注册 HTTP 路由。
func New(cfg config.Config, repo eventRepository, publisher mq.Publisher) *Server {
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
	traceID := "trace_" + uuid.NewString()
	if !codeup.VerifyToken(ctx.Request, s.cfg.CodeupToken) {
		logWebhook("unauthorized", traceID, 0, "yunxiao.codeup", ctx.Request.Header.Get("Codeup-Event"), ctx.Request.Header.Get("X-Codeup-Delivery"), http.StatusUnauthorized, errors.New("invalid codeup token"))
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid codeup token"})
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		logWebhook("read_body_failed", traceID, 0, "yunxiao.codeup", ctx.Request.Header.Get("Codeup-Event"), ctx.Request.Header.Get("X-Codeup-Delivery"), http.StatusBadRequest, err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "read body"})
		return
	}

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

	payload, err := codeup.PayloadFromJSON(body)
	if err != nil {
		_ = s.repo.MarkRawEventInvalidPayload(ctx.Request.Context(), rawID, "invalid codeup json: "+err.Error())
		logWebhook("invalid_payload", traceID, rawID, raw.Source, raw.EventType, raw.ExternalEventID, http.StatusBadRequest, err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid codeup json", "raw_payload_id": rawPayloadID, "trace_id": traceID})
		return
	}

	event, err := codeup.Normalize(payload, rawPayloadID, body, traceID)
	if err != nil {
		if errors.Is(err, codeup.ErrIgnoredEvent) {
			_ = s.repo.MarkRawEventIgnored(ctx.Request.Context(), rawID, err.Error())
			logWebhook("ignored", traceID, rawID, raw.Source, raw.EventType, raw.ExternalEventID, http.StatusAccepted, err)
			ctx.JSON(http.StatusAccepted, gin.H{"raw_payload_id": rawPayloadID, "status": "ignored", "trace_id": traceID})
			return
		}
		_ = s.repo.MarkRawEventFailed(ctx.Request.Context(), rawID, err.Error())
		logWebhook("failed", traceID, rawID, raw.Source, raw.EventType, raw.ExternalEventID, http.StatusBadRequest, err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.persistAndPublish(ctx, event, rawID)
}

// handleProjexWebhook 接收 Projex 自动化规则 webhook。
func (s *Server) handleProjexWebhook(ctx *gin.Context) {
	traceID := "trace_" + uuid.NewString()
	if !projex.VerifySignature(ctx.Request, s.cfg.ProjexSecret) {
		logWebhook("unauthorized", traceID, 0, "yunxiao.projex", ctx.Query("event_type"), "", http.StatusUnauthorized, errors.New("invalid projex signature"))
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid projex signature"})
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		logWebhook("read_body_failed", traceID, 0, "yunxiao.projex", ctx.Query("event_type"), "", http.StatusBadRequest, err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "read body"})
		return
	}

	eventType := ctx.Query("event_type")
	raw := model.RawEventMeta{
		Source:    "yunxiao.projex",
		EventType: eventType,
	}
	rawID, rawPayloadID, err := s.repo.SaveRawEvent(ctx.Request.Context(), raw, ctx.Request.Header, body, true, traceID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "save raw event"})
		return
	}

	payload, err := projex.PayloadFromJSON(body)
	if err != nil {
		_ = s.repo.MarkRawEventInvalidPayload(ctx.Request.Context(), rawID, "invalid projex json: "+err.Error())
		logWebhook("invalid_payload", traceID, rawID, raw.Source, raw.EventType, raw.ExternalEventID, http.StatusBadRequest, err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid projex json", "raw_payload_id": rawPayloadID, "trace_id": traceID})
		return
	}
	raw.ExternalEventID = payload.ID
	_ = s.repo.UpdateRawEventMeta(ctx.Request.Context(), rawID, raw)

	event, err := projex.Normalize(payload, eventType, rawPayloadID, body, traceID)
	if err != nil {
		if errors.Is(err, projex.ErrIgnoredEvent) {
			_ = s.repo.MarkRawEventIgnored(ctx.Request.Context(), rawID, err.Error())
			logWebhook("ignored", traceID, rawID, raw.Source, raw.EventType, raw.ExternalEventID, http.StatusAccepted, err)
			ctx.JSON(http.StatusAccepted, gin.H{"raw_payload_id": rawPayloadID, "status": "ignored", "trace_id": traceID})
			return
		}
		_ = s.repo.MarkRawEventFailed(ctx.Request.Context(), rawID, err.Error())
		logWebhook("failed", traceID, rawID, raw.Source, raw.EventType, raw.ExternalEventID, http.StatusBadRequest, err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.persistAndPublish(ctx, event, rawID)
}

// handleFlowWebhook 接收 Flow 流水线通知 webhook。
func (s *Server) handleFlowWebhook(ctx *gin.Context) {
	traceID := "trace_" + uuid.NewString()
	if !flow.VerifyToken(ctx.Request, s.cfg.FlowToken) {
		logWebhook("unauthorized", traceID, 0, "yunxiao.flow", ctx.Request.Header.Get("X-Flow-Event"), "", http.StatusUnauthorized, errors.New("invalid flow token"))
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid flow token"})
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		logWebhook("read_body_failed", traceID, 0, "yunxiao.flow", ctx.Request.Header.Get("X-Flow-Event"), "", http.StatusBadRequest, err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "read body"})
		return
	}

	raw := model.RawEventMeta{
		Source:    "yunxiao.flow",
		EventType: ctx.Request.Header.Get("X-Flow-Event"),
	}
	rawID, rawPayloadID, err := s.repo.SaveRawEvent(ctx.Request.Context(), raw, ctx.Request.Header, body, true, traceID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "save raw event"})
		return
	}

	payload, err := flow.PayloadFromJSON(body)
	if err != nil {
		_ = s.repo.MarkRawEventInvalidPayload(ctx.Request.Context(), rawID, "invalid flow json: "+err.Error())
		logWebhook("invalid_payload", traceID, rawID, raw.Source, raw.EventType, raw.ExternalEventID, http.StatusBadRequest, err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid flow json", "raw_payload_id": rawPayloadID, "trace_id": traceID})
		return
	}
	raw.EventType = payload.Event + "." + payload.Action
	raw.ExternalEventID = payload.Task.PipelineID + ":" + payload.Task.BuildNumber + ":" + payload.Task.StatusCode
	_ = s.repo.UpdateRawEventMeta(ctx.Request.Context(), rawID, raw)

	event, err := flow.Normalize(payload, rawPayloadID, body, traceID)
	if err != nil {
		if errors.Is(err, flow.ErrIgnoredEvent) {
			_ = s.repo.MarkRawEventIgnored(ctx.Request.Context(), rawID, err.Error())
			logWebhook("ignored", traceID, rawID, raw.Source, raw.EventType, raw.ExternalEventID, http.StatusAccepted, err)
			ctx.JSON(http.StatusAccepted, gin.H{"raw_payload_id": rawPayloadID, "status": "ignored", "trace_id": traceID})
			return
		}
		_ = s.repo.MarkRawEventFailed(ctx.Request.Context(), rawID, err.Error())
		logWebhook("failed", traceID, rawID, raw.Source, raw.EventType, raw.ExternalEventID, http.StatusBadRequest, err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.persistAndPublish(ctx, event, rawID)
}

// persistAndPublish 保存标准事件并发布到 RabbitMQ。
func (s *Server) persistAndPublish(ctx *gin.Context, event model.NormalizedEvent, rawID int64) {
	if err := s.repo.SaveNormalizedEvent(ctx.Request.Context(), event, rawID); err != nil {
		if errors.Is(err, repository.ErrDuplicateEvent) {
			eventID := event.EventID
			if existingEventID, findErr := s.repo.FindNormalizedEventIDByDedupKey(ctx.Request.Context(), event.DedupKey); findErr == nil {
				eventID = existingEventID
			}
			_ = s.repo.MarkRawEventNormalized(ctx.Request.Context(), rawID, eventID)
			logWebhook("normalized", event.TraceID, rawID, event.Source, event.EventType, "", http.StatusOK, nil)
			ctx.JSON(http.StatusOK, gin.H{"event_id": eventID, "status": "duplicate"})
			return
		}
		_ = s.repo.MarkRawEventFailed(ctx.Request.Context(), rawID, "save normalized event: "+err.Error())
		logWebhook("failed", event.TraceID, rawID, event.Source, event.EventType, "", http.StatusInternalServerError, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "save normalized event"})
		return
	}

	if err := s.repo.MarkRawEventNormalized(ctx.Request.Context(), rawID, event.EventID); err != nil {
		logWebhook("failed", event.TraceID, rawID, event.Source, event.EventType, "", http.StatusAccepted, err)
	}

	if err := s.publisher.Publish(ctx.Request.Context(), event); err != nil {
		_ = s.repo.MarkPublishFailed(ctx.Request.Context(), event.EventID)
		logWebhook("normalized", event.TraceID, rawID, event.Source, event.EventType, "", http.StatusAccepted, err)
		ctx.JSON(http.StatusAccepted, gin.H{"event_id": event.EventID, "status": "stored_publish_failed"})
		return
	}

	if err := s.repo.MarkPublished(ctx.Request.Context(), event.EventID); err != nil {
		logWebhook("normalized", event.TraceID, rawID, event.Source, event.EventType, "", http.StatusAccepted, err)
		ctx.JSON(http.StatusAccepted, gin.H{"event_id": event.EventID, "status": "published_mark_failed"})
		return
	}

	logWebhook("normalized", event.TraceID, rawID, event.Source, event.EventType, "", http.StatusAccepted, nil)
	ctx.JSON(http.StatusAccepted, gin.H{"event_id": event.EventID, "status": "published"})
}

func logWebhook(processStatus string, traceID string, rawEventID int64, source string, eventType string, externalEventID string, httpStatus int, err error) {
	processError := ""
	if err != nil {
		processError = err.Error()
	}
	log.Printf(
		"yunxiao webhook processed trace_id=%s raw_event_id=%d source=%s event_type=%s external_event_id=%s process_status=%s http_status=%d process_error=%q",
		traceID,
		rawEventID,
		source,
		eventType,
		externalEventID,
		processStatus,
		httpStatus,
		processError,
	)
}
