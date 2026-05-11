import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Settings:
    postgres_host: str
    postgres_port: int
    postgres_user: str
    postgres_password: str
    postgres_database: str
    rabbitmq_url: str
    rabbitmq_exchange: str
    rabbitmq_queues: tuple[str, ...]


def load_settings() -> Settings:
    queues = os.getenv(
        "RABBITMQ_QUEUES",
        "orchestrator.pr,orchestrator.pipeline,orchestrator.work_item",
    )
    return Settings(
        postgres_host=os.getenv("POSTGRES_HOST", "127.0.0.1"),
        postgres_port=int(os.getenv("POSTGRES_PORT", "5432")),
        postgres_user=os.getenv("POSTGRES_USER", "yunxiao"),
        postgres_password=os.getenv("POSTGRES_PASSWORD", "yunxiao"),
        postgres_database=os.getenv("POSTGRES_DATABASE", "yunxiao_agent"),
        rabbitmq_url=os.getenv("RABBITMQ_URL", "amqp://guest:guest@127.0.0.1:5672/"),
        rabbitmq_exchange=os.getenv("RABBITMQ_EXCHANGE", "yunxiao.events"),
        rabbitmq_queues=tuple(queue.strip() for queue in queues.split(",") if queue.strip()),
    )
