package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/fx"

	"github.com/jycamier/retrotro/backend/internal/config"
	"github.com/jycamier/retrotro/backend/internal/middleware"
)

var RouterModule = fx.Module("router",
	fx.Provide(NewRouter),
)

var ServerModule = fx.Module("server",
	fx.Invoke(StartServer),
)

// NewRouter creates and configures the chi router
func NewRouter(
	cfg *config.Config,
	authHandler *AuthHandler,
	teamHandler *TeamHandler,
	retroHandler *RetrospectiveHandler,
	wsHandler *WebSocketHandler,
	statsHandler *StatsHandler,
	adminHandler *AdminHandler,
	webhookHandler *WebhookHandler,
) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.SlogLogger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Auth routes (public)
	r.Route("/auth", func(r chi.Router) {
		r.Get("/info", authHandler.GetLoginInfo)
		r.Get("/login", authHandler.Login)
		r.Get("/callback", authHandler.Callback)
		r.Post("/logout", authHandler.Logout)
		r.Post("/refresh", authHandler.RefreshToken)
		r.Post("/dev-login", authHandler.DevLogin)
		r.Get("/dev-users", authHandler.GetDevUsers)
	})

	// API routes (protected)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.JWTAuth(cfg.JWT.Secret))

		r.Get("/me", authHandler.GetCurrentUser)

		// Admin routes
		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.RequireAdmin)
			r.Get("/users", adminHandler.ListUsers)
			r.Get("/teams", adminHandler.ListTeams)
			r.Get("/teams/{teamId}/members", adminHandler.GetTeamMembers)
		})

		// Teams
		r.Route("/teams", func(r chi.Router) {
			r.Get("/", teamHandler.List)
			r.Post("/", teamHandler.Create)
			r.Route("/{teamId}", func(r chi.Router) {
				r.Get("/", teamHandler.Get)
				r.Put("/", teamHandler.Update)
				r.Delete("/", teamHandler.Delete)
				r.Get("/members", teamHandler.ListMembers)
				r.Post("/members", teamHandler.AddMember)
				r.Delete("/members/{userId}", teamHandler.RemoveMember)
				r.Put("/members/{userId}/role", teamHandler.UpdateMemberRole)

				r.Route("/stats", func(r chi.Router) {
					r.Get("/roti", statsHandler.GetTeamRotiStats)
					r.Get("/mood", statsHandler.GetTeamMoodStats)
					r.Get("/me", statsHandler.GetMyStats)
					r.Get("/users/{userId}/roti", statsHandler.GetUserRotiStats)
					r.Get("/users/{userId}/mood", statsHandler.GetUserMoodStats)
				})

				// Team actions from completed retrospectives
				r.Get("/actions", retroHandler.ListTeamActions)
				r.Patch("/actions/{actionId}", retroHandler.PatchTeamAction)

				// Team topics from completed Lean Coffee sessions
				r.Get("/topics", retroHandler.ListTeamTopics)
				r.Post("/topics/analyze", retroHandler.AnalyzeTeamTopics)

				// Webhooks
				r.Route("/webhooks", func(r chi.Router) {
					r.Post("/", webhookHandler.Create)
					r.Get("/", webhookHandler.List)
					r.Route("/{webhookId}", func(r chi.Router) {
						r.Get("/", webhookHandler.Get)
						r.Put("/", webhookHandler.Update)
						r.Delete("/", webhookHandler.Delete)
						r.Get("/deliveries", webhookHandler.ListDeliveries)
					})
				})
			})
		})

		// Templates
		r.Route("/templates", func(r chi.Router) {
			r.Get("/", retroHandler.ListTemplates)
			r.Post("/", retroHandler.CreateTemplate)
			r.Get("/{templateId}", retroHandler.GetTemplate)
		})

		// Retrospectives
		r.Route("/retrospectives", func(r chi.Router) {
			r.Get("/", retroHandler.List)
			r.Post("/", retroHandler.Create)
			r.Route("/{retroId}", func(r chi.Router) {
				r.Get("/", retroHandler.Get)
				r.Put("/", retroHandler.Update)
				r.Delete("/", retroHandler.Delete)
				r.Post("/start", retroHandler.Start)
				r.Post("/end", retroHandler.End)

				r.Route("/items", func(r chi.Router) {
					r.Get("/", retroHandler.ListItems)
					r.Post("/", retroHandler.CreateItem)
					r.Put("/{itemId}", retroHandler.UpdateItem)
					r.Delete("/{itemId}", retroHandler.DeleteItem)
					r.Post("/{itemId}/group", retroHandler.GroupItems)
				})

				r.Post("/items/{itemId}/vote", retroHandler.Vote)
				r.Delete("/items/{itemId}/vote", retroHandler.Unvote)

				r.Route("/actions", func(r chi.Router) {
					r.Get("/", retroHandler.ListActions)
					r.Post("/", retroHandler.CreateAction)
					r.Put("/{actionId}", retroHandler.UpdateAction)
					r.Delete("/{actionId}", retroHandler.DeleteAction)
				})

				r.Route("/timer", func(r chi.Router) {
					r.Post("/start", retroHandler.StartTimer)
					r.Post("/pause", retroHandler.PauseTimer)
					r.Post("/resume", retroHandler.ResumeTimer)
					r.Post("/reset", retroHandler.ResetTimer)
					r.Post("/add-time", retroHandler.AddTime)
				})

				r.Post("/phase/next", retroHandler.NextPhase)
				r.Post("/phase/set", retroHandler.SetPhase)

				r.Get("/roti", retroHandler.GetRotiResults)
				r.Get("/icebreaker", retroHandler.GetIcebreakerMoods)
			})
		})
	})

	// WebSocket endpoint
	r.Get("/ws", wsHandler.HandleConnection)

	return r
}

// StartServer starts the HTTP server with lifecycle management
func StartServer(lc fx.Lifecycle, cfg *config.Config, router *chi.Mux) {
	srv := &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				slog.Info("server starting", "port", cfg.Port)
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					slog.Error("server failed", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			slog.Info("shutting down server...")
			return srv.Shutdown(ctx)
		},
	})
}
