import unittest
import sys
import types
from dataclasses import dataclass
from unittest.mock import patch

pika_module = types.ModuleType("pika")
pika_module.URLParameters = object
pika_module.BlockingConnection = object
sys.modules.setdefault("pika", pika_module)

psycopg_module = types.ModuleType("psycopg")
psycopg_module.Connection = object
psycopg_module.connect = object
psycopg_errors_module = types.ModuleType("psycopg.errors")
psycopg_errors_module.UniqueViolation = type("UniqueViolation", (Exception,), {})
psycopg_rows_module = types.ModuleType("psycopg.rows")
psycopg_rows_module.dict_row = object()
sys.modules.setdefault("psycopg", psycopg_module)
sys.modules.setdefault("psycopg.errors", psycopg_errors_module)
sys.modules.setdefault("psycopg.rows", psycopg_rows_module)

from app.retry_worker import process_retry_batch


@dataclass(frozen=True)
class FakeSettings:
    rabbitmq_url: str = "amqp://example"
    rabbitmq_exchange: str = "yunxiao.events"
    rabbitmq_connect_initial_backoff_seconds: int = 1
    rabbitmq_connect_max_backoff_seconds: int = 10
    rabbitmq_connect_max_wait_seconds: int = 60
    normalized_event_retry_batch_size: int = 10
    normalized_event_max_attempts: int = 3


class FakePostgresConnection:
    def __init__(self, settings):
        self.settings = settings

    def __enter__(self):
        return "connection"

    def __exit__(self, exc_type, exc, tb):
        return False


class RetryWorkerTests(unittest.TestCase):
    def test_process_retry_batch_marks_success(self):
        event = {"event_id": "evt_1", "source": "yunxiao.codeup", "event_type": "pr.updated"}

        with (
            patch("app.retry_worker.postgres_connection", side_effect=FakePostgresConnection),
            patch("app.retry_worker.fetch_retry_candidates", return_value=[event]) as fetch_candidates,
            patch("app.retry_worker.publish_normalized_event") as publish_event,
            patch("app.retry_worker.mark_publish_retry_succeeded") as mark_succeeded,
            patch("app.retry_worker.mark_publish_retry_failed") as mark_failed,
        ):
            handled = process_retry_batch(FakeSettings(), object())

        self.assertEqual(handled, 1)
        fetch_candidates.assert_called_once_with("connection", 10, 3)
        publish_event.assert_called_once()
        mark_succeeded.assert_called_once_with("connection", "evt_1")
        mark_failed.assert_not_called()

    def test_process_retry_batch_marks_failure(self):
        event = {"event_id": "evt_1", "source": "yunxiao.flow", "event_type": "pipeline.failed"}

        with (
            patch("app.retry_worker.LOGGER"),
            patch("app.retry_worker.postgres_connection", side_effect=FakePostgresConnection),
            patch("app.retry_worker.fetch_retry_candidates", return_value=[event]),
            patch("app.retry_worker.publish_normalized_event", side_effect=RuntimeError("mq down")),
            patch("app.retry_worker.mark_publish_retry_succeeded") as mark_succeeded,
            patch("app.retry_worker.mark_publish_retry_failed") as mark_failed,
        ):
            handled = process_retry_batch(FakeSettings(), object())

        self.assertEqual(handled, 0)
        mark_succeeded.assert_not_called()
        mark_failed.assert_called_once_with("connection", "evt_1", "mq down", 3)


if __name__ == "__main__":
    unittest.main()
