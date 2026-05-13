package httpserver

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"yunxiao-ingress-service/internal/config"
	"yunxiao-ingress-service/internal/model"
	"yunxiao-ingress-service/internal/repository"
)

func TestCodeupInvalidJSONMarksRawEventInvalidPayload(t *testing.T) {
	server, repo, _ := newTestServer()
	logs := captureLogs(t)

	response := performRequest(server, http.MethodPost, "/webhooks/yunxiao/codeup", "{", map[string]string{
		"X-Codeup-Token": "codeup-token",
		"Codeup-Event":   "Merge Request Hook",
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if repo.invalidPayloadRawID != 1 {
		t.Fatalf("invalidPayloadRawID = %d, want 1", repo.invalidPayloadRawID)
	}
	if repo.lastRaw.payloadJSONValid {
		t.Fatal("payloadJSONValid = true, want false for invalid JSON")
	}
	if !strings.Contains(logs.String(), "process_status=invalid_payload") {
		t.Fatalf("logs = %q, want invalid_payload", logs.String())
	}
}

func TestFlowIgnoredEventMarksRawEventIgnored(t *testing.T) {
	server, repo, _ := newTestServer()
	logs := captureLogs(t)
	body := `{
		"event": "task",
		"action": "status",
		"task": {
			"pipelineId": "pipe_1",
			"buildNumber": "202605110001",
			"statusCode": "UNKOWN"
		}
	}`

	response := performRequest(server, http.MethodPost, "/webhooks/yunxiao/flow", body, map[string]string{
		"Authorization": "Bearer flow-token",
	})

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if repo.ignoredRawID != 1 {
		t.Fatalf("ignoredRawID = %d, want 1", repo.ignoredRawID)
	}
	if !strings.Contains(repo.ignoredError, "ignored flow event") {
		t.Fatalf("ignoredError = %q, want ignored flow event", repo.ignoredError)
	}
	if !strings.Contains(logs.String(), "process_status=ignored") {
		t.Fatalf("logs = %q, want ignored", logs.String())
	}
}

func TestProjexPublishedMarksRawEventNormalized(t *testing.T) {
	server, repo, publisher := newTestServer()
	logs := captureLogs(t)
	body := `{
		"id": "workitem_1",
		"identifier": "REQ-001",
		"subject": "接入云效 Webhook",
		"space": {"id": "space_1", "identifier": "PROJ_AGENT"},
		"gmtModified": "2026-05-11T10:30:00.000Z"
	}`

	response := performRequest(server, http.MethodPost, "/webhooks/yunxiao/projex?event_type=work_item.updated", body, map[string]string{
		"X-Projex-Signature": "projex-secret",
	})

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if repo.normalizedRawID != 1 {
		t.Fatalf("normalizedRawID = %d, want 1", repo.normalizedRawID)
	}
	if repo.normalizedEventID == "" {
		t.Fatal("normalizedEventID is empty")
	}
	if publisher.publishedEventID == "" {
		t.Fatal("publishedEventID is empty")
	}
	if repo.publishedEventID != publisher.publishedEventID {
		t.Fatalf("publishedEventID = %q, want %q", repo.publishedEventID, publisher.publishedEventID)
	}
	if !strings.Contains(logs.String(), "process_status=normalized") {
		t.Fatalf("logs = %q, want normalized", logs.String())
	}
}

func TestPublishFailedStillKeepsRawEventNormalized(t *testing.T) {
	server, repo, publisher := newTestServer()
	logs := captureLogs(t)
	publisher.err = errors.New("rabbitmq unavailable")
	body := codeupMergeRequestWebhookBody()

	response := performRequest(server, http.MethodPost, "/webhooks/yunxiao/codeup", body, map[string]string{
		"X-Codeup-Token": "codeup-token",
		"Codeup-Event":   "Merge Request Hook",
	})

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if !strings.Contains(response.Body.String(), "stored_publish_failed") {
		t.Fatalf("body = %q, want stored_publish_failed", response.Body.String())
	}
	if repo.normalizedRawID != 1 {
		t.Fatalf("normalizedRawID = %d, want 1", repo.normalizedRawID)
	}
	if repo.publishFailedEventID == "" {
		t.Fatal("publishFailedEventID is empty")
	}
	if !strings.Contains(logs.String(), "process_status=normalized") {
		t.Fatalf("logs = %q, want normalized log for publish failure", logs.String())
	}
}

func TestCodeupNormalizeFailureMarksRawEventFailed(t *testing.T) {
	server, repo, _ := newTestServer()
	logs := captureLogs(t)
	body := `{
		"object_kind": "merge_request",
		"project_id": 1001,
		"object_attributes": {
			"action": "open",
			"source_branch": "feature/test",
			"target_branch": "main"
		},
		"repository": {"name": "yunxiao-workflow-agent"},
		"user": {"username": "zhangsan"}
	}`

	response := performRequest(server, http.MethodPost, "/webhooks/yunxiao/codeup", body, map[string]string{
		"X-Codeup-Token": "codeup-token",
		"Codeup-Event":   "Merge Request Hook",
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if repo.failedRawID != 1 {
		t.Fatalf("failedRawID = %d, want 1", repo.failedRawID)
	}
	if !strings.Contains(logs.String(), "process_status=failed") {
		t.Fatalf("logs = %q, want failed", logs.String())
	}
}

func TestCodeupDuplicateEventIsLoggedAsNormalized(t *testing.T) {
	server, repo, _ := newTestServer()
	logs := captureLogs(t)
	body := codeupMergeRequestWebhookBody()

	first := performRequest(server, http.MethodPost, "/webhooks/yunxiao/codeup", body, map[string]string{
		"X-Codeup-Token": "codeup-token",
		"Codeup-Event":   "Merge Request Hook",
	})
	if first.Code != http.StatusAccepted {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusAccepted)
	}
	second := performRequest(server, http.MethodPost, "/webhooks/yunxiao/codeup", body, map[string]string{
		"X-Codeup-Token": "codeup-token",
		"Codeup-Event":   "Merge Request Hook",
	})

	if second.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d", second.Code, http.StatusOK)
	}
	if repo.normalizedRawID != 2 {
		t.Fatalf("normalizedRawID = %d, want 2", repo.normalizedRawID)
	}
	if !strings.Contains(logs.String(), "process_status=normalized") {
		t.Fatalf("logs = %q, want normalized", logs.String())
	}
}

func TestUnauthorizedWebhookRequestsAreLogged(t *testing.T) {
	server, _, _ := newTestServer()
	logs := captureLogs(t)

	response := performRequest(server, http.MethodPost, "/webhooks/yunxiao/codeup", "{}", map[string]string{
		"X-Codeup-Token": "wrong",
	})

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(logs.String(), "process_status=unauthorized") {
		t.Fatalf("logs = %q, want unauthorized", logs.String())
	}
}

func newTestServer() (*Server, *fakeRepository, *fakePublisher) {
	gin.SetMode(gin.TestMode)
	repo := &fakeRepository{}
	publisher := &fakePublisher{}
	server := New(config.Config{
		CodeupToken:  "codeup-token",
		ProjexSecret: "projex-secret",
		FlowToken:    "flow-token",
	}, repo, publisher)
	return server, repo, publisher
}

func performRequest(server *Server, method string, path string, body string, headers map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)
	return response
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buffer bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&buffer)
	t.Cleanup(func() {
		log.SetOutput(previous)
	})
	return &buffer
}

