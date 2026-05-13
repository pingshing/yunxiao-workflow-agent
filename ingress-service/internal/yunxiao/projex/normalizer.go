package projex

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"yunxiao-ingress-service/internal/model"
)

// ErrIgnoredEvent 表示该 Projex Webhook 合法接收并已保存 raw_event，但当前不进入后续编排。
var ErrIgnoredEvent = errors.New("ignored projex event")

// Normalize 将 Projex 工作项 Webhook payload 转换为内部标准事件。
func Normalize(payload WebhookPayload, eventType string, rawPayloadID string, rawBody []byte, traceID string) (model.NormalizedEvent, error) {
	if traceID == "" {
		traceID = "trace_" + uuid.NewString()
	}
	normalizedEventType, err := normalizeEventType(eventType)
	if err != nil {
		return model.NormalizedEvent{}, err
	}

	workItemID := firstNonEmpty(payload.Identifier, payload.ID)
	if workItemID == "" {
		return model.NormalizedEvent{}, fmt.Errorf("projex work item id is required")
	}

	occurredAt := parseOptionalTime(firstNonEmpty(payload.GmtModified, payload.GmtCreate))
	projectID := firstNonEmpty(payload.Space.Identifier, payload.Space.ID)
	externalRefs := map[string]any{
		"category_id":       payload.CategoryID,
		"id_path":           payload.IDPath,
		"serial_number":     payload.SerialNumber,
		"logical_status":    payload.LogicalStatus,
		"projex_space_id":   payload.Space.ID,
		"projex_space_name": firstNonEmpty(payload.Space.DisplayName, payload.Space.Name),
		"work_item_title":   payload.Subject,
		"work_item_status":  firstNonEmpty(payload.Status.DisplayName, payload.Status.Name),
		"status_stage_id":   payload.Status.StatusStageID,
		"work_item_type":    firstNonEmpty(payload.WorkitemType.DisplayName, payload.WorkitemType.Name),
		"work_item_type_id": payload.WorkitemType.ID,
		"assignee":          firstNonEmpty(payload.AssignedTo.DisplayName, payload.AssignedTo.Name),
		"assignee_id":       payload.AssignedTo.ID,
		"creator":           firstNonEmpty(payload.Creator.DisplayName, payload.Creator.Name),
		"creator_id":        payload.Creator.ID,
		"modifier":          firstNonEmpty(payload.Modifier.DisplayName, payload.Modifier.Name),
		"modifier_id":       payload.Modifier.ID,
		"verifier":          firstNonEmpty(payload.Verifier.DisplayName, payload.Verifier.Name),
		"verifier_id":       payload.Verifier.ID,
		"sprint":            firstNonEmpty(payload.Sprint.DisplayName, payload.Sprint.Name),
		"sprint_id":         payload.Sprint.ID,
		"parent_id":         firstNonEmpty(payload.Parent.Identifier, payload.Parent.ID, payload.ParentID),
		"participants":      namedValues(payload.Participants),
		"trackers":          namedValues(payload.Trackers),
		"versions":          namedValues(payload.Versions),
		"labels":            namedValues(payload.Labels),
		"custom_fields":     customFields(payload.CustomFieldValues),
	}

	return model.NormalizedEvent{
		EventID:      "evt_" + uuid.NewString(),
		Source:       "yunxiao.projex",
		EventType:    normalizedEventType,
		OccurredAt:   occurredAt,
		TraceID:      traceID,
		DedupKey:     buildDedupKey(normalizedEventType, workItemID, payload, rawBody),
		ProjectID:    projectID,
		WorkItemID:   workItemID,
		Subject:      model.Subject{Type: "work_item", ID: workItemID},
		ExternalRefs: externalRefs,
		RawPayloadID: rawPayloadID,
	}, nil
}

func normalizeEventType(eventType string) (string, error) {
	switch strings.TrimSpace(eventType) {
	case "":
		return "work_item.updated", nil
	case "work_item.created", "work_item.updated", "work_item.status_changed", "work_item.assignee_changed", "work_item.deleted":
		return eventType, nil
	default:
		return "", fmt.Errorf("%w: unsupported event_type %q", ErrIgnoredEvent, eventType)
	}
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

func namedValues(values []NamedValue) []map[string]string {
	result := make([]map[string]string, 0, len(values))
	for _, value := range values {
		result = append(result, map[string]string{
			"id":              value.ID,
			"identifier":      value.Identifier,
			"name":            value.Name,
			"name_en":         value.NameEn,
			"display_name":    value.DisplayName,
			"color":           value.Color,
			"status_stage_id": value.StatusStageID,
		})
	}
	return result
}

func customFields(values []CustomFieldValue) map[string]any {
	result := make(map[string]any, len(values))
	for _, value := range values {
		key := firstNonEmpty(value.FieldIdentifier, value.FieldID, value.FieldName)
		if key != "" {
			result[key] = customFieldValue(value)
		}
	}
	return result
}

func customFieldValue(value CustomFieldValue) any {
	if value.Value != nil {
		return value.Value
	}
	entries := make([]map[string]string, 0, len(value.Values))
	for _, item := range value.Values {
		entries = append(entries, map[string]string{
			"display_value": item.DisplayValue,
			"identifier":    item.Identifier,
		})
	}
	return entries
}

// PayloadFromJSON 将请求体解析为 Projex Webhook payload。
func PayloadFromJSON(body []byte) (WebhookPayload, error) {
	var payload WebhookPayload
	err := json.Unmarshal(body, &payload)
	return payload, err
}
