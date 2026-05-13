ALTER TABLE normalized_event
  ADD COLUMN publish_attempts INT NOT NULL DEFAULT 0,
  ADD COLUMN last_publish_error TEXT NULL,
  ADD COLUMN last_publish_attempt_at TIMESTAMP NULL;

CREATE INDEX idx_normalized_event_publish_status_attempts_created_at
  ON normalized_event (publish_status, publish_attempts, created_at);
