import importlib
import os
import sys
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

    # --- New tests ---

    def test_401_response_body_is_generic(self):
        """Rejection body must be {"detail": "Unauthorized"}, nothing more."""
        with patch.dict(os.environ, {"AI_SERVICE_API_KEY": "test-secret"}):
            client = TestClient(app, raise_server_exceptions=True)
            response = client.post(
                CLASSIFY_URL,
                json=VALID_PAYLOAD,
                headers={"X-API-Key": "bad"},
            )
        self.assertEqual(response.status_code, 401)
        self.assertEqual(response.json(), {"detail": "Unauthorized"})

    def test_missing_header_response_body_is_generic(self):
        """Missing header must also return {"detail": "Unauthorized"}."""
        with patch.dict(os.environ, {"AI_SERVICE_API_KEY": "test-secret"}):
            client = TestClient(app, raise_server_exceptions=True)
            response = client.post(CLASSIFY_URL, json=VALID_PAYLOAD)
        self.assertEqual(response.status_code, 401)
        self.assertEqual(response.json(), {"detail": "Unauthorized"})

    def test_empty_string_key_env_treated_as_unconfigured(self):
        """An empty AI_SERVICE_API_KEY behaves the same as unset (dev mode)."""
        with patch.dict(os.environ, {"AI_SERVICE_API_KEY": ""}):
            client = TestClient(app, raise_server_exceptions=True)
            response = client.post(CLASSIFY_URL, json=VALID_PAYLOAD)
        self.assertEqual(response.status_code, 200)

    def test_production_startup_fails_without_api_key(self):
        """app.main must call sys.exit when APP_ENV=production and key is unset."""
        env = {"APP_ENV": "production", "AI_SERVICE_API_KEY": ""}
        with patch.dict(os.environ, env):
            os.environ.pop("AI_SERVICE_API_KEY", None)
            # Remove cached module so the module-level guard re-executes.
            sys.modules.pop("app.main", None)
            with self.assertRaises(SystemExit):
                import app.main  # noqa: F401

    def test_production_startup_succeeds_with_api_key(self):
        """app.main must not exit when APP_ENV=production and key is set."""
        env = {"APP_ENV": "production", "AI_SERVICE_API_KEY": "prod-secret"}
        with patch.dict(os.environ, env):
            sys.modules.pop("app.main", None)
            try:
                import app.main  # noqa: F401
            except SystemExit:
                self.fail("app.main raised SystemExit with a valid production key")
            finally:
                # Restore the original module so other tests are unaffected.
                sys.modules.pop("app.main", None)
                import app.main  # noqa: F401

    def test_docs_disabled_in_production(self):
        """Interactive docs endpoints must return 404 in production."""
        env = {"APP_ENV": "production", "AI_SERVICE_API_KEY": "prod-secret"}
        with patch.dict(os.environ, env):
            sys.modules.pop("app.main", None)
            import app.main as prod_main
            prod_client = TestClient(prod_main.app, raise_server_exceptions=True)
            for path in ("/docs", "/redoc", "/openapi.json"):
                resp = prod_client.get(path)
                self.assertEqual(
                    resp.status_code,
                    404,
                    msg=f"Expected {path} to be 404 in production, got {resp.status_code}",
                )
            # Restore dev module.
            sys.modules.pop("app.main", None)
            import app.main  # noqa: F401

    def test_docs_available_in_development(self):
        """Interactive docs must be accessible when APP_ENV != production."""
        with patch.dict(os.environ, {"APP_ENV": "development"}):
            sys.modules.pop("app.main", None)
            import app.main as dev_main
            dev_client = TestClient(dev_main.app, raise_server_exceptions=True)
            for path in ("/docs", "/redoc", "/openapi.json"):
                resp = dev_client.get(path)
                self.assertNotEqual(
                    resp.status_code,
                    404,
                    msg=f"Expected {path} to be accessible in development",
                )
            sys.modules.pop("app.main", None)
            import app.main  # noqa: F401


if __name__ == "__main__":
    unittest.main()
