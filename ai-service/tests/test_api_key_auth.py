import os
import unittest
from unittest.mock import patch

from fastapi.testclient import TestClient

from app.main import app

VALID_PAYLOAD = {
    "subject": "Payment failed",
    "description": "My payment failed twice.",
}

CLASSIFY_URL = "/api/v1/tickets/classify"


class TestAPIKeyAuth(unittest.TestCase):

    def test_no_key_configured_allows_request(self):
        with patch.dict(os.environ, {}, clear=False):
            os.environ.pop("AI_SERVICE_API_KEY", None)
            client = TestClient(app, raise_server_exceptions=True)
            response = client.post(CLASSIFY_URL, json=VALID_PAYLOAD)
        self.assertEqual(response.status_code, 200)

    def test_correct_key_allows_request(self):
        with patch.dict(os.environ, {"AI_SERVICE_API_KEY": "test-secret"}):
            client = TestClient(app, raise_server_exceptions=True)
            response = client.post(
                CLASSIFY_URL,
                json=VALID_PAYLOAD,
                headers={"X-API-Key": "test-secret"},
            )
        self.assertEqual(response.status_code, 200)

    def test_wrong_key_returns_401(self):
        with patch.dict(os.environ, {"AI_SERVICE_API_KEY": "test-secret"}):
            client = TestClient(app, raise_server_exceptions=True)
            response = client.post(
                CLASSIFY_URL,
                json=VALID_PAYLOAD,
                headers={"X-API-Key": "wrong-key"},
            )
        self.assertEqual(response.status_code, 401)

    def test_missing_header_with_key_configured_returns_401(self):
        with patch.dict(os.environ, {"AI_SERVICE_API_KEY": "test-secret"}):
            client = TestClient(app, raise_server_exceptions=True)
            response = client.post(CLASSIFY_URL, json=VALID_PAYLOAD)
        self.assertEqual(response.status_code, 401)

    def test_error_response_does_not_expose_key(self):
        secret = "super-secret-key"
        with patch.dict(os.environ, {"AI_SERVICE_API_KEY": secret}):
            client = TestClient(app, raise_server_exceptions=True)
            response = client.post(
                CLASSIFY_URL,
                json=VALID_PAYLOAD,
                headers={"X-API-Key": "wrong"},
            )
        self.assertEqual(response.status_code, 401)
        self.assertNotIn(secret, response.text)

    def test_health_endpoint_requires_no_key(self):
        with patch.dict(os.environ, {"AI_SERVICE_API_KEY": "test-secret"}):
            client = TestClient(app, raise_server_exceptions=True)
            response = client.get("/health")
        self.assertEqual(response.status_code, 200)


if __name__ == "__main__":
    unittest.main()
