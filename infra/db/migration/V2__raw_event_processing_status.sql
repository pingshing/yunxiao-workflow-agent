ALTER TABLE raw_event
  ADD COLUMN payload_text TEXT NULL,
  ADD COLUMN process_status VARCHAR(32) NOT NULL DEFAULT 'received',
  ADD COLUMN process_error TEXT NULL,
  ADD COLUMN normalized_event_id VARCHAR(128) NULL,
  ADD COLUMN processed_at TIMESTAMP NULL;

ALTER TABLE raw_event
  ALTER COLUMN payload_json DROP NOT NULL;

UPDATE raw_event
SET payload_text = payload_json::text
WHERE payload_text IS NULL AND payload_json IS NOT NULL;

CREATE INDEX idx_raw_event_process_status_created_at ON raw_event (process_status, created_at);
CREATE INDEX idx_raw_event_normalized_event_id ON raw_event (normalized_event_id);
