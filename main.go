package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
)

var pool *pgxpool.Pool

func main() {
	ctx := context.Background()

	var err error
	pool, err = pgxpool.New(ctx, os.Getenv("DB_URL"))
	if err != nil {
		log.Fatalf("ERROR: failed to create pool: %s", err)
	}
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ERROR: error while pinging database: %s", err)
	}
	log.Println("Database connected successfully")

	r := chi.NewRouter()

	r.Use(PanicRecovery)
	r.Use(CORSMiddleware())
	r.Use(middleware.Logger)
	r.Use(FingerprintMiddleware)
	r.Use(RateLimiter(60))

	r.Get("/health", healthHandler)

	r.Post("/auth/register", registerHandler)
	r.Post("/auth/login", loginHandler)
	r.Post("/auth/refresh", refreshHandler)
	r.Delete("/auth/logout", logoutHandler)

	r.Get("/boards", listBoardsHandler)
	r.Get("/boards/{id}", getBoardByIDHandler)
	r.Get("/boards/{id}/feedbacks", listFeedbacksHandler)
	r.Get("/boards/{id}/feedbacks/{feedbackId}", getFeedbackByIDHandler)

	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware)

		r.Post("/boards", createBoardHandler)
		r.Put("/boards/{id}", updateBoardHandler)
		r.Delete("/boards/{id}", deleteBoardHandler)

		r.Post("/boards/{id}/feedbacks", createFeedbackHandler)
		r.Patch("/boards/{id}/feedbacks/{feedbackId}", updateFeedbackStatusHandler)
		r.Delete("/boards/{id}/feedbacks/{feedbackId}", deleteFeedbackHandler)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "6767"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("Server started on port :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ERROR: server failed: %s", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("ERROR: server forced to shutdown: %s", err)
	}

	pool.Close()
	log.Println("Server stopped")
}
