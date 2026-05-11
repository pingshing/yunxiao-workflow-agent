package codeup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"yunxiao-ingress-service/internal/model"
)

// Normalize 将 Codeup Webhook payload 转换为内部标准事件。
func Normalize(payload WebhookPayload, rawPayloadID string, rawBody []byte, traceID string) (model.NormalizedEvent, error) {
	if traceID == "" {
		traceID = "trace_" + uuid.NewString()
	}

	eventType, err := mapEventType(payload)
	if err != nil {
		return model.NormalizedEvent{}, err
	}

	mr := payload.ObjectAttributes
	subjectID := mergeRequestID(mr)
	if subjectID == "" {
		return model.NormalizedEvent{}, fmt.Errorf("codeup merge request id is required")
	}

	occurredAt := parseOptionalTime(firstNonEmpty(mr.UpdatedAt, mr.CreatedAt))
	externalRefs := map[string]any{
		"codeup_project_id":    strconv.FormatInt(payload.ProjectID, 10),
		"source_project_id":    idString(mr.SourceProjectID),
		"target_project_id":    idString(mr.TargetProjectID),
		"repo_name":            payload.Repository.Name,
		"repository_url":       firstNonEmpty(payload.Repository.GitHTTPURL, payload.Repository.URL, payload.Repository.Homepage),
		"source_branch":        mr.SourceBranch,
		"target_branch":        mr.TargetBranch,
		"merge_request_title":  mr.Title,
		"merge_request_state":  mr.State,
		"merge_request_action": mr.Action,
		"user_name":            firstNonEmpty(payload.User.Username, payload.User.Name),
	}
	if mr.LastCommit != nil {
		externalRefs["commit_sha"] = mr.LastCommit.ID
	}

	return model.NormalizedEvent{
		EventID:      "evt_" + uuid.NewString(),
		Source:       "yunxiao.codeup",
		EventType:    eventType,
		OccurredAt:   occurredAt,
		TraceID:      traceID,
		DedupKey:     buildDedupKey(payload, rawBody, eventType, subjectID),
		Subject:      model.Subject{Type: "pull_request", ID: subjectID},
		ExternalRefs: externalRefs,
		RawPayloadID: rawPayloadID,
	}, nil
}

func mapEventType(payload WebhookPayload) (string, error) {
	if payload.ObjectKind != "merge_request" {
		return "", fmt.Errorf("unsupported codeup object_kind: %s", payload.ObjectKind)
	}

	switch strings.ToLower(payload.ObjectAttributes.Action) {
	case "open", "create", "reopen":
		return "pr.created", nil
	case "merge", "merged":
		return "pr.merged", nil
	case "close", "closed":
		return "pr.closed", nil
	default:
		return "pr.updated", nil
	}
}

func buildDedupKey(payload WebhookPayload, rawBody []byte, eventType string, subjectID string) string {
	if payload.ObjectAttributes.UpdatedAt != "" {
		return fmt.Sprintf("yunxiao.codeup:%s:%d:%s:%s", eventType, payload.ProjectID, subjectID, payload.ObjectAttributes.UpdatedAt)
	}

	payloadHash := sha256.Sum256(rawBody)
	return fmt.Sprintf("yunxiao.codeup:%s:%d:%s:%s", eventType, payload.ProjectID, subjectID, hex.EncodeToString(payloadHash[:])[:16])
}

func mergeRequestID(mr MergeRequestAttributes) string {
	if mr.BizID != "" {
		return mr.BizID
	}
	if mr.LocalID > 0 {
		return strconv.FormatInt(mr.LocalID, 10)
	}
	if mr.ID > 0 {
		return strconv.FormatInt(mr.ID, 10)
	}
	return ""
}

func parseOptionalTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05 MST", "2006-01-02 15:04:05"} {
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

func idString(value int64) string {
	if value == 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}

// PayloadFromJSON 将请求体解析为 Codeup Webhook payload。
func PayloadFromJSON(body []byte) (WebhookPayload, error) {
	var payload WebhookPayload
	err := json.Unmarshal(body, &payload)
	return payload, err
}
