package projex

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeDefaultEventType(t *testing.T) {
	body := projexWorkItemBody(t, map[string]any{
		"identifier": "REQ-001",
	})

	payload, err := PayloadFromJSON(body)
	if err != nil {
		t.Fatalf("解析 Projex payload 失败: %v", err)
	}

	event, err := Normalize(payload, "", "raw_projex_1", body, "trace_projex_1")
	if err != nil {
		t.Fatalf("归一化 Projex payload 失败: %v", err)
	}

	if event.Source != "yunxiao.projex" {
		t.Fatalf("Source = %q, want %q", event.Source, "yunxiao.projex")
	}
	if event.EventType != "work_item.updated" {
		t.Fatalf("EventType = %q, want %q", event.EventType, "work_item.updated")
	}
	if event.ProjectID != "PROJ_AGENT" {
		t.Fatalf("ProjectID = %q, want %q", event.ProjectID, "PROJ_AGENT")
	}
	if event.WorkItemID != "REQ-001" {
		t.Fatalf("WorkItemID = %q, want %q", event.WorkItemID, "REQ-001")
	}
	if event.Subject.Type != "work_item" {
		t.Fatalf("Subject.Type = %q, want %q", event.Subject.Type, "work_item")
	}
	if event.Subject.ID != "REQ-001" {
		t.Fatalf("Subject.ID = %q, want %q", event.Subject.ID, "REQ-001")
	}
	if event.RawPayloadID != "raw_projex_1" {
		t.Fatalf("RawPayloadID = %q, want %q", event.RawPayloadID, "raw_projex_1")
	}
	if event.TraceID != "trace_projex_1" {
		t.Fatalf("TraceID = %q, want %q", event.TraceID, "trace_projex_1")
	}
	assertExternalRef(t, event.ExternalRefs, "projex_space_id", "space_1001")
	assertExternalRef(t, event.ExternalRefs, "projex_space_name", "云效工作流 Agent")
	assertExternalRef(t, event.ExternalRefs, "category_id", "Req")
	assertExternalRef(t, event.ExternalRefs, "id_path", "workitem_epic_1/workitem_10001")
	assertExternalRef(t, event.ExternalRefs, "serial_number", "001")
	assertExternalRef(t, event.ExternalRefs, "logical_status", "normal")
	assertExternalRef(t, event.ExternalRefs, "work_item_title", "接入云效 Webhook")
	assertExternalRef(t, event.ExternalRefs, "work_item_status", "开发中")
	assertExternalRef(t, event.ExternalRefs, "status_stage_id", "stage_in_progress")
	assertExternalRef(t, event.ExternalRefs, "work_item_type", "需求")
	assertExternalRef(t, event.ExternalRefs, "work_item_type_id", "type_requirement")
	assertExternalRef(t, event.ExternalRefs, "assignee", "李四")
	assertExternalRef(t, event.ExternalRefs, "assignee_id", "user_lisi")
	assertExternalRef(t, event.ExternalRefs, "creator", "张三")
	assertExternalRef(t, event.ExternalRefs, "modifier_id", "user_wangwu")
	assertExternalRef(t, event.ExternalRefs, "modifier", "王五")
	assertExternalRef(t, event.ExternalRefs, "verifier", "赵六")
	assertExternalRef(t, event.ExternalRefs, "sprint", "第一迭代")
	assertExternalRef(t, event.ExternalRefs, "parent_id", "EPIC-001")
	participants, ok := event.ExternalRefs["participants"].([]map[string]string)
	if !ok {
		t.Fatalf("external_refs.participants type = %T, want []map[string]string", event.ExternalRefs["participants"])
	}
	if len(participants) != 1 || participants[0]["display_name"] != "钱七" {
		t.Fatalf("external_refs.participants = %v, want 钱七", participants)
	}
	trackers, ok := event.ExternalRefs["trackers"].([]map[string]string)
	if !ok {
		t.Fatalf("external_refs.trackers type = %T, want []map[string]string", event.ExternalRefs["trackers"])
	}
	if len(trackers) != 1 || trackers[0]["display_name"] != "孙八" {
		t.Fatalf("external_refs.trackers = %v, want 孙八", trackers)
	}
	customFields, ok := event.ExternalRefs["custom_fields"].(map[string]any)
	if !ok {
		t.Fatalf("external_refs.custom_fields type = %T, want map[string]any", event.ExternalRefs["custom_fields"])
	}
	if customFields["priority"] != "P1" {
		t.Fatalf("external_refs.custom_fields.priority = %v, want %q", customFields["priority"], "P1")
	}
	tags, ok := customFields["tags"].([]map[string]string)
	if !ok {
		t.Fatalf("external_refs.custom_fields.tags type = %T, want []map[string]string", customFields["tags"])
	}
	if len(tags) != 1 || tags[0]["display_value"] != "后端" {
		t.Fatalf("external_refs.custom_fields.tags = %v, want 后端", tags)
	}
	if !strings.HasPrefix(event.DedupKey, "yunxiao.projex:work_item.updated:REQ-001:") {
		t.Fatalf("DedupKey = %q, want Projex dedup key prefix", event.DedupKey)
	}
}

