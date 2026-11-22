
from pathlib import Path

import joblib

from app.schemas.ticket import (
    TicketClassificationRequest,
    TicketClassificationResponse,
)


MODEL_DIR = Path(__file__).resolve().parents[2] / "models"


def load_model(filename: str):
    model_path = MODEL_DIR / filename

    if not model_path.is_file():
        raise FileNotFoundError(
            f"Required ML model not found: {model_path}"
        )

    return joblib.load(model_path)


# Load trained models once when the service starts.
vectorizer = load_model("tfidf_vectorizer.joblib")
category_model = load_model("category_model.joblib")
priority_model = load_model("priority_model.joblib")
sentiment_model = load_model("sentiment_model.joblib")


def classify_ticket(
    request: TicketClassificationRequest,
) -> TicketClassificationResponse:

    # Combine subject and description.
    text = f"{request.subject} {request.description}"

    # Apply the trained TF-IDF vectorizer.
    features = vectorizer.transform([text])

    # Predict all three ticket attributes.
    category = category_model.predict(features)[0]
    priority = priority_model.predict(features)[0]
    sentiment = sentiment_model.predict(features)[0]

    # Calculate maximum class probabilities.
    category_confidence = float(
        category_model.predict_proba(features).max()
    )
    priority_confidence = float(
        priority_model.predict_proba(features).max()
    )
    sentiment_confidence = float(
        sentiment_model.predict_proba(features).max()
    )

    # Conservative aggregate confidence (not calibrated).
    confidence = min(
        category_confidence,
        priority_confidence,
        sentiment_confidence,
    )

    return TicketClassificationResponse(
        category=category,
        priority=priority,
        sentiment=sentiment,
        confidence=confidence,
    )
