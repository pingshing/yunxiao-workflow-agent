import sys
import types
import unittest
from dataclasses import dataclass
from unittest.mock import patch

psycopg_module = types.ModuleType("psycopg")
psycopg_module.Connection = object
psycopg_errors_module = types.ModuleType("psycopg.errors")
psycopg_errors_module.UniqueViolation = type("UniqueViolation", (Exception,), {})
psycopg_rows_module = types.ModuleType("psycopg.rows")
psycopg_rows_module.dict_row = object()
sys.modules.setdefault("psycopg", psycopg_module)
sys.modules.setdefault("psycopg.errors", psycopg_errors_module)
sys.modules.setdefault("psycopg.rows", psycopg_rows_module)

pika_module = types.ModuleType("pika")
pika_module.URLParameters = object
pika_module.BlockingConnection = object
pika_module.BasicProperties = lambda **kwargs: kwargs
sys.modules.setdefault("pika", pika_module)

from app.consumer import handle_message, workflow_type_for


@dataclass(frozen=True)
class FakeSettings:
    rabbitmq_exchange: str = "yunxiao.events"


class FakePostgresConnection:
    def __init__(self, settings):
        self.settings = settings

    def __enter__(self):
        return "connection"

    def __exit__(self, exc_type, exc, tb):
        return False


class FakeChannel:
    def __init__(self):
        self.acked = False
        self.nacked = False

    def basic_ack(self, delivery_tag):
        self.acked = delivery_tag

    def basic_nack(self, delivery_tag, requeue):
        self.nacked = (delivery_tag, requeue)


class FakeMethod:
    delivery_tag = "tag_1"


class ConsumerTests(unittest.TestCase):
    def test_workflow_type_for_uses_event_registry(self):
        self.assertEqual(workflow_type_for("pipeline.succeeded"), "delivery_summary")
        self.assertEqual(workflow_type_for("unknown.event"), "event_triage")

    def test_handle_message_runs_agent_and_persists_outputs(self):
        body = b"""{
          "event_id": "evt_1",
          "source": "yunxiao.codeup",
          "event_type": "pr.updated",
          "trace_id": "trace_1",
          "dedup_key": "dedup_1",
          "subject": {"type": "pull_request", "id": "123"},
          "external_refs": {"repo_name": "repo"}
        }"""
        channel = FakeChannel()

        with (
            patch("app.consumer.postgres_connection", side_effect=FakePostgresConnection),
            patch("app.consumer.create_workflow_task", return_value=("task_1", True)) as create_task,
            patch("app.consumer.save_context_snapshot", return_value="snap_1") as save_snapshot,
            patch("app.consumer.create_agent_run", return_value="run_1") as create_run,
            patch("app.consumer.save_agent_artifact", return_value="artifact_1") as save_artifact,
            patch("app.consumer.save_agent_actions", return_value=["action_1"]) as save_actions,
            patch("app.consumer.mark_agent_run_succeeded") as mark_run_succeeded,
            patch("app.consumer.mark_task_succeeded") as mark_task_succeeded,
        ):
            handle_message(FakeSettings(), channel, FakeMethod(), body)

        self.assertEqual(channel.acked, "tag_1")
        self.assertFalse(channel.nacked)
        create_task.assert_called_once()
        save_snapshot.assert_called_once()
        create_run.assert_called_once()
        save_artifact.assert_called_once()
        save_actions.assert_called_once()
        mark_run_succeeded.assert_called_once()
        mark_task_succeeded.assert_called_once_with("connection", "task_1", "snap_1")


if __name__ == "__main__":
    unittest.main()
