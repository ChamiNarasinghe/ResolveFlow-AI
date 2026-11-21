
from fastapi import APIRouter

from app.schemas.ticket import (
    TicketClassificationRequest,
    TicketClassificationResponse,
)
from app.services.ticket_classifier import classify_ticket


router = APIRouter(prefix="/tickets", tags=["Ticket AI"])


@router.post(
    "/classify",
    response_model=TicketClassificationResponse,
)
def classify_ticket_endpoint(
    request: TicketClassificationRequest,
):
    return classify_ticket(request)
