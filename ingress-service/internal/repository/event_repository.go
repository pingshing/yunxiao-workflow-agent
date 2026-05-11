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
func (r *EventRepository) SaveRawEvent(ctx context.Context, meta model.RawEventMeta, headers map[string][]string, payload []byte, signatureValid bool, traceID string) (int64, string, error) {
	headersJSON, err := json.Marshal(headers)
	if err != nil {
		return 0, "", err
	}

	now := time.Now().UTC()
	var id int64
	err = r.db.QueryRowContext(
		ctx,
		`INSERT INTO raw_event
			(source, event_type, external_event_id, signature_valid, headers_json, payload_json, received_at, trace_id, created_at)
		 VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8, $9)
		 RETURNING id`,
		meta.Source,
		meta.EventType,
		meta.ExternalEventID,
		signatureValid,
		string(headersJSON),
		string(payload),
		now,
		traceID,
		now,
	).Scan(&id)
	if err != nil {
		return 0, "", err
	}

	return id, rawPayloadID(id), nil
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

func rawPayloadID(id int64) string {
	return "raw_" + strconv.FormatInt(id, 10)
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