func TestNormalizeExplicitStatusChangedEventType(t *testing.T) {
	body := projexWorkItemBody(t, map[string]any{
		"identifier": "",
		"id":         "workitem_10001",
		"space": map[string]any{
			"id":          "space_1001",
			"identifier":  "",
			"displayName": "云效工作流 Agent",
		},
	})

	payload, err := PayloadFromJSON(body)
	if err != nil {
		t.Fatalf("解析 Projex payload 失败: %v", err)
	}

	event, err := Normalize(payload, "work_item.status_changed", "raw_projex_status", body, "trace_projex_status")
	if err != nil {
		t.Fatalf("归一化 Projex payload 失败: %v", err)
	}

	if event.EventType != "work_item.status_changed" {
		t.Fatalf("EventType = %q, want %q", event.EventType, "work_item.status_changed")
	}
	if event.ProjectID != "space_1001" {
		t.Fatalf("ProjectID = %q, want %q", event.ProjectID, "space_1001")
	}
	if event.WorkItemID != "workitem_10001" {
		t.Fatalf("WorkItemID = %q, want %q", event.WorkItemID, "workitem_10001")
	}
	if event.Subject.ID != "workitem_10001" {
		t.Fatalf("Subject.ID = %q, want %q", event.Subject.ID, "workitem_10001")
	}
}

func TestNormalizeSupportedWorkItemEventTypes(t *testing.T) {
	tests := []string{
		"work_item.created",
		"work_item.updated",
		"work_item.status_changed",
		"work_item.assignee_changed",
		"work_item.deleted",
	}

	for _, eventType := range tests {
		t.Run(eventType, func(t *testing.T) {
			body := projexWorkItemBody(t, map[string]any{})

			payload, err := PayloadFromJSON(body)
			if err != nil {
				t.Fatalf("解析 Projex payload 失败: %v", err)
			}

			event, err := Normalize(payload, eventType, "raw_projex_supported", body, "trace_projex_supported")
			if err != nil {
				t.Fatalf("归一化 Projex payload 失败: %v", err)
			}

			if event.EventType != eventType {
				t.Fatalf("EventType = %q, want %q", event.EventType, eventType)
			}
		})
	}
}

func TestNormalizeIgnoredWorkItemEventType(t *testing.T) {
	body := projexWorkItemBody(t, map[string]any{})

	payload, err := PayloadFromJSON(body)
	if err != nil {
		t.Fatalf("解析 Projex payload 失败: %v", err)
	}

	_, err = Normalize(payload, "work_item.priority_changed", "raw_projex_ignored", body, "trace_projex_ignored")
	if err == nil {
		t.Fatal("Normalize() error = nil, want ignored event_type error")
	}
	if !strings.Contains(err.Error(), "ignored projex event") {
		t.Fatalf("Normalize() error = %q, want ignored event_type error", err.Error())
	}
}

