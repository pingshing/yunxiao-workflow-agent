package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/lib/pq"

	"yunxiao-ingress-service/internal/model"
)

const (
	// RawEventStatusReceived 表示 webhook 请求已通过鉴权并写入 raw_event。
	RawEventStatusReceived = "received"
	// RawEventStatusInvalidPayload 表示请求体不是当前 adapter 可解析的 JSON。
	RawEventStatusInvalidPayload = "invalid_payload"
	// RawEventStatusIgnored 表示请求合法，但当前业务版本不进入标准事件编排。
	RawEventStatusIgnored = "ignored"
	// RawEventStatusNormalized 表示请求已经成功生成 normalized_event。
	RawEventStatusNormalized = "normalized"
	// RawEventStatusFailed 表示接入层处理过程中发生系统异常。
	RawEventStatusFailed = "failed"
)

// ErrDuplicateEvent 表示标准事件已经通过 dedup_key 入库。
var ErrDuplicateEvent = errors.New("duplicate normalized event")

// EventRepository 负责接入层事件相关的 PostgreSQL 持久化。
type EventRepository struct {
	db *sql.DB
}

// NewEventRepository 创建事件仓储。
func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

// SaveRawEvent 保存原始 webhook payload，并返回数据库 ID 和 raw_payload_id。
// payload_text 始终保存原始请求体；payload_json 只在请求体是合法 JSON 时保存。
func (r *EventRepository) SaveRawEvent(ctx context.Context, meta model.RawEventMeta, headers map[string][]string, payload []byte, signatureValid bool, traceID string) (int64, string, error) {
	headersJSON, err := json.Marshal(headers)
	if err != nil {
		return 0, "", err
	}

	payloadText := string(payload)
	var payloadJSON any
	if json.Valid(payload) {
		payloadJSON = payloadText
	}

	now := time.Now().UTC()
	var id int64
	err = r.db.QueryRowContext(
		ctx,
		`INSERT INTO raw_event
			(source, event_type, external_event_id, signature_valid, headers_json, payload_json,
			 payload_text, process_status, received_at, trace_id, created_at)
		 VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8, $9, $10, $11)
		 RETURNING id`,
		meta.Source,
		meta.EventType,
		meta.ExternalEventID,
		signatureValid,
		string(headersJSON),
		payloadJSON,
		payloadText,
		RawEventStatusReceived,
		now,
		traceID,
		now,
	).Scan(&id)
	if err != nil {
		return 0, "", err
	}

	return id, rawPayloadID(id), nil
}

// MarkRawEventInvalidPayload 标记原始事件因 JSON 解析失败而终止处理。
func (r *EventRepository) MarkRawEventInvalidPayload(ctx context.Context, rawEventID int64, processError string) error {
	return r.updateRawEventProcessStatus(ctx, rawEventID, RawEventStatusInvalidPayload, processError, "")
}

// MarkRawEventIgnored 标记原始事件被业务规则忽略，不进入后续编排。
func (r *EventRepository) MarkRawEventIgnored(ctx context.Context, rawEventID int64, processError string) error {
	return r.updateRawEventProcessStatus(ctx, rawEventID, RawEventStatusIgnored, processError, "")
}

// MarkRawEventNormalized 标记原始事件已经成功关联标准事件。
func (r *EventRepository) MarkRawEventNormalized(ctx context.Context, rawEventID int64, normalizedEventID string) error {
	return r.updateRawEventProcessStatus(ctx, rawEventID, RawEventStatusNormalized, "", normalizedEventID)
}

// MarkRawEventFailed 标记原始事件处理失败，并记录失败原因。
func (r *EventRepository) MarkRawEventFailed(ctx context.Context, rawEventID int64, processError string) error {
	return r.updateRawEventProcessStatus(ctx, rawEventID, RawEventStatusFailed, processError, "")
}

// UpdateRawEventMeta 回填需要解析 payload 后才能确定的事件元数据。
func (r *EventRepository) UpdateRawEventMeta(ctx context.Context, rawEventID int64, meta model.RawEventMeta) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE raw_event
		 SET event_type = COALESCE(NULLIF($1, ''), event_type),
		     external_event_id = COALESCE(NULLIF($2, ''), external_event_id)
		 WHERE id = $3`,
		meta.EventType,
		meta.ExternalEventID,
		rawEventID,
	)
	return err
}

// FindNormalizedEventIDByDedupKey 根据 dedup_key 查询已经存在的标准事件 ID。
func (r *EventRepository) FindNormalizedEventIDByDedupKey(ctx context.Context, dedupKey string) (string, error) {
	var eventID string
	err := r.db.QueryRowContext(
		ctx,
		`SELECT event_id FROM normalized_event WHERE dedup_key = $1`,
		dedupKey,
	).Scan(&eventID)
	if err != nil {
		return "", err
	}
	return eventID, nil
}

// SaveNormalizedEvent 保存内部标准事件。重复 dedup_key 会返回 ErrDuplicateEvent。
func (r *EventRepository) SaveNormalizedEvent(ctx context.Context, event model.NormalizedEvent, rawEventID int64) error {
	externalRefsJSON, err := json.Marshal(event.ExternalRefs)
	if err != nil {
		return err
	}

	var occurredAt any
	if event.OccurredAt != nil {
		occurredAt = event.OccurredAt.UTC()
	}

	now := time.Now().UTC()
	_, err = r.db.ExecContext(
		ctx,
		`INSERT INTO normalized_event
			(event_id, source, event_type, occurred_at, trace_id, dedup_key, project_id, work_item_id,
			 subject_type, subject_id, external_refs_json, raw_event_id, publish_status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12, 'pending', $13)`,
		event.EventID,
		event.Source,
		event.EventType,
		occurredAt,
		event.TraceID,
		event.DedupKey,
		nullableString(event.ProjectID),
		nullableString(event.WorkItemID),
		event.Subject.Type,
		event.Subject.ID,
		string(externalRefsJSON),
		rawEventID,
		now,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrDuplicateEvent
		}
		return err
	}

	return nil
}

// MarkPublished 标记标准事件已经成功发布到 MQ。
func (r *EventRepository) MarkPublished(ctx context.Context, eventID string) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE normalized_event SET publish_status = 'published', published_at = $1 WHERE event_id = $2`,
		time.Now().UTC(),
		eventID,
	)
	return err
}

// MarkPublishFailed 标记标准事件发布 MQ 失败，后续可由补偿任务重试。
func (r *EventRepository) MarkPublishFailed(ctx context.Context, eventID string) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE normalized_event SET publish_status = 'failed' WHERE event_id = $1`,
		eventID,
	)
	return err
}

func (r *EventRepository) updateRawEventProcessStatus(ctx context.Context, rawEventID int64, status string, processError string, normalizedEventID string) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE raw_event
		 SET process_status = $1,
		     process_error = $2,
		     normalized_event_id = COALESCE(NULLIF($3, ''), normalized_event_id),
		     processed_at = $4
		 WHERE id = $5`,
		status,
		nullableString(processError),
		normalizedEventID,
		time.Now().UTC(),
		rawEventID,
	)
	return err
}

func rawPayloadID(id int64) string {
	return "raw_" + strconv.FormatInt(id, 10)
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
