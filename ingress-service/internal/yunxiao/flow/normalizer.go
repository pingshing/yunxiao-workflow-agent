package flow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"yunxiao-ingress-service/internal/model"
)

// ErrIgnoredEvent 表示该 Flow 通知合法接收并已保存 raw_event，但当前不进入后续编排。
var ErrIgnoredEvent = errors.New("ignored flow event")

// Normalize 将 Flow Webhook 通知 payload 转换为内部标准事件。
func Normalize(payload WebhookPayload, rawPayloadID string, rawBody []byte, traceID string) (model.NormalizedEvent, error) {
	if traceID == "" {
		traceID = "trace_" + uuid.NewString()
	}

	eventType, err := mapEventType(payload.Task.StatusCode)
	if err != nil {
		return model.NormalizedEvent{}, err
	}

	runID := firstNonEmpty(payload.Task.BuildNumber, payload.Task.PipelineURL)
	if runID == "" {
		return model.NormalizedEvent{}, fmt.Errorf("flow pipeline run id is required")
	}

	source := firstSource(payload.Sources)
	externalRefs := map[string]any{
		"pipeline_id":     payload.Task.PipelineID,
		"pipeline_name":   payload.Task.PipelineName,
		"stage_name":      payload.Task.StageName,
		"task_name":       payload.Task.TaskName,
		"status_code":     payload.Task.StatusCode,
		"status_name":     payload.Task.StatusName,
		"pipeline_url":    payload.Task.PipelineURL,
		"message":         payload.Task.Message,
		"repo":            source.Repo,
		"branch":          source.Branch,
		"commit_sha":      source.CommitID,
		"previous_commit": source.PreviousCommitID,
		"global_params":   globalParamsMap(payload.GlobalParams),
	}

	return model.NormalizedEvent{
		EventID:      "evt_" + uuid.NewString(),
		Source:       "yunxiao.flow",
		EventType:    eventType,
		TraceID:      traceID,
		DedupKey:     buildDedupKey(eventType, payload, rawBody, runID),
		Subject:      model.Subject{Type: "pipeline_run", ID: runID},
		ExternalRefs: externalRefs,
		RawPayloadID: rawPayloadID,
	}, nil
}

func mapEventType(statusCode string) (string, error) {
	switch strings.ToUpper(statusCode) {
	case "FAIL":
		return "pipeline.failed", nil
	case "SUCCESS":
		return "pipeline.succeeded", nil
	case "FINISH":
		return "pipeline.finished", nil
	case "CANCELED", "CANCELLING":
		return "pipeline.canceled", nil
	case "RUNNING", "WAITING":
		return "pipeline.started", nil
	case "SKIP":
		return "pipeline.skipped", nil
	case "UNKOWN", "UNKNOWN":
		return "", fmt.Errorf("%w: statusCode %q", ErrIgnoredEvent, statusCode)
	default:
		return "", fmt.Errorf("%w: unsupported statusCode %q", ErrIgnoredEvent, statusCode)
	}
}

func buildDedupKey(eventType string, payload WebhookPayload, rawBody []byte, runID string) string {
	if payload.Task.PipelineID != "" && runID != "" && payload.Task.StatusCode != "" {
		return fmt.Sprintf("yunxiao.flow:%s:%s:%s:%s", eventType, payload.Task.PipelineID, runID, payload.Task.StatusCode)
	}
	payloadHash := sha256.Sum256(rawBody)
	return fmt.Sprintf("yunxiao.flow:%s:%s", eventType, hex.EncodeToString(payloadHash[:])[:16])
}

func firstSource(sources []Source) Source {
	if len(sources) == 0 {
		return Source{}
	}
	return sources[0]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func globalParamsMap(params []GlobalParam) map[string]string {
	values := make(map[string]string, len(params))
	for _, param := range params {
		if param.Key != "" {
			values[param.Key] = param.Value
		}
	}
	return values
}

// PayloadFromJSON 将请求体解析为 Flow Webhook payload。
func PayloadFromJSON(body []byte) (WebhookPayload, error) {
	var payload WebhookPayload
	err := json.Unmarshal(body, &payload)
	return payload, err
}
