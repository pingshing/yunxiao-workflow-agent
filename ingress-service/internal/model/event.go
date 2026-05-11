package model

import "time"

// Subject 表示标准事件的主对象，例如 pull_request、pipeline_run 或 work_item。
type Subject struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// NormalizedEvent 是接入层输出给后续系统的内部标准事件。
// 它只承载路由、追踪和对象引用信息，不承载 PR diff、流水线日志等重上下文。
type NormalizedEvent struct {
	EventID      string         `json:"event_id"`
	Source       string         `json:"source"`
	EventType    string         `json:"event_type"`
	OccurredAt   *time.Time     `json:"occurred_at,omitempty"`
	TraceID      string         `json:"trace_id"`
	DedupKey     string         `json:"dedup_key"`
	ProjectID    string         `json:"project_id,omitempty"`
	WorkItemID   string         `json:"work_item_id,omitempty"`
	Subject      Subject        `json:"subject"`
	ExternalRefs map[string]any `json:"external_refs,omitempty"`
	RawPayloadID string         `json:"raw_payload_id"`
}

// RawEventMeta 是保存原始 webhook payload 时使用的轻量元数据。
type RawEventMeta struct {
	Source          string
	EventType       string
	ExternalEventID string
}
