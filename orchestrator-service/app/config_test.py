import unittest

from app.config import (
    DEV_POSTGRES_DATABASE,
    DEV_POSTGRES_HOST,
    DEV_POSTGRES_PASSWORD,
    DEV_POSTGRES_PORT,
    DEV_POSTGRES_USER,
    DEV_RABBITMQ_URL,
    Settings,
)


def build_settings(**overrides) -> Settings:
    values = {
        "app_env": "dev",
        "postgres_host": DEV_POSTGRES_HOST,
        "postgres_port": DEV_POSTGRES_PORT,
        "postgres_user": DEV_POSTGRES_USER,
        "postgres_password": DEV_POSTGRES_PASSWORD,
        "postgres_database": DEV_POSTGRES_DATABASE,
        "rabbitmq_url": DEV_RABBITMQ_URL,
        "rabbitmq_exchange": "yunxiao.events",
        "rabbitmq_queues": ("orchestrator.pr",),
        "rabbitmq_connect_initial_backoff_seconds": 1,
        "rabbitmq_connect_max_backoff_seconds": 10,
        "rabbitmq_connect_max_wait_seconds": 60,
        "normalized_event_retry_interval_seconds": 30,
        "normalized_event_retry_batch_size": 20,
        "normalized_event_max_attempts": 5,
    }
    values.update(overrides)
    return Settings(**values)


class ConfigValidationTests(unittest.TestCase):
    def test_validate_allows_development_defaults_outside_prod(self):
        build_settings().validate()

    def test_validate_rejects_development_defaults_in_prod(self):
        settings = build_settings(app_env="prod")

        with self.assertRaises(ValueError) as context:
            settings.validate()

        message = str(context.exception)
        self.assertIn("POSTGRES_HOST must not use development default", message)
        self.assertIn("POSTGRES_PASSWORD must not use development default", message)
        self.assertIn("RABBITMQ_URL must not use development default", message)

    def test_validate_allows_explicit_production_config(self):
        settings = build_settings(
            app_env="prod",
            postgres_host="postgres.internal",
            postgres_port=15432,
            postgres_user="prod_user",
            postgres_password="prod_secret",
            postgres_database="prod_agent",
            rabbitmq_url="amqp://prod:secret@rabbitmq:5672/",
        )

        settings.validate()


if __name__ == "__main__":
    unittest.main()
