
from typing import Literal

from pydantic import BaseModel, Field


class TicketClassificationRequest(BaseModel):
    subject: str = Field(min_length=3, max_length=255)
    description: str = Field(min_length=5, max_length=5000)


class TicketClassificationResponse(BaseModel):
    category: Literal[
        "billing",
        "technical_support",
        "account",
        "general_inquiry",
    ]

    priority: Literal[
        "low",
        "medium",
        "high",
        "urgent",
    ]

    sentiment: Literal[
        "positive",
        "neutral",
        "negative",
    ]

    confidence: float = Field(ge=0.0, le=1.0)
