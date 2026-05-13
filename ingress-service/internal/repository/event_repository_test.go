package repository

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"yunxiao-ingress-service/internal/model"
)

func TestSaveRawEventStoresValidJSONPayload(t *testing.T) {
	repo, mock, closeDB := newMockRepository(t)
	defer closeDB()

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO raw_event
			(source, event_type, external_event_id, signature_valid, headers_json, payload_json,
			 payload_text, process_status, received_at, trace_id, created_at)
		 VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8, $9, $10, $11)
		 RETURNING id`)).
		WithArgs(
			"yunxiao.codeup",
			"Merge Request Hook",
			"delivery_1",
			true,
			sqlmock.AnyArg(),
			`{"object_kind":"merge_request"}`,
			`{"object_kind":"merge_request"}`,
			RawEventStatusReceived,
			sqlmock.AnyArg(),
			"trace_1",
			sqlmock.AnyArg(),
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

	rawID, rawPayloadID, err := repo.SaveRawEvent(
		context.Background(),
		model.RawEventMeta{Source: "yunxiao.codeup", EventType: "Merge Request Hook", ExternalEventID: "delivery_1"},
		map[string][]string{"Codeup-Event": {"Merge Request Hook"}},
		[]byte(`{"object_kind":"merge_request"}`),
		true,
		"trace_1",
	)
	if err != nil {
		t.Fatalf("SaveRawEvent() error = %v", err)
	}
	if rawID != 42 {
		t.Fatalf("rawID = %d, want 42", rawID)
	}
	if rawPayloadID != "raw_42" {
		t.Fatalf("rawPayloadID = %q, want raw_42", rawPayloadID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestSaveRawEventStoresInvalidJSONAsTextOnly(t *testing.T) {
	repo, mock, closeDB := newMockRepository(t)
	defer closeDB()

	mock.ExpectQuery("INSERT INTO raw_event").
		WithArgs(
			"yunxiao.flow",
			"",
			"",
			true,
			sqlmock.AnyArg(),
			nil,
			"{",
			RawEventStatusReceived,
			sqlmock.AnyArg(),
			"trace_invalid",
			sqlmock.AnyArg(),
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))

	rawID, rawPayloadID, err := repo.SaveRawEvent(
		context.Background(),
		model.RawEventMeta{Source: "yunxiao.flow"},
		map[string][]string{},
		[]byte(`{`),
		true,
		"trace_invalid",
	)
	if err != nil {
		t.Fatalf("SaveRawEvent() error = %v", err)
	}
	if rawID != 7 {
		t.Fatalf("rawID = %d, want 7", rawID)
	}
	if rawPayloadID != "raw_7" {
		t.Fatalf("rawPayloadID = %q, want raw_7", rawPayloadID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestRawEventStatusUpdates(t *testing.T) {
	tests := []struct {
		name             string
		run              func(*EventRepository) error
		wantStatus       string
		wantProcessError driver.Value
		wantNormalizedID string
	}{
		{
			name: "invalid payload",
			run: func(repo *EventRepository) error {
				return repo.MarkRawEventInvalidPayload(context.Background(), 1, "invalid json")
			},
			wantStatus:       RawEventStatusInvalidPayload,
			wantProcessError: "invalid json",
		},
		{
			name: "ignored",
			run: func(repo *EventRepository) error {
				return repo.MarkRawEventIgnored(context.Background(), 1, "ignored event")
			},
			wantStatus:       RawEventStatusIgnored,
			wantProcessError: "ignored event",
		},
		{
			name: "normalized",
			run: func(repo *EventRepository) error {
				return repo.MarkRawEventNormalized(context.Background(), 1, "evt_1")
			},
			wantStatus:       RawEventStatusNormalized,
			wantProcessError: nil,
			wantNormalizedID: "evt_1",
		},
		{
			name: "failed",
			run: func(repo *EventRepository) error {
				return repo.MarkRawEventFailed(context.Background(), 1, "save normalized event failed")
			},
			wantStatus:       RawEventStatusFailed,
			wantProcessError: "save normalized event failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, closeDB := newMockRepository(t)
			defer closeDB()

			mock.ExpectExec("UPDATE raw_event").
				WithArgs(tt.wantStatus, tt.wantProcessError, tt.wantNormalizedID, sqlmock.AnyArg(), int64(1)).
				WillReturnResult(sqlmock.NewResult(0, 1))

			if err := tt.run(repo); err != nil {
				t.Fatalf("status update error = %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet sql expectations: %v", err)
			}
		})
	}
}

func newMockRepository(t *testing.T) (*EventRepository, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	return NewEventRepository(db), mock, func() {
		_ = db.Close()
	}
}
