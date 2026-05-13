package flow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeTaskStatusCodes(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    string
		wantEventType string
	}{
		{name: "流水线失败", statusCode: "FAIL", wantEventType: "pipeline.failed"},
		{name: "流水线成功", statusCode: "SUCCESS", wantEventType: "pipeline.succeeded"},
		{name: "流水线完成", statusCode: "FINISH", wantEventType: "pipeline.finished"},
		{name: "流水线取消", statusCode: "CANCELED", wantEventType: "pipeline.canceled"},
		{name: "流水线取消中", statusCode: "CANCELLING", wantEventType: "pipeline.canceled"},
		{name: "流水线运行中", statusCode: "RUNNING", wantEventType: "pipeline.started"},
		{name: "流水线等待中", statusCode: "WAITING", wantEventType: "pipeline.started"},
		{name: "流水线跳过", statusCode: "SKIP", wantEventType: "pipeline.skipped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := flowTaskBody(t, tt.statusCode)

			payload, err := PayloadFromJSON(body)
			if err != nil {
				t.Fatalf("解析 Flow payload 失败: %v", err)
			}

			event, err := Normalize(payload, "raw_flow_1", body, "trace_flow_1")
			if err != nil {
				t.Fatalf("归一化 Flow payload 失败: %v", err)
			}

			if event.Source != "yunxiao.flow" {
				t.Fatalf("Source = %q, want %q", event.Source, "yunxiao.flow")
			}
			if event.EventType != tt.wantEventType {
				t.Fatalf("EventType = %q, want %q", event.EventType, tt.wantEventType)
			}
			if event.Subject.Type != "pipeline_run" {
				t.Fatalf("Subject.Type = %q, want %q", event.Subject.Type, "pipeline_run")
			}
			if event.Subject.ID != "202605110001" {
				t.Fatalf("Subject.ID = %q, want %q", event.Subject.ID, "202605110001")
			}
			if event.RawPayloadID != "raw_flow_1" {
				t.Fatalf("RawPayloadID = %q, want %q", event.RawPayloadID, "raw_flow_1")
			}
			if event.TraceID != "trace_flow_1" {
				t.Fatalf("TraceID = %q, want %q", event.TraceID, "trace_flow_1")
			}
			assertExternalRef(t, event.ExternalRefs, "pipeline_id", "pipe_123")
			assertExternalRef(t, event.ExternalRefs, "pipeline_name", "云效工作流 Agent 构建")
			assertExternalRef(t, event.ExternalRefs, "stage_name", "构建阶段")
			assertExternalRef(t, event.ExternalRefs, "task_name", "Go 单元测试")
			assertExternalRef(t, event.ExternalRefs, "status_code", tt.statusCode)
			assertExternalRef(t, event.ExternalRefs, "status_name", "任务状态")
			assertExternalRef(t, event.ExternalRefs, "pipeline_url", "https://flow.aliyun.com/pipelines/pipe_123/builds/202605110001")
			assertExternalRef(t, event.ExternalRefs, "repo", "yunxiao-workflow-agent")
			assertExternalRef(t, event.ExternalRefs, "branch", "main")
			assertExternalRef(t, event.ExternalRefs, "commit_sha", "4f9f2cab7a2c0d338da59b9a29e6f2a7e2f76111")
			assertExternalRef(t, event.ExternalRefs, "previous_commit", "3a8e2cab7a2c0d338da59b9a29e6f2a7e2f76000")
			globalParams, ok := event.ExternalRefs["global_params"].(map[string]string)
			if !ok {
				t.Fatalf("external_refs.global_params type = %T, want map[string]string", event.ExternalRefs["global_params"])
			}
			if globalParams["env"] != "test" {
				t.Fatalf("external_refs.global_params.env = %q, want %q", globalParams["env"], "test")
			}
			if event.DedupKey != "yunxiao.flow:"+tt.wantEventType+":pipe_123:202605110001:"+tt.statusCode {
				t.Fatalf("DedupKey = %q, want deterministic Flow dedup key", event.DedupKey)
			}
		})
	}
}

func TestNormalizeIgnoredTaskStatusCode(t *testing.T) {
	body := flowTaskBody(t, "PAUSED")

	payload, err := PayloadFromJSON(body)
	if err != nil {
		t.Fatalf("解析 Flow payload 失败: %v", err)
	}

	_, err = Normalize(payload, "raw_flow_unknown", body, "trace_flow_unknown")
	if err == nil {
		t.Fatal("Normalize() error = nil, want ignored statusCode error")
	}
	if !strings.Contains(err.Error(), "ignored flow event") {
		t.Fatalf("Normalize() error = %q, want ignored statusCode error", err.Error())
	}
}

func TestNormalizeOfficialUnknownTaskStatusCodeIsIgnored(t *testing.T) {
	body := flowTaskBody(t, "UNKOWN")

	payload, err := PayloadFromJSON(body)
	if err != nil {
		t.Fatalf("解析 Flow payload 失败: %v", err)
	}

	_, err = Normalize(payload, "raw_flow_unkown", body, "trace_flow_unkown")
	if err == nil {
		t.Fatal("Normalize() error = nil, want ignored UNKOWN statusCode error")
	}
	if !strings.Contains(err.Error(), "ignored flow event") {
		t.Fatalf("Normalize() error = %q, want ignored UNKOWN statusCode error", err.Error())
	}
}

func flowTaskBody(t *testing.T, statusCode string) []byte {
	t.Helper()

	body := map[string]any{
		"event":  "task",
		"action": "status",
		"task": map[string]any{
			"pipelineId":   "pipe_123",
			"pipelineName": "云效工作流 Agent 构建",
			"stageName":    "构建阶段",
			"taskName":     "Go 单元测试",
			"buildNumber":  "202605110001",
			"statusCode":   statusCode,
			"statusName":   "任务状态",
			"pipelineUrl":  "https://flow.aliyun.com/pipelines/pipe_123/builds/202605110001",
			"message":      "任务状态变更",
		},
		"sources": []map[string]any{
			{
				"repo":            "yunxiao-workflow-agent",
				"branch":          "main",
				"commitId":        "4f9f2cab7a2c0d338da59b9a29e6f2a7e2f76111",
				"privousCommitId": "3a8e2cab7a2c0d338da59b9a29e6f2a7e2f76000",
			},
		},
		"globalParams": []map[string]any{
			{"key": "env", "value": "test"},
		},
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("序列化 Flow 测试 payload 失败: %v", err)
	}
	return encoded
}

func assertExternalRef(t *testing.T, refs map[string]any, key string, want any) {
	t.Helper()

	if got := refs[key]; got != want {
		t.Fatalf("external_refs.%s = %v, want %v", key, got, want)
	}
}
