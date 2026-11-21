package routes

import (
	"net/http"

	"github.com/chami2002/resolveflow-api/internal/auth"
	"github.com/chami2002/resolveflow-api/internal/handlers"
	"github.com/chami2002/resolveflow-api/internal/middleware"
	"github.com/chami2002/resolveflow-api/internal/repository"
	"github.com/chami2002/resolveflow-api/internal/services"
	"github.com/chami2002/resolveflow-api/internal/users"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewRouter registers all API routes.
func NewRouter(
	db *pgxpool.Pool,
	jwtService *auth.JWTService,
	allowRegistration bool,
	aiServiceURL string,
) *http.ServeMux {
	mux := http.NewServeMux()

	// Health endpoints
	mux.HandleFunc("GET /health", handlers.HealthHandler)
	mux.HandleFunc("GET /health/ready", handlers.ReadinessHandler(db))

	// Authentication services
	userRepo := users.NewUserRepository(db)
	authService := auth.NewAuthService(userRepo, jwtService)
	authHandler := handlers.NewAuthHandler(authService)

	// Public login endpoint
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	// JWT authentication middleware
	authMiddleware := middleware.AuthMiddleware(jwtService)
	mux.Handle(
		"GET /api/v1/auth/me",
		authMiddleware(http.HandlerFunc(authHandler.Me)),
	)

	// Registration is enabled only in development mode.
	if allowRegistration {
		mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	}

	// Ticket endpoints
	ticketRepo := repository.NewTicketRepository(db)

	// AI classification dependencies
	aiClient := services.NewAIClient(aiServiceURL)
	predictionRepo := repository.NewAIPredictionRepository(db)

	ticketHandler := handlers.NewTicketHandlerWithAI(
		ticketRepo,
		aiClient,
		predictionRepo,
	)

	// Comment endpoints
	commentRepo := repository.NewCommentRepository(db)

	commentHandler := handlers.NewCommentHandler(
		commentRepo,
		ticketRepo,
	)

	mux.Handle(
		"POST /api/v1/tickets/{id}/comments",
		authMiddleware(http.HandlerFunc(commentHandler.CreateComment)),
	)

	mux.Handle(
		"GET /api/v1/tickets/{id}/comments",
		authMiddleware(http.HandlerFunc(commentHandler.ListComments)),
	)

	mux.Handle("POST /api/v1/tickets",
		authMiddleware(http.HandlerFunc(ticketHandler.CreateTicket)))

	mux.Handle("GET /api/v1/tickets",
		authMiddleware(http.HandlerFunc(ticketHandler.ListTickets)))

	mux.Handle("GET /api/v1/tickets/{id}",
		authMiddleware(http.HandlerFunc(ticketHandler.GetTicket)))

	mux.Handle("PATCH /api/v1/tickets/{id}",
		authMiddleware(http.HandlerFunc(ticketHandler.UpdateTicket)))

	return mux
}
