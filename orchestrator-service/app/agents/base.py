from dataclasses import dataclass, field
from typing import Any


@dataclass(frozen=True)
class AgentAction:
    action_type: str
    target_type: str
    target_id: str
    payload: dict[str, Any]


@dataclass(frozen=True)
class AgentResult:
    artifact_type: str
    artifact: dict[str, Any]
    actions: list[AgentAction] = field(default_factory=list)
