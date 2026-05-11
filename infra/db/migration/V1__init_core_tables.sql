CREATE TABLE raw_event (
  id BIGSERIAL PRIMARY KEY,
  source VARCHAR(64) NOT NULL,
  event_type VARCHAR(128) NULL,
  external_event_id VARCHAR(255) NULL,
  signature_valid BOOLEAN NOT NULL DEFAULT FALSE,
  headers_json JSONB NULL,
  payload_json JSONB NOT NULL,
  received_at TIMESTAMP NOT NULL,
  trace_id VARCHAR(128) NOT NULL,
  created_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_raw_event_source_event_created_at ON raw_event (source, event_type, created_at);
CREATE INDEX idx_raw_event_external_event_id ON raw_event (external_event_id);
CREATE INDEX idx_raw_event_trace_id ON raw_event (trace_id);

CREATE TABLE normalized_event (
  id BIGSERIAL PRIMARY KEY,
  event_id VARCHAR(128) NOT NULL,
  source VARCHAR(64) NOT NULL,
  event_type VARCHAR(128) NOT NULL,
  occurred_at TIMESTAMP NULL,
  trace_id VARCHAR(128) NOT NULL,
  dedup_key VARCHAR(255) NOT NULL,
  project_id VARCHAR(128) NULL,
  work_item_id VARCHAR(128) NULL,
  subject_type VARCHAR(64) NOT NULL,
  subject_id VARCHAR(255) NOT NULL,
  external_refs_json JSONB NULL,
  raw_event_id BIGINT NOT NULL,
  publish_status VARCHAR(32) NOT NULL,
  published_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL,
  CONSTRAINT uk_event_id UNIQUE (event_id),
  CONSTRAINT uk_dedup_key UNIQUE (dedup_key),
  CONSTRAINT fk_normalized_event_raw_event FOREIGN KEY (raw_event_id) REFERENCES raw_event(id)
);

CREATE INDEX idx_normalized_event_project_event_created_at ON normalized_event (project_id, event_type, created_at);
CREATE INDEX idx_normalized_event_work_item_id ON normalized_event (work_item_id);
CREATE INDEX idx_normalized_event_subject ON normalized_event (subject_type, subject_id);
CREATE INDEX idx_normalized_event_trace_id ON normalized_event (trace_id);

CREATE TABLE context_snapshot (
  id BIGSERIAL PRIMARY KEY,
  snapshot_id VARCHAR(128) NOT NULL,
  event_id VARCHAR(128) NOT NULL,
  context_type VARCHAR(64) NOT NULL,
  subject_type VARCHAR(64) NOT NULL,
  subject_id VARCHAR(255) NOT NULL,
  project_id VARCHAR(128) NULL,
  work_item_id VARCHAR(128) NULL,
  context_json JSONB NOT NULL,
  source_refs_json JSONB NULL,
  created_at TIMESTAMP NOT NULL,
  CONSTRAINT uk_snapshot_id UNIQUE (snapshot_id)
);

CREATE INDEX idx_context_snapshot_event_id ON context_snapshot (event_id);
CREATE INDEX idx_context_snapshot_subject ON context_snapshot (subject_type, subject_id);
CREATE INDEX idx_context_snapshot_context_type_created_at ON context_snapshot (context_type, created_at);

CREATE TABLE workflow_task (
  id BIGSERIAL PRIMARY KEY,
  task_id VARCHAR(128) NOT NULL,
  event_id VARCHAR(128) NOT NULL,
  event_type VARCHAR(128) NOT NULL,
  workflow_type VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  retry_count INT NOT NULL DEFAULT 0,
  trace_id VARCHAR(128) NOT NULL,
  context_snapshot_id VARCHAR(128) NULL,
  error_message TEXT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  CONSTRAINT uk_task_id UNIQUE (task_id),
  CONSTRAINT uk_event_workflow UNIQUE (event_id, workflow_type)
);

CREATE INDEX idx_workflow_task_status_created_at ON workflow_task (status, created_at);
CREATE INDEX idx_workflow_task_trace_id ON workflow_task (trace_id);
CREATE INDEX idx_workflow_task_context_snapshot_id ON workflow_task (context_snapshot_id);
