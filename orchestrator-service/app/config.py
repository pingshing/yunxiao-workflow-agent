import os
from dataclasses import dataclass

DEV_POSTGRES_HOST = "127.0.0.1"
DEV_POSTGRES_PORT = 5432
DEV_POSTGRES_USER = "yunxiao"
DEV_POSTGRES_PASSWORD = "yunxiao"
DEV_POSTGRES_DATABASE = "yunxiao_agent"
DEV_RABBITMQ_URL = "amqp://guest:guest@127.0.0.1:5672/"


@dataclass(frozen=True)
class Settings:
    app_env: str
    postgres_host: str
    postgres_port: int
    postgres_user: str
    postgres_password: str
    postgres_database: str
    rabbitmq_url: str
    rabbitmq_exchange: str
    rabbitmq_queues: tuple[str, ...]
    rabbitmq_connect_initial_backoff_seconds: int
    rabbitmq_connect_max_backoff_seconds: int
    rabbitmq_connect_max_wait_seconds: int
    normalized_event_retry_interval_seconds: int
    normalized_event_retry_batch_size: int
    normalized_event_max_attempts: int

    def validate(self) -> None:
        if self.app_env.lower() != "prod":
            return

        problems = []
        if self.postgres_host.strip() == "" or self.postgres_host == DEV_POSTGRES_HOST:
            problems.append("POSTGRES_HOST must not use development default")
        if self.postgres_port == DEV_POSTGRES_PORT:
            problems.append("POSTGRES_PORT must not use development default")
        if self.postgres_user.strip() == "" or self.postgres_user == DEV_POSTGRES_USER:
            problems.append("POSTGRES_USER must not use development default")
        if self.postgres_password.strip() == "" or self.postgres_password == DEV_POSTGRES_PASSWORD:
            problems.append("POSTGRES_PASSWORD must not use development default")
        if self.postgres_database.strip() == "" or self.postgres_database == DEV_POSTGRES_DATABASE:
            problems.append("POSTGRES_DATABASE must not use development default")
        if self.rabbitmq_url.strip() == "" or self.rabbitmq_url == DEV_RABBITMQ_URL:
            problems.append("RABBITMQ_URL must not use development default")
        if self.rabbitmq_exchange.strip() == "":
            problems.append("RABBITMQ_EXCHANGE is required")
        if not self.rabbitmq_queues:
            problems.append("RABBITMQ_QUEUES is required")

        if problems:
            raise ValueError("; ".join(problems))


def load_settings() -> Settings:
    queues = os.getenv(
        "RABBITMQ_QUEUES",
        "orchestrator.pr,orchestrator.repo,orchestrator.pipeline,orchestrator.work_item",
    )
    return Settings(
        app_env=os.getenv("APP_ENV", "dev"),
        postgres_host=os.getenv("POSTGRES_HOST", DEV_POSTGRES_HOST),
        postgres_port=int(os.getenv("POSTGRES_PORT", str(DEV_POSTGRES_PORT))),
        postgres_user=os.getenv("POSTGRES_USER", DEV_POSTGRES_USER),
        postgres_password=os.getenv("POSTGRES_PASSWORD", DEV_POSTGRES_PASSWORD),
        postgres_database=os.getenv("POSTGRES_DATABASE", DEV_POSTGRES_DATABASE),
        rabbitmq_url=os.getenv("RABBITMQ_URL", DEV_RABBITMQ_URL),
        rabbitmq_exchange=os.getenv("RABBITMQ_EXCHANGE", "yunxiao.events"),
        rabbitmq_queues=tuple(queue.strip() for queue in queues.split(",") if queue.strip()),
        rabbitmq_connect_initial_backoff_seconds=int(os.getenv("RABBITMQ_CONNECT_INITIAL_BACKOFF_SECONDS", "1")),
        rabbitmq_connect_max_backoff_seconds=int(os.getenv("RABBITMQ_CONNECT_MAX_BACKOFF_SECONDS", "10")),
        rabbitmq_connect_max_wait_seconds=int(os.getenv("RABBITMQ_CONNECT_MAX_WAIT_SECONDS", "60")),
        normalized_event_retry_interval_seconds=int(os.getenv("NORMALIZED_EVENT_RETRY_INTERVAL_SECONDS", "30")),
        normalized_event_retry_batch_size=int(os.getenv("NORMALIZED_EVENT_RETRY_BATCH_SIZE", "20")),
        normalized_event_max_attempts=int(os.getenv("NORMALIZED_EVENT_MAX_ATTEMPTS", "5")),
    )
