package codeup

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeMergeRequestActions(t *testing.T) {
	tests := []struct {
		name          string
		action        string
		wantEventType string
	}{
		{name: "打开合并请求", action: "open", wantEventType: "pr.created"},
		{name: "合并合并请求", action: "merge", wantEventType: "pr.merged"},
		{name: "关闭合并请求", action: "close", wantEventType: "pr.closed"},
		{name: "更新合并请求", action: "update", wantEventType: "pr.updated"},
		{name: "评审通过合并请求", action: "approved", wantEventType: "pr.updated"},
		{name: "未知动作归一为更新", action: "custom_action", wantEventType: "pr.updated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := codeupMergeRequestBody(t, map[string]any{
				"action": tt.action,
				"biz_id": "MR-20260511-0001",
			})

			payload, err := PayloadFromJSON(body)
			if err != nil {
				t.Fatalf("解析 Codeup payload 失败: %v", err)
			}

			event, err := Normalize(payload, "raw_codeup_1", body, "trace_codeup_1")
			if err != nil {
				t.Fatalf("归一化 Codeup payload 失败: %v", err)
			}

			if event.Source != "yunxiao.codeup" {
				t.Fatalf("Source = %q, want %q", event.Source, "yunxiao.codeup")
			}
			if event.EventType != tt.wantEventType {
				t.Fatalf("EventType = %q, want %q", event.EventType, tt.wantEventType)
			}
			if event.Subject.Type != "pull_request" {
				t.Fatalf("Subject.Type = %q, want %q", event.Subject.Type, "pull_request")
			}
			if event.Subject.ID != "MR-20260511-0001" {
				t.Fatalf("Subject.ID = %q, want %q", event.Subject.ID, "MR-20260511-0001")
			}
			if event.RawPayloadID != "raw_codeup_1" {
				t.Fatalf("RawPayloadID = %q, want %q", event.RawPayloadID, "raw_codeup_1")
			}
			if event.TraceID != "trace_codeup_1" {
				t.Fatalf("TraceID = %q, want %q", event.TraceID, "trace_codeup_1")
			}
			if got := event.ExternalRefs["source_branch"]; got != "feature/yunxiao-webhook" {
				t.Fatalf("external_refs.source_branch = %v, want %q", got, "feature/yunxiao-webhook")
			}
			if got := event.ExternalRefs["target_branch"]; got != "main" {
				t.Fatalf("external_refs.target_branch = %v, want %q", got, "main")
			}
			if got := event.ExternalRefs["commit_sha"]; got != "4f9f2cab7a2c0d338da59b9a29e6f2a7e2f76111" {
				t.Fatalf("external_refs.commit_sha = %v, want %q", got, "4f9f2cab7a2c0d338da59b9a29e6f2a7e2f76111")
			}
			if got := event.ExternalRefs["repo_name"]; got != "yunxiao-workflow-agent" {
				t.Fatalf("external_refs.repo_name = %v, want %q", got, "yunxiao-workflow-agent")
			}
			if !strings.HasPrefix(event.DedupKey, "yunxiao.codeup:"+tt.wantEventType+":1001:MR-20260511-0001:") {
				t.Fatalf("DedupKey = %q, want prefix for Codeup event", event.DedupKey)
			}
		})
	}
}

func TestNormalizeMergeRequestIDFallbackUsesOfficialIID(t *testing.T) {
	body := codeupMergeRequestBody(t, map[string]any{
		"action": "update",
		"biz_id": "",
		"iid":    17,
	})

	payload, err := PayloadFromJSON(body)
	if err != nil {
		t.Fatalf("解析 Codeup payload 失败: %v", err)
	}

	event, err := Normalize(payload, "raw_codeup_iid", body, "trace_codeup_iid")
	if err != nil {
		t.Fatalf("归一化 Codeup payload 失败: %v", err)
	}

	if event.Subject.ID != "17" {
		t.Fatalf("Subject.ID = %q, want %q", event.Subject.ID, "17")
	}
}

