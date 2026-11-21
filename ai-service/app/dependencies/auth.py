import os

from fastapi import Header, HTTPException, status


def verify_api_key(x_api_key: str = Header(default="")) -> None:
    """Validate the X-API-Key header against AI_SERVICE_API_KEY.

    Enforcement is skipped when the environment variable is not set,
    allowing the service to run without authentication in development.
    The configured key is never included in error responses.
    """
    configured_key = os.environ.get("AI_SERVICE_API_KEY", "")
    if not configured_key:
        return
    if x_api_key != configured_key:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED)
