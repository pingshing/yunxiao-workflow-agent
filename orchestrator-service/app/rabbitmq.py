import logging
import time
from typing import Callable

import pika

from app.config import Settings

LOGGER = logging.getLogger(__name__)


def connect_with_retry(settings: Settings, sleep: Callable[[float], None] = time.sleep):
    backoff = max(settings.rabbitmq_connect_initial_backoff_seconds, 1)
    max_backoff = max(settings.rabbitmq_connect_max_backoff_seconds, backoff)
    deadline = time.monotonic() + max(settings.rabbitmq_connect_max_wait_seconds, backoff)
    attempt = 1
    last_error = None

    while True:
        try:
            connection = pika.BlockingConnection(pika.URLParameters(settings.rabbitmq_url))
            if attempt > 1:
                LOGGER.info("rabbitmq connected after retry attempts=%s", attempt)
            return connection
        except Exception as exc:
            last_error = exc
            next_time = time.monotonic() + backoff
            if next_time >= deadline:
                break
            LOGGER.warning("rabbitmq connect failed attempt=%s backoff=%ss error=%r", attempt, backoff, exc)
            sleep(backoff)
            backoff = min(backoff * 2, max_backoff)
            attempt += 1

    raise RuntimeError(f"connect rabbitmq after {settings.rabbitmq_connect_max_wait_seconds}s: {last_error}") from last_error
