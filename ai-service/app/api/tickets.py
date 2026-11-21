
from fastapi import APIRouter, Depends

from app.dependencies.auth import verify_api_key
from app.schemas.ticket import (
    TicketClassificationRequest,
    TicketClassificationResponse,
)
from app.services.ticket_classifier import classify_ticket


router = APIRouter(prefix="/tickets", tags=["Ticket AI"])


@router.post(
    "/classify",
    response_model=TicketClassificationResponse,
    dependencies=[Depends(verify_api_key)],
)
def classify_ticket_endpoint(
    request: TicketClassificationRequest,
):
    return classify_ticket(request)
