import json
import logging

import pika

from app.config import Settings
from app.context_builders import build_context
from app.db import postgres_connection
from app.models import NormalizedEvent
from app.repositories import (
    create_workflow_task,
    mark_task_failed,
    mark_task_succeeded,
    save_context_snapshot,
)

LOGGER = logging.getLogger(__name__)


def workflow_type_for(event_type: str) -> str:
    if event_type.startswith("pr."):
        return "pr_review"
    if event_type == "pipeline.failed":
        return "pipeline_failure"
    if event_type.startswith("work_item."):
        return "work_item_sync"
    return "unknown"


def consume(settings: Settings) -> None:
    parameters = pika.URLParameters(settings.rabbitmq_url)
    connection = pika.BlockingConnection(parameters)
    channel = connection.channel()
    channel.exchange_declare(exchange=settings.rabbitmq_exchange, exchange_type="topic", durable=True)

    declare_bindings(channel, settings)
    channel.basic_qos(prefetch_count=1)

    for queue_name in settings.rabbitmq_queues:
        channel.basic_consume(queue=queue_name, on_message_callback=lambda ch, method, props, body: handle_message(settings, ch, method, body))

    LOGGER.info("orchestrator consuming queues: %s", ", ".join(settings.rabbitmq_queues))
    channel.start_consuming()


def declare_bindings(channel, settings: Settings) -> None:
    bindings = {
        "orchestrator.pr": ["pr.created", "pr.updated"],
        "orchestrator.pipeline": ["pipeline.failed"],
        "orchestrator.work_item": ["work_item.updated", "work_item.status_changed"],
    }
    for queue_name, routing_keys in bindings.items():
        channel.queue_declare(queue=queue_name, durable=True)
        for routing_key in routing_keys:
            channel.queue_bind(queue=queue_name, exchange=settings.rabbitmq_exchange, routing_key=routing_key)


def handle_message(settings: Settings, channel, method, body: bytes) -> None:
    event: NormalizedEvent = json.loads(body.decode("utf-8"))
    workflow_type = workflow_type_for(event["event_type"])

    try:
        with postgres_connection(settings) as connection:
            task_id, created = create_workflow_task(connection, event, workflow_type)
            if not created:
                LOGGER.info("duplicate workflow task ignored: event_id=%s workflow=%s", event["event_id"], workflow_type)
                channel.basic_ack(delivery_tag=method.delivery_tag)
                return

            context_type, context, source_refs = build_context(event)
            snapshot_id = save_context_snapshot(connection, event, context_type, context, source_refs)
            mark_task_succeeded(connection, task_id, snapshot_id)

        LOGGER.info("context snapshot saved: event_id=%s snapshot_id=%s", event["event_id"], snapshot_id)
        channel.basic_ack(delivery_tag=method.delivery_tag)
    except Exception as exc:
        LOGGER.exception("consume failed: event_id=%s", event.get("event_id"))
        try:
            with postgres_connection(settings) as connection:
                task_id, _ = create_workflow_task(connection, event, workflow_type)
                mark_task_failed(connection, task_id, str(exc))
        except Exception:
            LOGGER.exception("mark task failed failed")
        channel.basic_nack(delivery_tag=method.delivery_tag, requeue=False)
