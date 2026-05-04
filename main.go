package main

import (
	"context"
	"log"
	"net/http"
	"os"

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
	r.Use(middleware.Logger)

	r.Get("/health", healthHandler)

	r.Get("/boards", listBoardsHandler)
	r.Post("/boards", createBoardHandler)
	r.Get("/boards/{id}", getBoardByIDHandler)
	r.Put("/boards/{id}", updateBoardHandler)
	r.Delete("/boards/{id}", deleteBoardHandler)

	r.Get("/boards/{id}/feedbacks", listFeedbacksHandler)
	r.Post("/boards/{id}/feedbacks", createFeedbackHandler)
	r.Get("/boards/{id}/feedbacks/{feedbackId}", getFeedbackByIDHandler)
	r.Patch("/boards/{id}/feedbacks/{feedbackId}", updateFeedbackStatusHandler)
	r.Delete("/boards/{id}/feedbacks/{feedbackId}", deleteFeedbackHandler)

	log.Println("Server started in port :6767")
	log.Fatal(http.ListenAndServe(":6767", r))
}
