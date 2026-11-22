import unittest
from unittest.mock import patch

from fastapi.testclient import TestClient

from app.main import app

READY_URL = "/health/ready"

# Patch target: the function on the module object that app.main accesses
# via `_classifier_module.classify_ticket` at request time.
_CLASSIFIER_PATH = "app.services.ticket_classifier.classify_ticket"


class TestHealthReady(unittest.TestCase):

    def test_ready_returns_200(self):
        client = TestClient(app, raise_server_exceptions=True)
        response = client.get(READY_URL)
        self.assertEqual(response.status_code, 200)

    def test_ready_response_body(self):
        client = TestClient(app, raise_server_exceptions=True)
        response = client.get(READY_URL)
        self.assertEqual(response.json(), {"status": "ready", "classifier": "ok"})

    def test_ready_requires_no_api_key(self):
        """Readiness endpoint must be reachable without authentication."""
        with patch.dict("os.environ", {"AI_SERVICE_API_KEY": "secret"}):
            client = TestClient(app, raise_server_exceptions=True)
            response = client.get(READY_URL)
        self.assertEqual(response.status_code, 200)

    def test_classifier_failure_returns_503(self):
        """When classify_ticket raises, /health/ready must return 503."""
        with patch(_CLASSIFIER_PATH, side_effect=RuntimeError("classifier exploded")):
            client = TestClient(app, raise_server_exceptions=False)
            response = client.get(READY_URL)
        self.assertEqual(response.status_code, 503)

    def test_classifier_failure_response_body(self):
        """503 body must be safe — no internal exception details."""
        with patch(
            _CLASSIFIER_PATH,
            side_effect=RuntimeError("internal detail that must not leak"),
        ):
            client = TestClient(app, raise_server_exceptions=False)
            response = client.get(READY_URL)
        body = response.json()
        self.assertEqual(body, {"status": "not_ready", "classifier": "unavailable"})
        self.assertNotIn("internal detail that must not leak", response.text)

    def test_liveness_still_returns_200(self):
        """/health liveness probe is unaffected by this change."""
        client = TestClient(app, raise_server_exceptions=True)
        response = client.get("/health")
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json()["status"], "healthy")


if __name__ == "__main__":
    unittest.main()
