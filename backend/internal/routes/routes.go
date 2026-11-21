package routes

import (
	"net/http"

	"github.com/chami2002/resolveflow-api/internal/handlers"
	"github.com/chami2002/resolveflow-api/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewRouter registers all API routes.
func NewRouter(db *pgxpool.Pool) *http.ServeMux {
	mux := http.NewServeMux()

	// Health endpoints
	mux.HandleFunc("GET /health", handlers.HealthHandler)
	mux.HandleFunc("GET /health/ready", handlers.ReadinessHandler(db))

	// Ticket endpoints
	ticketRepo := repository.NewTicketRepository(db)
	ticketHandler := handlers.NewTicketHandler(ticketRepo)

	mux.HandleFunc("POST /api/v1/tickets", ticketHandler.CreateTicket)
	mux.HandleFunc("GET /api/v1/tickets", ticketHandler.ListTickets)
	mux.HandleFunc("GET /api/v1/tickets/{id}", ticketHandler.GetTicket)
	mux.HandleFunc("PATCH /api/v1/tickets/{id}", ticketHandler.UpdateTicket)

	return mux
}
