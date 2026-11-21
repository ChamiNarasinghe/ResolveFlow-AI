package routes

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/chami2002/resolveflow-api/internal/auth"
	"github.com/chami2002/resolveflow-api/internal/handlers"
	"github.com/chami2002/resolveflow-api/internal/middleware"
	"github.com/chami2002/resolveflow-api/internal/repository"
	"github.com/chami2002/resolveflow-api/internal/services"
	"github.com/chami2002/resolveflow-api/internal/users"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewRouter registers all API routes. The provided ctx controls the lifecycle
// of background goroutines (rate-limiter cleanup); cancel it on shutdown.
func NewRouter(
	ctx context.Context,
	db *pgxpool.Pool,
	jwtService *auth.JWTService,
	allowRegistration bool,
	aiServiceURL string,
	log *slog.Logger,
) http.Handler {
	// Rate limiters — cleanup goroutines stop when ctx is cancelled.
	// Login: 5 req/min. Registration: 3 req/min. General API: 100 req/min.
	// Limits are per application instance (see RateLimitConfig for details).
	loginLimiter := middleware.NewRateLimiter(ctx, middleware.RateLimitConfig{RequestsPerMinute: 5})
	registerLimiter := middleware.NewRateLimiter(ctx, middleware.RateLimitConfig{RequestsPerMinute: 3})
	apiLimiter := middleware.NewRateLimiter(ctx, middleware.RateLimitConfig{RequestsPerMinute: 100})

	mux := http.NewServeMux()

	// Health endpoints — no rate limiting.
	mux.HandleFunc("GET /health", handlers.HealthHandler)
	mux.HandleFunc("GET /health/ready", handlers.ReadinessHandler(db))

	// Authentication services
	userRepo := users.NewUserRepository(db)
	authService := auth.NewAuthService(userRepo, jwtService)
	authHandler := handlers.NewAuthHandler(authService)

	// Public login endpoint — strict rate limit.
	mux.Handle("POST /api/v1/auth/login",
		loginLimiter.Limit(http.HandlerFunc(authHandler.Login)))

	// JWT authentication middleware
	authMiddleware := middleware.AuthMiddleware(jwtService)
	mux.Handle(
		"GET /api/v1/auth/me",
		apiLimiter.Limit(authMiddleware(http.HandlerFunc(authHandler.Me))),
	)

	// Registration is enabled only in development mode.
	if allowRegistration {
		mux.Handle("POST /api/v1/auth/register",
			registerLimiter.Limit(http.HandlerFunc(authHandler.Register)))
	}

	// Ticket endpoints
	ticketRepo := repository.NewTicketRepository(db)

	// AI classification dependencies
	aiClient := services.NewAIClient(aiServiceURL)
	predictionRepo := repository.NewAIPredictionRepository(db)

	// AI prediction endpoints
	aiPredictionHandler := handlers.NewAIPredictionHandler(
		ticketRepo,
		predictionRepo,
	)

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
		apiLimiter.Limit(authMiddleware(http.HandlerFunc(commentHandler.CreateComment))),
	)

	mux.Handle(
		"GET /api/v1/tickets/{id}/comments",
		apiLimiter.Limit(authMiddleware(http.HandlerFunc(commentHandler.ListComments))),
	)

	mux.Handle("POST /api/v1/tickets",
		apiLimiter.Limit(authMiddleware(http.HandlerFunc(ticketHandler.CreateTicket))))

	mux.Handle("GET /api/v1/tickets",
		apiLimiter.Limit(authMiddleware(http.HandlerFunc(ticketHandler.ListTickets))))

	mux.Handle("GET /api/v1/tickets/{id}",
		apiLimiter.Limit(authMiddleware(http.HandlerFunc(ticketHandler.GetTicket))))

	mux.Handle("PATCH /api/v1/tickets/{id}",
		apiLimiter.Limit(authMiddleware(http.HandlerFunc(ticketHandler.UpdateTicket))))

	mux.Handle(
		"GET /api/v1/tickets/{id}/history",
		apiLimiter.Limit(authMiddleware(http.HandlerFunc(ticketHandler.GetTicketHistory))),
	)

	mux.Handle(
		"GET /api/v1/tickets/{id}/ai-prediction",
		apiLimiter.Limit(authMiddleware(http.HandlerFunc(aiPredictionHandler.GetLatestPrediction))),
	)

	// Team endpoints
	teamRepo := repository.NewTeamRepository(db)
	teamHandler := handlers.NewTeamHandler(teamRepo)
	adminOnly := middleware.RequireRoles("admin")

	mux.Handle(
		"PATCH /api/v1/tickets/{id}/assignment",
		apiLimiter.Limit(authMiddleware(adminOnly(http.HandlerFunc(ticketHandler.AssignTicket)))),
	)

	mux.Handle("POST /api/v1/teams",
		apiLimiter.Limit(authMiddleware(adminOnly(http.HandlerFunc(teamHandler.CreateTeam)))))

	mux.Handle("GET /api/v1/teams",
		apiLimiter.Limit(authMiddleware(http.HandlerFunc(teamHandler.ListTeams))))

	mux.Handle("GET /api/v1/teams/{id}",
		apiLimiter.Limit(authMiddleware(http.HandlerFunc(teamHandler.GetTeam))))

	mux.Handle("PATCH /api/v1/teams/{id}",
		apiLimiter.Limit(authMiddleware(adminOnly(http.HandlerFunc(teamHandler.UpdateTeam)))))

	mux.Handle("POST /api/v1/teams/{id}/members",
		apiLimiter.Limit(authMiddleware(adminOnly(http.HandlerFunc(teamHandler.AddTeamMember)))))

	mux.Handle("GET /api/v1/teams/{id}/members",
		apiLimiter.Limit(authMiddleware(http.HandlerFunc(teamHandler.ListTeamMembers))))

	mux.Handle("DELETE /api/v1/teams/{id}/members/{userId}",
		apiLimiter.Limit(authMiddleware(adminOnly(http.HandlerFunc(teamHandler.RemoveTeamMember)))))

	// Global middleware chain: RequestID → RequestLogger → Recovery → mux.
	// RequestID runs first so every log line and panic log carries the ID.
	// RequestLogger wraps Recovery so its deferred log always executes, even
	// when Recovery catches a panic and writes the 500 response.
	return middleware.RequestID(
		middleware.RequestLogger(log)(
			middleware.Recovery(log)(mux),
		),
	)
}