func TestNormalizePushEvent(t *testing.T) {
	body := codeupPushBody(t, "push", "refs/heads/main")

	payload, err := PayloadFromJSON(body)
	if err != nil {
		t.Fatalf("解析 Codeup push payload 失败: %v", err)
	}

	event, err := Normalize(payload, "raw_codeup_push", body, "trace_codeup_push")
	if err != nil {
		t.Fatalf("归一化 Codeup push payload 失败: %v", err)
	}

	if event.EventType != "repo.pushed" {
		t.Fatalf("EventType = %q, want %q", event.EventType, "repo.pushed")
	}
	if event.Subject.Type != "branch" {
		t.Fatalf("Subject.Type = %q, want %q", event.Subject.Type, "branch")
	}
	if event.Subject.ID != "main" {
		t.Fatalf("Subject.ID = %q, want %q", event.Subject.ID, "main")
	}
	if got := event.ExternalRefs["commit_sha"]; got != "8f7e2cab7a2c0d338da59b9a29e6f2a7e2f76222" {
		t.Fatalf("external_refs.commit_sha = %v, want %q", got, "8f7e2cab7a2c0d338da59b9a29e6f2a7e2f76222")
	}
	if got := event.ExternalRefs["ref_name"]; got != "main" {
		t.Fatalf("external_refs.ref_name = %v, want %q", got, "main")
	}
	if event.DedupKey != "yunxiao.codeup:repo.pushed:1001:main:8f7e2cab7a2c0d338da59b9a29e6f2a7e2f76222" {
		t.Fatalf("DedupKey = %q, want deterministic push dedup key", event.DedupKey)
	}
}

func TestNormalizeTagPushEvent(t *testing.T) {
	body := codeupPushBody(t, "tag_push", "refs/tags/v1.0.0")

	payload, err := PayloadFromJSON(body)
	if err != nil {
		t.Fatalf("解析 Codeup tag push payload 失败: %v", err)
	}

	event, err := Normalize(payload, "raw_codeup_tag", body, "trace_codeup_tag")
	if err != nil {
		t.Fatalf("归一化 Codeup tag push payload 失败: %v", err)
	}

	if event.EventType != "repo.tag_pushed" {
		t.Fatalf("EventType = %q, want %q", event.EventType, "repo.tag_pushed")
	}
	if event.Subject.Type != "tag" {
		t.Fatalf("Subject.Type = %q, want %q", event.Subject.Type, "tag")
	}
	if event.Subject.ID != "v1.0.0" {
		t.Fatalf("Subject.ID = %q, want %q", event.Subject.ID, "v1.0.0")
	}
}

func TestNormalizeNoteEvent(t *testing.T) {
	body := codeupNoteBody(t)

	payload, err := PayloadFromJSON(body)
	if err != nil {
		t.Fatalf("解析 Codeup note payload 失败: %v", err)
	}

	event, err := Normalize(payload, "raw_codeup_note", body, "trace_codeup_note")
	if err != nil {
		t.Fatalf("归一化 Codeup note payload 失败: %v", err)
	}

	if event.EventType != "repo.note_created" {
		t.Fatalf("EventType = %q, want %q", event.EventType, "repo.note_created")
	}
	if event.Subject.Type != "note" {
		t.Fatalf("Subject.Type = %q, want %q", event.Subject.Type, "note")
	}
	if event.Subject.ID != "70001" {
		t.Fatalf("Subject.ID = %q, want %q", event.Subject.ID, "70001")
	}
	if got := event.ExternalRefs["merge_request_id"]; got != "MR-20260511-0001" {
		t.Fatalf("external_refs.merge_request_id = %v, want %q", got, "MR-20260511-0001")
	}
}

func TestNormalizeIgnoredUnknownEvent(t *testing.T) {
	body := []byte(`{
		"object_kind": "wiki_page",
		"event_name": "wiki_page",
		"project_id": 1001
	}`)

	payload, err := PayloadFromJSON(body)
	if err != nil {
		t.Fatalf("解析 Codeup unknown payload 失败: %v", err)
	}

	_, err = Normalize(payload, "raw_codeup_unknown", body, "trace_codeup_unknown")
	if err == nil {
		t.Fatal("Normalize() error = nil, want ignored event error")
	}
	if !strings.Contains(err.Error(), "ignored codeup event") {
		t.Fatalf("Normalize() error = %q, want ignored event error", err.Error())
	}
}

