import unittest

from app.event_registry import resolve_event_route


class EventRegistryTests(unittest.TestCase):
    def test_resolves_core_events(self):
        self.assertEqual(resolve_event_route("pr.updated").workflow_type, "pr_review")
        self.assertEqual(resolve_event_route("pipeline.failed").agent_type, "pipeline_diagnosis_agent")
        self.assertEqual(resolve_event_route("pipeline.succeeded").workflow_type, "delivery_summary")
        self.assertEqual(resolve_event_route("work_item.updated").agent_type, "work_item_agent")

    def test_unknown_event_goes_to_triage_agent(self):
        route = resolve_event_route("custom.event")

        self.assertEqual(route.workflow_type, "event_triage")
        self.assertEqual(route.agent_type, "event_triage_agent")
        self.assertEqual(route.context_type, "event_triage")


if __name__ == "__main__":
    unittest.main()
