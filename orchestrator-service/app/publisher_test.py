import sys
import types
import unittest

pika_module = types.ModuleType("pika")
pika_module.BasicProperties = lambda **kwargs: kwargs
sys.modules.setdefault("pika", pika_module)

from app.publisher import build_normalized_event_message, publish_normalized_event


class FakeChannel:
    def __init__(self):
        self.published = None

    def basic_publish(self, **kwargs):
        self.published = kwargs


class PublisherTests(unittest.TestCase):
    def test_build_message_preserves_ingress_contract(self):
        row = {
            "event_id": "evt_1",
            "source": "yunxiao.codeup",
            "event_type": "pr.updated",
            "trace_id": "trace_1",
            "dedup_key": "dedup_1",
            "project_id": None,
            "work_item_id": None,
            "subject_type": "pull_request",
            "subject_id": "123",
            "external_refs_json": {"repo_name": "repo"},
            "raw_event_id": 42,
            "occurred_at": None,
        }

        message = build_normalized_event_message(row)

        self.assertEqual(message["raw_payload_id"], "raw_42")
        self.assertEqual(message["subject"], {"type": "pull_request", "id": "123"})
        self.assertEqual(message["external_refs"], {"repo_name": "repo"})
        self.assertNotIn("project_id", message)
        self.assertNotIn("work_item_id", message)

    def test_publish_uses_event_type_as_routing_key(self):
        channel = FakeChannel()
        row = {
            "event_id": "evt_1",
            "source": "yunxiao.flow",
            "event_type": "pipeline.failed",
            "trace_id": "trace_1",
            "dedup_key": "dedup_1",
            "subject_type": "pipeline_run",
            "subject_id": "88",
            "external_refs_json": {},
            "raw_event_id": 7,
            "occurred_at": None,
        }

        publish_normalized_event(channel, "yunxiao.events", row)

        self.assertEqual(channel.published["exchange"], "yunxiao.events")
        self.assertEqual(channel.published["routing_key"], "pipeline.failed")
        self.assertEqual(channel.published["properties"]["message_id"], "evt_1")
        self.assertEqual(channel.published["properties"]["headers"]["x-trace-id"], "trace_1")


if __name__ == "__main__":
    unittest.main()