func codeupMergeRequestBody(t *testing.T, overrides map[string]any) []byte {
	t.Helper()

	objectAttributes := map[string]any{
		"id":                90001,
		"iid":               12,
		"local_id":          12,
		"biz_id":            "MR-20260511-0001",
		"title":             "接入云效 Codeup Webhook",
		"description":       "补充合并请求事件接入",
		"action":            "open",
		"state":             "opened",
		"source_branch":     "feature/yunxiao-webhook",
		"target_branch":     "main",
		"source_project_id": 1001,
		"target_project_id": 1001,
		"created_at":        "2026-05-11T10:00:00+08:00",
		"updated_at":        "2026-05-11T10:03:00+08:00",
		"last_commit": map[string]any{
			"id":        "4f9f2cab7a2c0d338da59b9a29e6f2a7e2f76111",
			"message":   "feat: add codeup webhook adapter",
			"timestamp": "2026-05-11T10:02:00+08:00",
			"url":       "https://codeup.aliyun.com/acme/yunxiao-workflow-agent/commit/4f9f2cab7a2c0d338da59b9a29e6f2a7e2f76111",
		},
	}
	for key, value := range overrides {
		objectAttributes[key] = value
	}

	body := map[string]any{
		"object_kind": "merge_request",
		"event_name":  "merge_request",
		"project_id":  1001,
		"user": map[string]any{
			"name":     "张三",
			"username": "zhangsan",
			"email":    "zhangsan@example.com",
		},
		"repository": map[string]any{
			"name":         "yunxiao-workflow-agent",
			"url":          "https://codeup.aliyun.com/acme/yunxiao-workflow-agent",
			"git_http_url": "https://codeup.aliyun.com/acme/yunxiao-workflow-agent.git",
			"git_ssh_url":  "git@codeup.aliyun.com:acme/yunxiao-workflow-agent.git",
			"homepage":     "https://codeup.aliyun.com/acme/yunxiao-workflow-agent",
		},
		"object_attributes": objectAttributes,
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("序列化 Codeup 测试 payload 失败: %v", err)
	}
	return encoded
}

func codeupPushBody(t *testing.T, objectKind string, ref string) []byte {
	t.Helper()

	body := map[string]any{
		"object_kind":         objectKind,
		"event_name":          objectKind,
		"project_id":          1001,
		"ref":                 ref,
		"before":              "4f9f2cab7a2c0d338da59b9a29e6f2a7e2f76111",
		"after":               "8f7e2cab7a2c0d338da59b9a29e6f2a7e2f76222",
		"checkout_sha":        "8f7e2cab7a2c0d338da59b9a29e6f2a7e2f76222",
		"total_commits_count": 1,
		"repository": map[string]any{
			"name":         "yunxiao-workflow-agent",
			"git_http_url": "https://codeup.aliyun.com/acme/yunxiao-workflow-agent.git",
		},
		"user": map[string]any{
			"name":     "张三",
			"username": "zhangsan",
		},
		"commits": []map[string]any{
			{
				"id":        "8f7e2cab7a2c0d338da59b9a29e6f2a7e2f76222",
				"message":   "docs: update webhook guide",
				"timestamp": "2026-05-11T10:10:00+08:00",
				"url":       "https://codeup.aliyun.com/acme/yunxiao-workflow-agent/commit/8f7e2cab7a2c0d338da59b9a29e6f2a7e2f76222",
			},
		},
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("序列化 Codeup push 测试 payload 失败: %v", err)
	}
	return encoded
}

func codeupNoteBody(t *testing.T) []byte {
	t.Helper()

	body := map[string]any{
		"object_kind": "note",
		"event_name":  "note",
		"project_id":  1001,
		"user": map[string]any{
			"name":     "张三",
			"username": "zhangsan",
		},
		"repository": map[string]any{
			"name":         "yunxiao-workflow-agent",
			"git_http_url": "https://codeup.aliyun.com/acme/yunxiao-workflow-agent.git",
		},
		"object_attributes": map[string]any{
			"id":         70001,
			"title":      "请补充边界条件测试",
			"action":     "create",
			"created_at": "2026-05-11T10:12:00+08:00",
			"updated_at": "2026-05-11T10:12:00+08:00",
		},
		"merge_request": map[string]any{
			"id":     90001,
			"iid":    12,
			"biz_id": "MR-20260511-0001",
			"title":  "接入云效 Codeup Webhook",
		},
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("序列化 Codeup note 测试 payload 失败: %v", err)
	}
	return encoded
}
