CREATE TABLE agent_run (
  id BIGSERIAL PRIMARY KEY,
  run_id VARCHAR(128) NOT NULL,
  task_id VARCHAR(128) NOT NULL,
  event_id VARCHAR(128) NOT NULL,
  workflow_type VARCHAR(64) NOT NULL,
  agent_type VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  model VARCHAR(128) NULL,
  input_snapshot_id VARCHAR(128) NOT NULL,
  started_at TIMESTAMP NOT NULL,
  finished_at TIMESTAMP NULL,
  error_message TEXT NULL,
  raw_output_json JSONB NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  CONSTRAINT uk_agent_run_id UNIQUE (run_id),
  CONSTRAINT fk_agent_run_task_id FOREIGN KEY (task_id) REFERENCES workflow_task(task_id),
  CONSTRAINT fk_agent_run_snapshot_id FOREIGN KEY (input_snapshot_id) REFERENCES context_snapshot(snapshot_id)
);

CREATE INDEX idx_agent_run_event_id ON agent_run (event_id);
CREATE INDEX idx_agent_run_task_id ON agent_run (task_id);
CREATE INDEX idx_agent_run_status_created_at ON agent_run (status, created_at);
CREATE INDEX idx_agent_run_workflow_agent ON agent_run (workflow_type, agent_type);

CREATE TABLE agent_artifact (
  id BIGSERIAL PRIMARY KEY,
  artifact_id VARCHAR(128) NOT NULL,
  run_id VARCHAR(128) NOT NULL,
  event_id VARCHAR(128) NOT NULL,
  artifact_type VARCHAR(64) NOT NULL,
  artifact_json JSONB NOT NULL,
  created_at TIMESTAMP NOT NULL,
  CONSTRAINT uk_agent_artifact_id UNIQUE (artifact_id),
  CONSTRAINT fk_agent_artifact_run_id FOREIGN KEY (run_id) REFERENCES agent_run(run_id)
);

CREATE INDEX idx_agent_artifact_run_id ON agent_artifact (run_id);
CREATE INDEX idx_agent_artifact_event_id ON agent_artifact (event_id);
CREATE INDEX idx_agent_artifact_type_created_at ON agent_artifact (artifact_type, created_at);

CREATE TABLE agent_action_outbox (
  id BIGSERIAL PRIMARY KEY,
  action_id VARCHAR(128) NOT NULL,
  run_id VARCHAR(128) NOT NULL,
  event_id VARCHAR(128) NOT NULL,
  action_type VARCHAR(64) NOT NULL,
  target_type VARCHAR(64) NOT NULL,
  target_id VARCHAR(255) NOT NULL,
  payload_json JSONB NOT NULL,
  status VARCHAR(32) NOT NULL,
  attempts INT NOT NULL DEFAULT 0,
  last_error TEXT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  executed_at TIMESTAMP NULL,
  CONSTRAINT uk_agent_action_id UNIQUE (action_id),
  CONSTRAINT fk_agent_action_run_id FOREIGN KEY (run_id) REFERENCES agent_run(run_id)
);

CREATE INDEX idx_agent_action_status_created_at ON agent_action_outbox (status, created_at);
CREATE INDEX idx_agent_action_run_id ON agent_action_outbox (run_id);
CREATE INDEX idx_agent_action_event_id ON agent_action_outbox (event_id);