func TestNormalizeMissingWorkItemID(t *testing.T) {
	body := projexWorkItemBody(t, map[string]any{
		"identifier": "",
		"id":         "",
	})

	payload, err := PayloadFromJSON(body)
	if err != nil {
		t.Fatalf("解析 Projex payload 失败: %v", err)
	}

	_, err = Normalize(payload, "work_item.updated", "raw_projex_missing_id", body, "trace_projex_missing_id")
	if err == nil {
		t.Fatal("Normalize() error = nil, want missing work item id error")
	}
	if !strings.Contains(err.Error(), "projex work item id is required") {
		t.Fatalf("Normalize() error = %q, want missing work item id error", err.Error())
	}
}

func projexWorkItemBody(t *testing.T, overrides map[string]any) []byte {
	t.Helper()

	body := map[string]any{
		"categoryId":    "Req",
		"id":            "workitem_10001",
		"idPath":        "workitem_epic_1/workitem_10001",
		"identifier":    "REQ-001",
		"serialNumber":  "001",
		"subject":       "接入云效 Webhook",
		"description":   "通过自动化规则将工作项数据推送到接入服务",
		"formatType":    "RICHTEXT",
		"logicalStatus": "normal",
		"status": map[string]any{
			"id":            "status_doing",
			"name":          "doing",
			"displayName":   "开发中",
			"statusStageId": "stage_in_progress",
		},
		"assignedTo": map[string]any{
			"id":          "user_lisi",
			"name":        "lisi",
			"displayName": "李四",
		},
		"creator": map[string]any{
			"id":          "user_zhangsan",
			"name":        "zhangsan",
			"displayName": "张三",
		},
		"modifier": map[string]any{
			"id":          "user_wangwu",
			"name":        "wangwu",
			"displayName": "王五",
		},
		"verifier": map[string]any{
			"id":          "user_zhaoliu",
			"name":        "zhaoliu",
			"displayName": "赵六",
		},
		"space": map[string]any{
			"id":          "space_1001",
			"identifier":  "PROJ_AGENT",
			"displayName": "云效工作流 Agent",
		},
		"sprint": map[string]any{
			"id":          "sprint_1",
			"displayName": "第一迭代",
		},
		"parent": map[string]any{
			"id":         "workitem_epic_1",
			"identifier": "EPIC-001",
		},
		"participants": []map[string]any{
			{"id": "user_qianqi", "displayName": "钱七"},
		},
		"trackers": []map[string]any{
			{"id": "user_sunba", "displayName": "孙八"},
		},
		"versions": []map[string]any{
			{"id": "version_1", "displayName": "v1.0.0"},
		},
		"workitemType": map[string]any{
			"id":          "type_requirement",
			"name":        "requirement",
			"displayName": "需求",
		},
		"labels": []map[string]any{
			{"id": "label_backend", "displayName": "后端"},
		},
		"customFieldValues": []map[string]any{
			{"fieldId": "cf_priority", "fieldIdentifier": "priority", "fieldName": "优先级", "fieldFormat": "list", "value": "P1"},
			{"fieldId": "cf_tags", "fieldIdentifier": "tags", "fieldName": "标签", "fieldFormat": "multiList", "values": []map[string]any{
				{"displayValue": "后端", "identifier": "backend"},
			}},
		},
		"gmtCreate":      "2026-05-11T10:00:00.000Z",
		"gmtModified":    "2026-05-11T10:30:00.000Z",
		"updateStatusAt": "2026-05-11T10:20:00.000Z",
	}
	for key, value := range overrides {
		body[key] = value
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("序列化 Projex 测试 payload 失败: %v", err)
	}
	return encoded
}

func assertExternalRef(t *testing.T, refs map[string]any, key string, want any) {
	t.Helper()

	if got := refs[key]; got != want {
		t.Fatalf("external_refs.%s = %v, want %v", key, got, want)
	}
}
