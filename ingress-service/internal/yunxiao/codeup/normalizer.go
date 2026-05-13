package codeup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"yunxiao-ingress-service/internal/model"
)

// ErrIgnoredEvent 表示该 Codeup 事件合法接收并已保存 raw_event，但当前不进入后续编排。
var ErrIgnoredEvent = errors.New("ignored codeup event")

// Normalize 将 Codeup Webhook payload 转换为内部标准事件。
func Normalize(payload WebhookPayload, rawPayloadID string, rawBody []byte, traceID string) (model.NormalizedEvent, error) {
	if traceID == "" {
		traceID = "trace_" + uuid.NewString()
	}

	switch normalizeObjectKind(payload.ObjectKind) {
	case "merge_request":
		return normalizeMergeRequest(payload, rawPayloadID, rawBody, traceID)
	case "push":
		return normalizePush(payload, rawPayloadID, rawBody, traceID, false)
	case "tag_push":
		return normalizePush(payload, rawPayloadID, rawBody, traceID, true)
	case "note":
		return normalizeNote(payload, rawPayloadID, rawBody, traceID)
	default:
		return model.NormalizedEvent{}, fmt.Errorf("%w: unsupported object_kind %q", ErrIgnoredEvent, payload.ObjectKind)
	}
}

func normalizeMergeRequest(payload WebhookPayload, rawPayloadID string, rawBody []byte, traceID string) (model.NormalizedEvent, error) {
	eventType := mapMergeRequestEventType(payload)

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
		"user_name":            payloadUserName(payload),
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

func normalizePush(payload WebhookPayload, rawPayloadID string, rawBody []byte, traceID string, tagPush bool) (model.NormalizedEvent, error) {
	refName := refName(payload.Ref)
	if refName == "" {
		return model.NormalizedEvent{}, fmt.Errorf("codeup ref is required")
	}

	eventType := "repo.pushed"
	subjectType := "branch"
	if tagPush {
		eventType = "repo.tag_pushed"
		subjectType = "tag"
	}
	commit := lastCommit(payload)
	externalRefs := map[string]any{
		"codeup_project_id": strconv.FormatInt(payload.ProjectID, 10),
		"repo_name":         payload.Repository.Name,
		"repository_url":    firstNonEmpty(payload.Repository.GitHTTPURL, payload.Repository.URL, payload.Repository.Homepage),
		"ref":               payload.Ref,
		"ref_name":          refName,
		"before":            payload.Before,
		"after":             payload.After,
		"checkout_sha":      payload.CheckoutSHA,
		"commit_sha":        firstNonEmpty(commit.ID, payload.CheckoutSHA, payload.After),
		"commit_message":    commit.Message,
		"commit_url":        commit.URL,
		"total_commits":     payload.TotalCommits,
		"user_name":         payloadUserName(payload),
	}

	return model.NormalizedEvent{
		EventID:      "evt_" + uuid.NewString(),
		Source:       "yunxiao.codeup",
		EventType:    eventType,
		OccurredAt:   parseOptionalTime(commit.Timestamp),
		TraceID:      traceID,
		DedupKey:     buildPushDedupKey(payload, rawBody, eventType, refName),
		Subject:      model.Subject{Type: subjectType, ID: refName},
		ExternalRefs: externalRefs,
		RawPayloadID: rawPayloadID,
	}, nil
}

func normalizeNote(payload WebhookPayload, rawPayloadID string, rawBody []byte, traceID string) (model.NormalizedEvent, error) {
	note := payload.ObjectAttributes
	noteID := firstNonEmpty(note.BizID, idString(note.ID))
	if noteID == "" {
		return model.NormalizedEvent{}, fmt.Errorf("codeup note id is required")
	}

	externalRefs := map[string]any{
		"codeup_project_id": strconv.FormatInt(payload.ProjectID, 10),
		"repo_name":         payload.Repository.Name,
		"repository_url":    firstNonEmpty(payload.Repository.GitHTTPURL, payload.Repository.URL, payload.Repository.Homepage),
		"note_title":        note.Title,
		"note_action":       note.Action,
		"user_name":         payloadUserName(payload),
	}
	if payload.MergeRequest != nil {
		externalRefs["merge_request_id"] = mergeRequestID(*payload.MergeRequest)
		externalRefs["merge_request_title"] = payload.MergeRequest.Title
	}
	if payload.Commit != nil {
		externalRefs["commit_sha"] = payload.Commit.ID
	}

	return model.NormalizedEvent{
		EventID:      "evt_" + uuid.NewString(),
		Source:       "yunxiao.codeup",
		EventType:    "repo.note_created",
		OccurredAt:   parseOptionalTime(firstNonEmpty(note.UpdatedAt, note.CreatedAt)),
		TraceID:      traceID,
		DedupKey:     buildNoteDedupKey(payload, rawBody, noteID),
		Subject:      model.Subject{Type: "note", ID: noteID},
		ExternalRefs: externalRefs,
		RawPayloadID: rawPayloadID,
	}, nil
}

func mapMergeRequestEventType(payload WebhookPayload) string {
	switch strings.ToLower(payload.ObjectAttributes.Action) {
	case "open", "create", "reopen":
		return "pr.created"
	case "merge", "merged":
		return "pr.merged"
	case "close", "closed":
		return "pr.closed"
	default:
		return "pr.updated"
	}
}

func buildDedupKey(payload WebhookPayload, rawBody []byte, eventType string, subjectID string) string {
	if payload.ObjectAttributes.UpdatedAt != "" {
		return fmt.Sprintf("yunxiao.codeup:%s:%d:%s:%s", eventType, payload.ProjectID, subjectID, payload.ObjectAttributes.UpdatedAt)
	}

	payloadHash := sha256.Sum256(rawBody)
	return fmt.Sprintf("yunxiao.codeup:%s:%d:%s:%s", eventType, payload.ProjectID, subjectID, hex.EncodeToString(payloadHash[:])[:16])
}

func buildPushDedupKey(payload WebhookPayload, rawBody []byte, eventType string, refName string) string {
	sha := firstNonEmpty(payload.CheckoutSHA, payload.After)
	if sha != "" {
		return fmt.Sprintf("yunxiao.codeup:%s:%d:%s:%s", eventType, payload.ProjectID, refName, sha)
	}

	payloadHash := sha256.Sum256(rawBody)
	return fmt.Sprintf("yunxiao.codeup:%s:%d:%s:%s", eventType, payload.ProjectID, refName, hex.EncodeToString(payloadHash[:])[:16])
}

func buildNoteDedupKey(payload WebhookPayload, rawBody []byte, noteID string) string {
	if payload.ObjectAttributes.UpdatedAt != "" {
		return fmt.Sprintf("yunxiao.codeup:repo.note_created:%d:%s:%s", payload.ProjectID, noteID, payload.ObjectAttributes.UpdatedAt)
	}

	payloadHash := sha256.Sum256(rawBody)
	return fmt.Sprintf("yunxiao.codeup:repo.note_created:%d:%s:%s", payload.ProjectID, noteID, hex.EncodeToString(payloadHash[:])[:16])
}

func mergeRequestID(mr MergeRequestAttributes) string {
	if mr.BizID != "" {
		return mr.BizID
	}
	if mr.IID > 0 {
		return strconv.FormatInt(mr.IID, 10)
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

func normalizeObjectKind(objectKind string) string {
	switch strings.ToLower(objectKind) {
	case "merge_request":
		return "merge_request"
	case "push":
		return "push"
	case "tag_push":
		return "tag_push"
	case "note":
		return "note"
	default:
		return strings.ToLower(objectKind)
	}
}

func refName(ref string) string {
	return strings.TrimPrefix(strings.TrimPrefix(ref, "refs/heads/"), "refs/tags/")
}

func lastCommit(payload WebhookPayload) Commit {
	if payload.Commit != nil {
		return *payload.Commit
	}
	if len(payload.Commits) > 0 {
		return payload.Commits[len(payload.Commits)-1]
	}
	return Commit{}
}

func payloadUserName(payload WebhookPayload) string {
	return firstNonEmpty(payload.User.Username, payload.User.Name, payload.UserUsername, payload.UserName)
}

// PayloadFromJSON 将请求体解析为 Codeup Webhook payload。
func PayloadFromJSON(body []byte) (WebhookPayload, error) {
	var payload WebhookPayload
	err := json.Unmarshal(body, &payload)
	return payload, err
}
