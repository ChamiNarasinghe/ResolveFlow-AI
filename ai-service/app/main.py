import os
import sys

from fastapi import FastAPI
from fastapi.responses import JSONResponse

from app.api.tickets import router as tickets_router
from app.schemas.ticket import TicketClassificationRequest
import app.services.ticket_classifier as _classifier_module

# Minimal fixed probe used by /health/ready — cheap, no side-effects.
_PROBE_REQUEST = TicketClassificationRequest(
    subject="readiness probe",
    description="internal readiness check",
)

_env = os.environ.get("APP_ENV", "development")
_is_production = _env == "production"

# Production guard: refuse to start without a configured API key so that
# the classify endpoint is never silently left open in production.
if _is_production and not os.environ.get("AI_SERVICE_API_KEY", ""):
    sys.exit(
        "FATAL: AI_SERVICE_API_KEY must be set when APP_ENV=production. "
        "Refusing to start."
    )

# Disable interactive API docs in production to avoid exposing internal
# endpoint details and schema to the public internet.
app = FastAPI(
    title="ResolveFlow AI Service",
    description="AI-powered ticket intelligence for ResolveFlow",
    version="0.1.0",
    docs_url=None if _is_production else "/docs",
    redoc_url=None if _is_production else "/redoc",
    openapi_url=None if _is_production else "/openapi.json",
)


@app.get("/health")
def health_check():
    return {
        "status": "healthy",
        "service": "resolveflow-ai-service",
    }


@app.get("/health/ready")
def readiness_check():
    """Readiness probe: verifies the classifier can be called successfully.

    Uses a pre-built minimal request so no expensive work is repeated.
    Internal exception details are never exposed in the response.
    """
    try:
        _classifier_module.classify_ticket(_PROBE_REQUEST)
    except Exception:
        return JSONResponse(
            status_code=503,
            content={"status": "not_ready", "classifier": "unavailable"},
        )
    return {"status": "ready", "classifier": "ok"}


app.include_router(
    tickets_router,
    prefix="/api/v1",
)
