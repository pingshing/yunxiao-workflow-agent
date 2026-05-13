import unittest

from app.agent_runner import run_agent


class AgentRunnerTests(unittest.TestCase):
    def test_unknown_agent_falls_back_to_event_triage(self):
        result = run_agent(
            "missing_agent",
            {
                "snapshot_id": "snap_1",
                "context_type": "event_triage",
                "context": {
                    "event": {
                        "event_id": "evt_1",
                        "source": "yunxiao.codeup",
                        "event_type": "custom.event",
                        "subject": {"type": "note", "id": "1"},
                    }
                },
                "source_refs": {},
            },
        )

        self.assertEqual(result.artifact_type, "event_triage_result")
        self.assertEqual(result.artifact["event_id"], "evt_1")
        self.assertEqual(result.actions, [])

    def test_pr_review_agent_creates_comment_action(self):
        result = run_agent(
            "pr_review_agent",
            {
                "snapshot_id": "snap_1",
                "context_type": "pr_review",
                "context": {
                    "event": {"event_id": "evt_1", "event_type": "pr.updated"},
                    "pull_request": {"pr_id": "123"},
                },
                "source_refs": {},
            },
        )

        self.assertEqual(result.artifact_type, "pr_review_result")
        self.assertEqual(len(result.actions), 1)
        self.assertEqual(result.actions[0].action_type, "codeup.pr.comment")
        self.assertEqual(result.actions[0].target_id, "123")


if __name__ == "__main__":
    unittest.main()
