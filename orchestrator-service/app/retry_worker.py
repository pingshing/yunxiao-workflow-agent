import logging
import threading
import time

from app.config import Settings
from app.db import postgres_connection
from app.publisher import publish_normalized_event
from app.rabbitmq import connect_with_retry
from app.repositories import (
    fetch_retry_candidates,
    mark_publish_retry_failed,
    mark_publish_retry_succeeded,
)

LOGGER = logging.getLogger(__name__)


def start_retry_worker(settings: Settings) -> threading.Thread:
    thread = threading.Thread(target=run_retry_worker, args=(settings,), daemon=True)
    thread.start()
    return thread


def run_retry_worker(settings: Settings) -> None:
    while True:
        connection = None
        try:
            connection = connect_with_retry(settings)
            channel = connection.channel()
            channel.exchange_declare(exchange=settings.rabbitmq_exchange, exchange_type="topic", durable=True)

            while True:
                handled = process_retry_batch(settings, channel)
                if handled == 0:
                    time.sleep(settings.normalized_event_retry_interval_seconds)
        except Exception:
            LOGGER.exception("normalized event retry worker failed")
            time.sleep(settings.normalized_event_retry_interval_seconds)
        finally:
            if connection is not None and not connection.is_closed:
                connection.close()


def process_retry_batch(settings: Settings, channel) -> int:
    with postgres_connection(settings) as connection:
        events = fetch_retry_candidates(connection, settings.normalized_event_retry_batch_size, settings.normalized_event_max_attempts)
        if not events:
            return 0

        handled = 0
        for event in events:
            try:
                publish_normalized_event(channel, settings.rabbitmq_exchange, event)
                mark_publish_retry_succeeded(connection, event["event_id"])
                handled += 1
                LOGGER.info(
                    "normalized event republished event_id=%s source=%s event_type=%s publish_status=published",
                    event["event_id"],
                    event["source"],
                    event["event_type"],
                )
            except Exception as exc:
                mark_publish_retry_failed(connection, event["event_id"], str(exc), settings.normalized_event_max_attempts)
                LOGGER.exception(
                    "normalized event republish failed event_id=%s source=%s event_type=%s",
                    event["event_id"],
                    event["source"],
                    event["event_type"],
                )
        return handled
