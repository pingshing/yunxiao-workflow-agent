package projex

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"yunxiao-ingress-service/internal/model"
)

// Normalize 将 Projex 工作项 Webhook payload 转换为内部标准事件。
func Normalize(payload WebhookPayload, eventType string, rawPayloadID string, rawBody []byte, traceID string) (model.NormalizedEvent, error) {
	if traceID == "" {
		traceID = "trace_" + uuid.NewString()
	}
	if eventType == "" {
		eventType = "work_item.updated"
	}

	workItemID := firstNonEmpty(payload.Identifier, payload.ID)
	if workItemID == "" {
		return model.NormalizedEvent{}, fmt.Errorf("projex work item id is required")
	}

	occurredAt := parseOptionalTime(firstNonEmpty(payload.GmtModified, payload.GmtCreate))
	projectID := firstNonEmpty(payload.Space.Identifier, payload.Space.ID)
	externalRefs := map[string]any{
		"projex_space_id":   payload.Space.ID,
		"projex_space_name": firstNonEmpty(payload.Space.DisplayName, payload.Space.Name),
		"work_item_title":   payload.Subject,
		"work_item_status":  firstNonEmpty(payload.Status.DisplayName, payload.Status.Name),
		"work_item_type":    firstNonEmpty(payload.WorkitemType.DisplayName, payload.WorkitemType.Name),
		"assignee":          firstNonEmpty(payload.AssignedTo.DisplayName, payload.AssignedTo.Name),
		"modifier":          firstNonEmpty(payload.Modifier.DisplayName, payload.Modifier.Name),
	}

	return model.NormalizedEvent{
		EventID:      "evt_" + uuid.NewString(),
		Source:       "yunxiao.projex",
		EventType:    eventType,
		OccurredAt:   occurredAt,
		TraceID:      traceID,
		DedupKey:     buildDedupKey(eventType, workItemID, payload, rawBody),
		ProjectID:    projectID,
		WorkItemID:   workItemID,
		Subject:      model.Subject{Type: "work_item", ID: workItemID},
		ExternalRefs: externalRefs,
		RawPayloadID: rawPayloadID,
	}, nil
}

func buildDedupKey(eventType string, workItemID string, payload WebhookPayload, rawBody []byte) string {
	if payload.GmtModified != "" {
		return fmt.Sprintf("yunxiao.projex:%s:%s:%s", eventType, workItemID, payload.GmtModified)
	}
	payloadHash := sha256.Sum256(rawBody)
	return fmt.Sprintf("yunxiao.projex:%s:%s:%s", eventType, workItemID, hex.EncodeToString(payloadHash[:])[:16])
}

func parseOptionalTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05.000Z"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// PayloadFromJSON 将请求体解析为 Projex Webhook payload。
func PayloadFromJSON(body []byte) (WebhookPayload, error) {
	var payload WebhookPayload
	err := json.Unmarshal(body, &payload)
	return payload, err
}
