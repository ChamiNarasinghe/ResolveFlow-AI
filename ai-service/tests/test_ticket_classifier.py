
import unittest

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

    def test_negative_sentiment(self):
        request = TicketClassificationRequest(
            subject="Very disappointed",
            description="I am frustrated with this terrible service.",
        )

        result = classify_ticket(request)

        self.assertEqual(result.sentiment, "negative")

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

        self.assertGreaterEqual(result.confidence, 0.0)
        self.assertLessEqual(result.confidence, 1.0)


if __name__ == "__main__":
    unittest.main()
