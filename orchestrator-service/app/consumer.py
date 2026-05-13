import json
import logging

from app.agent_runner import run_agent
from app.config import Settings
from app.context_builders import build_context
from app.db import postgres_connection
from app.event_registry import resolve_event_route
from app.models import NormalizedEvent
from app.rabbitmq import connect_with_retry
from app.repositories import (
    create_agent_run,
    create_workflow_task,
    mark_agent_run_failed,
    mark_agent_run_succeeded,
    mark_task_failed,
    mark_task_succeeded,
    save_agent_actions,
    save_agent_artifact,
    save_context_snapshot,
)

LOGGER = logging.getLogger(__name__)


def workflow_type_for(event_type: str) -> str:
    return resolve_event_route(event_type).workflow_type


def consume(settings: Settings) -> None:
    connection = connect_with_retry(settings)
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
        "orchestrator.pr": ["pr.*"],
        "orchestrator.repo": ["repo.*"],
        "orchestrator.pipeline": ["pipeline.*"],
        "orchestrator.work_item": ["work_item.*"],
    }
    for queue_name, routing_keys in bindings.items():
        channel.queue_declare(queue=queue_name, durable=True)
        for routing_key in routing_keys:
            channel.queue_bind(queue=queue_name, exchange=settings.rabbitmq_exchange, routing_key=routing_key)


def handle_message(settings: Settings, channel, method, body: bytes) -> None:
    event: NormalizedEvent = json.loads(body.decode("utf-8"))
    route = resolve_event_route(event["event_type"])
    workflow_type = route.workflow_type
    run_id = None

    try:
        with postgres_connection(settings) as connection:
            task_id, created = create_workflow_task(connection, event, workflow_type)
            if not created:
                LOGGER.info("duplicate workflow task ignored: event_id=%s workflow=%s", event["event_id"], workflow_type)
                channel.basic_ack(delivery_tag=method.delivery_tag)
                return

            context_type, context, source_refs = build_context(event, route)
            snapshot_id = save_context_snapshot(connection, event, context_type, context, source_refs)
            context_snapshot = {
                "snapshot_id": snapshot_id,
                "context_type": context_type,
                "context": context,
                "source_refs": source_refs,
            }
            run_id = create_agent_run(
                connection,
                task_id=task_id,
                event=event,
                workflow_type=workflow_type,
                agent_type=route.agent_type,
                input_snapshot_id=snapshot_id,
                model="rules-v1",
            )
            try:
                agent_result = run_agent(route.agent_type, context_snapshot)
            except Exception as agent_exc:
                mark_agent_run_failed(connection, run_id, str(agent_exc))
                mark_task_failed(connection, task_id, str(agent_exc))
                LOGGER.exception(
                    "agent workflow failed: event_id=%s workflow_type=%s agent_type=%s run_id=%s",
                    event["event_id"],
                    workflow_type,
                    route.agent_type,
                    run_id,
                )
                channel.basic_ack(delivery_tag=method.delivery_tag)
                return
            artifact_id = save_agent_artifact(
                connection,
                run_id=run_id,
                event_id=event["event_id"],
                artifact_type=agent_result.artifact_type,
                artifact=agent_result.artifact,
            )
            action_payloads = [
                {
                    "action_type": action.action_type,
                    "target_type": action.target_type,
                    "target_id": action.target_id,
                    "payload": action.payload,
                }
                for action in agent_result.actions
            ]
            action_ids = save_agent_actions(connection, run_id, event["event_id"], action_payloads)
            mark_agent_run_succeeded(
                connection,
                run_id,
                {
                    "artifact_id": artifact_id,
                    "artifact_type": agent_result.artifact_type,
                    "action_ids": action_ids,
                },
            )
            mark_task_succeeded(connection, task_id, snapshot_id)

        LOGGER.info(
            "agent workflow succeeded: event_id=%s workflow_type=%s agent_type=%s snapshot_id=%s run_id=%s",
            event["event_id"],
            workflow_type,
            route.agent_type,
            snapshot_id,
            run_id,
        )
        channel.basic_ack(delivery_tag=method.delivery_tag)
    except Exception as exc:
        LOGGER.exception("consume failed: event_id=%s", event.get("event_id"))
        try:
            with postgres_connection(settings) as connection:
                task_id, _ = create_workflow_task(connection, event, workflow_type)
                if run_id:
                    mark_agent_run_failed(connection, run_id, str(exc))
                mark_task_failed(connection, task_id, str(exc))
        except Exception:
            LOGGER.exception("mark task failed failed")
        channel.basic_nack(delivery_tag=method.delivery_tag, requeue=False)
