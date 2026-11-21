
from app.schemas.ticket import (
    TicketClassificationRequest,
    TicketClassificationResponse,
)


CATEGORY_KEYWORDS = {
    "billing": [
        "payment", "invoice", "refund", "charge",
        "billing", "transaction", "subscription",
    ],
    "technical_support": [
        "error", "bug", "crash", "not working",
        "failed", "issue", "slow", "technical",
    ],
    "account": [
        "login", "password", "account", "sign in",
        "reset password", "locked", "profile",
    ],
}


URGENT_KEYWORDS = [
    "urgent", "critical", "immediately",
    "emergency", "security breach", "hacked",
]

HIGH_KEYWORDS = [
    "cannot access", "not working", "failed",
    "unable", "blocked", "data loss",
]

NEGATIVE_KEYWORDS = [
    "angry", "frustrated", "disappointed",
    "terrible", "worst", "unhappy",
    "annoyed", "unacceptable",
]

POSITIVE_KEYWORDS = [
    "thank you", "thanks", "excellent",
    "great", "appreciate", "wonderful",
]


def classify_ticket(
    request: TicketClassificationRequest,
) -> TicketClassificationResponse:

    text = f"{request.subject} {request.description}".lower()

    # 1. Predict category
    category_scores = {
        category: sum(
            1 for keyword in keywords if keyword in text
        )
        for category, keywords in CATEGORY_KEYWORDS.items()
    }

    best_category = max(
        category_scores,
        key=category_scores.get,
    )

    if category_scores[best_category] == 0:
        best_category = "general_inquiry"

    # 2. Predict priority
    if any(keyword in text for keyword in URGENT_KEYWORDS):
        priority = "urgent"
    elif any(keyword in text for keyword in HIGH_KEYWORDS):
        priority = "high"
    elif any(
        keyword in text
        for keyword in ["question", "information", "inquiry"]
    ):
        priority = "low"
    else:
        priority = "medium"

    # 3. Predict sentiment
    negative_count = sum(
        1 for keyword in NEGATIVE_KEYWORDS if keyword in text
    )

    positive_count = sum(
        1 for keyword in POSITIVE_KEYWORDS if keyword in text
    )

    if negative_count > positive_count:
        sentiment = "negative"
    elif positive_count > negative_count:
        sentiment = "positive"
    else:
        sentiment = "neutral"

    # 4. Estimate confidence from keyword matches.
    # This is a heuristic score, not a calibrated AI probability.
    total_matches = sum(category_scores.values())

    confidence = (
        min(0.95, 0.60 + (0.10 * total_matches))
        if total_matches > 0
        else 0.50
    )

    return TicketClassificationResponse(
        category=best_category,
        priority=priority,
        sentiment=sentiment,
        confidence=confidence,
    )