func codeupMergeRequestWebhookBody() string {
	return `{
		"object_kind": "merge_request",
		"project_id": 1001,
		"object_attributes": {
			"id": 90001,
			"iid": 12,
			"biz_id": "MR-1",
			"action": "open",
			"source_branch": "feature/test",
			"target_branch": "main",
			"updated_at": "2026-05-11T10:03:00+08:00"
		},
		"repository": {"name": "yunxiao-workflow-agent"},
		"user": {"username": "zhangsan"}
	}`
}

type rawEventRecord struct {
	meta             model.RawEventMeta
	payloadJSONValid bool
}

type fakeRepository struct {
	nextRawID            int64
	lastRaw              rawEventRecord
	invalidPayloadRawID  int64
	ignoredRawID         int64
	ignoredError         string
	normalizedRawID      int64
	normalizedEventID    string
	failedRawID          int64
	publishedEventID     string
	publishFailedEventID string
	normalizedByDedup    map[string]string
}

func (r *fakeRepository) SaveRawEvent(ctx context.Context, meta model.RawEventMeta, headers map[string][]string, payload []byte, signatureValid bool, traceID string) (int64, string, error) {
	r.nextRawID++
	r.lastRaw = rawEventRecord{meta: meta, payloadJSONValid: json.Valid(payload)}
	return r.nextRawID, "raw_1", nil
}

func (r *fakeRepository) UpdateRawEventMeta(ctx context.Context, rawEventID int64, meta model.RawEventMeta) error {
	r.lastRaw.meta = meta
	return nil
}

func (r *fakeRepository) MarkRawEventInvalidPayload(ctx context.Context, rawEventID int64, processError string) error {
	r.invalidPayloadRawID = rawEventID
	return nil
}

func (r *fakeRepository) MarkRawEventIgnored(ctx context.Context, rawEventID int64, processError string) error {
	r.ignoredRawID = rawEventID
	r.ignoredError = processError
	return nil
}

func (r *fakeRepository) MarkRawEventNormalized(ctx context.Context, rawEventID int64, normalizedEventID string) error {
	r.normalizedRawID = rawEventID
	r.normalizedEventID = normalizedEventID
	return nil
}

func (r *fakeRepository) MarkRawEventFailed(ctx context.Context, rawEventID int64, processError string) error {
	r.failedRawID = rawEventID
	return nil
}

func (r *fakeRepository) SaveNormalizedEvent(ctx context.Context, event model.NormalizedEvent, rawEventID int64) error {
	if r.normalizedByDedup == nil {
		r.normalizedByDedup = map[string]string{}
	}
	if eventID, exists := r.normalizedByDedup[event.DedupKey]; exists {
		_ = eventID
		return repository.ErrDuplicateEvent
	}
	r.normalizedByDedup[event.DedupKey] = event.EventID
	return nil
}

func (r *fakeRepository) FindNormalizedEventIDByDedupKey(ctx context.Context, dedupKey string) (string, error) {
	if eventID, exists := r.normalizedByDedup[dedupKey]; exists {
		return eventID, nil
	}
	return "", sql.ErrNoRows
}

func (r *fakeRepository) MarkPublished(ctx context.Context, eventID string) error {
	r.publishedEventID = eventID
	return nil
}

func (r *fakeRepository) MarkPublishFailed(ctx context.Context, eventID string) error {
	r.publishFailedEventID = eventID
	return nil
}

type fakePublisher struct {
	err              error
	publishedEventID string
}

func (p *fakePublisher) Publish(ctx context.Context, event model.NormalizedEvent) error {
	if p.err != nil {
		return p.err
	}
	p.publishedEventID = event.EventID
	return nil
}
