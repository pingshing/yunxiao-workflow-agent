import unittest
import sys
import types

psycopg_module = types.ModuleType("psycopg")
psycopg_module.Connection = object
psycopg_errors_module = types.ModuleType("psycopg.errors")
psycopg_errors_module.UniqueViolation = type("UniqueViolation", (Exception,), {})
sys.modules.setdefault("psycopg", psycopg_module)
sys.modules.setdefault("psycopg.errors", psycopg_errors_module)

from app.repositories import (
    create_agent_run,
    fetch_retry_candidates,
    mark_agent_run_failed,
    mark_agent_run_succeeded,
    mark_publish_retry_failed,
    mark_publish_retry_succeeded,
    save_agent_actions,
    save_agent_artifact,
)


class FakeCursor:
    def __init__(self, rows=None):
        self.rows = rows or []
        self.executed_sql = ""
        self.executed_args = None

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb):
        return False

    def execute(self, sql, args):
        self.executed_sql = sql
        self.executed_args = args

    def fetchall(self):
        return self.rows


class FakeConnection:
    def __init__(self, rows=None):
        self.cursor_instance = FakeCursor(rows)

    def cursor(self):
        return self.cursor_instance


class RepositoryTests(unittest.TestCase):
    def test_fetch_retry_candidates_uses_skip_locked_and_max_attempts(self):
        connection = FakeConnection(rows=[{"event_id": "evt_1", "publish_attempts": 1}])

        rows = fetch_retry_candidates(connection, limit=20, max_attempts=5)

        self.assertEqual(rows, [{"event_id": "evt_1", "publish_attempts": 1}])
        self.assertIn("publish_status IN ('pending', 'failed')", connection.cursor_instance.executed_sql)
        self.assertIn("FOR UPDATE SKIP LOCKED", connection.cursor_instance.executed_sql)
        self.assertEqual(connection.cursor_instance.executed_args, (5, 20))

    def test_mark_publish_retry_succeeded_updates_published_state(self):
        connection = FakeConnection()

        mark_publish_retry_succeeded(connection, "evt_1")

        self.assertIn("publish_status = 'published'", connection.cursor_instance.executed_sql)
        self.assertIn("last_publish_error = NULL", connection.cursor_instance.executed_sql)
        self.assertEqual(connection.cursor_instance.executed_args[2], "evt_1")

    def test_mark_publish_retry_failed_increments_attempts(self):
        connection = FakeConnection()

        mark_publish_retry_failed(connection, "evt_1", "publish failed", max_attempts=5)

        self.assertIn("publish_attempts = publish_attempts + 1", connection.cursor_instance.executed_sql)
        self.assertIn("last_publish_error", connection.cursor_instance.executed_sql)
        self.assertEqual(connection.cursor_instance.executed_args[0], "publish failed")
        self.assertEqual(connection.cursor_instance.executed_args[2], "evt_1")

    def test_create_agent_run_inserts_running_state(self):
        connection = FakeConnection()
        event = {"event_id": "evt_1", "event_type": "pr.updated", "trace_id": "trace_1"}

        run_id = create_agent_run(connection, "task_1", event, "pr_review", "pr_review_agent", "snap_1", "rules-v1")

        self.assertTrue(run_id.startswith("run_"))
        self.assertIn("INSERT INTO agent_run", connection.cursor_instance.executed_sql)
        self.assertIn("'running'", connection.cursor_instance.executed_sql)
        self.assertEqual(connection.cursor_instance.executed_args[1], "task_1")
        self.assertEqual(connection.cursor_instance.executed_args[4], "pr_review_agent")

    def test_mark_agent_run_succeeded_persists_raw_output(self):
        connection = FakeConnection()

        mark_agent_run_succeeded(connection, "run_1", {"artifact_id": "artifact_1"})

        self.assertIn("status = 'succeeded'", connection.cursor_instance.executed_sql)
        self.assertEqual(connection.cursor_instance.executed_args[3], "run_1")

    def test_mark_agent_run_failed_persists_error(self):
        connection = FakeConnection()

        mark_agent_run_failed(connection, "run_1", "agent failed")

        self.assertIn("status = 'failed'", connection.cursor_instance.executed_sql)
        self.assertEqual(connection.cursor_instance.executed_args[1], "agent failed")
        self.assertEqual(connection.cursor_instance.executed_args[3], "run_1")

    def test_save_agent_artifact_inserts_artifact_json(self):
        connection = FakeConnection()

        artifact_id = save_agent_artifact(connection, "run_1", "evt_1", "event_triage_result", {"summary": "ok"})

        self.assertTrue(artifact_id.startswith("artifact_"))
        self.assertIn("INSERT INTO agent_artifact", connection.cursor_instance.executed_sql)
        self.assertEqual(connection.cursor_instance.executed_args[1], "run_1")
        self.assertEqual(connection.cursor_instance.executed_args[3], "event_triage_result")

    def test_save_agent_actions_inserts_pending_actions(self):
        connection = FakeConnection()

        action_ids = save_agent_actions(
            connection,
            "run_1",
            "evt_1",
            [
                {
                    "action_type": "codeup.pr.comment",
                    "target_type": "pull_request",
                    "target_id": "123",
                    "payload": {"body": "hello"},
                }
            ],
        )

        self.assertEqual(len(action_ids), 1)
        self.assertTrue(action_ids[0].startswith("action_"))
        self.assertIn("INSERT INTO agent_action_outbox", connection.cursor_instance.executed_sql)
        self.assertIn("'pending'", connection.cursor_instance.executed_sql)
        self.assertEqual(connection.cursor_instance.executed_args[3], "codeup.pr.comment")


if __name__ == "__main__":
    unittest.main()
