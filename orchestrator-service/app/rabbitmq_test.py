import sys
import types
import unittest
from dataclasses import dataclass
from unittest.mock import patch

pika_module = types.ModuleType("pika")
pika_module.URLParameters = lambda url: url
pika_module.BlockingConnection = object
sys.modules.setdefault("pika", pika_module)
sys.modules["pika"].URLParameters = lambda url: url
sys.modules["pika"].BlockingConnection = object

from app.rabbitmq import connect_with_retry


@dataclass(frozen=True)
class FakeSettings:
    rabbitmq_url: str = "amqp://example"
    rabbitmq_connect_initial_backoff_seconds: int = 1
    rabbitmq_connect_max_backoff_seconds: int = 4
    rabbitmq_connect_max_wait_seconds: int = 30


class RabbitMQRetryTests(unittest.TestCase):
    def test_connect_with_retry_retries_until_success(self):
        attempts = []
        sleeps = []

        def blocking_connection(parameters):
            attempts.append(parameters)
            if len(attempts) < 3:
                raise RuntimeError("rabbitmq unavailable")
            return "connection"

        with (
            patch("app.rabbitmq.LOGGER"),
            patch("app.rabbitmq.pika.BlockingConnection", side_effect=blocking_connection),
        ):
            connection = connect_with_retry(FakeSettings(), sleep=lambda seconds: sleeps.append(seconds))

        self.assertEqual(connection, "connection")
        self.assertEqual(attempts, ["amqp://example", "amqp://example", "amqp://example"])
        self.assertEqual(sleeps, [1, 2])

    def test_connect_with_retry_raises_after_max_wait(self):
        settings = FakeSettings(rabbitmq_connect_initial_backoff_seconds=2, rabbitmq_connect_max_wait_seconds=3)
        attempts = []

        def blocking_connection(parameters):
            attempts.append(parameters)
            raise RuntimeError("rabbitmq unavailable")

        with (
            patch("app.rabbitmq.LOGGER"),
            patch("app.rabbitmq.pika.BlockingConnection", side_effect=blocking_connection),
        ):
            with self.assertRaises(RuntimeError):
                connect_with_retry(settings, sleep=lambda seconds: None)

        self.assertEqual(len(attempts), 2)


if __name__ == "__main__":
    unittest.main()
