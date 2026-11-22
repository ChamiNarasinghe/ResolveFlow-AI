import os

from fastapi import Header, HTTPException, status

# Sentinel response used for every auth failure — never echoes the received
# or configured key.
_UNAUTHORIZED = HTTPException(
    status_code=status.HTTP_401_UNAUTHORIZED,
    detail="Unauthorized",
)


def verify_api_key(x_api_key: str = Header(default="")) -> None:
    """Validate the X-API-Key header against AI_SERVICE_API_KEY.

    When the key is configured, any request that omits or supplies an
    incorrect key is rejected with HTTP 401.  The configured key and the
    received value are never included in error responses or logs.

    When the key is not configured the service is running in development
    mode (APP_ENV != production) and authentication is skipped.  Production
    startup is guarded in app.main so this branch is never reached in
    production.
    """
    configured_key = os.environ.get("AI_SERVICE_API_KEY", "")
    if not configured_key:
        # Development mode — no key configured, allow all requests.
        return
    if x_api_key != configured_key:
        raise _UNAUTHORIZED
