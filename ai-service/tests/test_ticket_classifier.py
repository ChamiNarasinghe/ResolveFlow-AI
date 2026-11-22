
import unittest
import math

from app.schemas.ticket import TicketClassificationRequest
from app.services.ticket_classifier import classify_ticket


class TestTicketClassifier(unittest.TestCase):

    def test_billing_classification(self):
        request = TicketClassificationRequest(
            subject="Payment refund request",
            description="I was charged twice for my subscription.",
        )

        result = classify_ticket(request)

        self.assertEqual(result.category, "billing")

    def test_urgent_priority(self):
        request = TicketClassificationRequest(
            subject="Urgent security issue",
            description="My account has been hacked. Please help immediately.",
        )

        result = classify_ticket(request)

        self.assertEqual(result.priority, "urgent")

    def test_sentiment_prediction_is_valid(self):
        request = TicketClassificationRequest(
            subject="Very disappointed",
            description="I am frustrated with this terrible service.",
        )

        result = classify_ticket(request)

        self.assertIn(
            result.sentiment,
            ["positive", "neutral", "negative"],
        )

    def test_general_inquiry(self):
        request = TicketClassificationRequest(
            subject="Product information",
            description="I would like to know more about your services.",
        )

        result = classify_ticket(request)

        self.assertEqual(result.category, "general_inquiry")

    def test_confidence_range(self):
        request = TicketClassificationRequest(
            subject="Payment problem",
            description="My payment failed.",
        )

        result = classify_ticket(request)

        self.assertTrue(math.isfinite(result.confidence))
        self.assertGreaterEqual(result.confidence, 0.0)
        self.assertLessEqual(result.confidence, 1.0)

    def test_all_predictions_are_valid(self):
        request = TicketClassificationRequest(
            subject="Account access problem",
            description="I cannot log in to my account.",
        )

        result = classify_ticket(request)

        self.assertIn(
            result.category,
            ["billing", "technical_support", "account", "general_inquiry"],
        )
        self.assertIn(
            result.priority,
            ["low", "medium", "high", "urgent"],
        )
        self.assertIn(
            result.sentiment,
            ["positive", "neutral", "negative"],
        )


if __name__ == "__main__":
    unittest.main()
