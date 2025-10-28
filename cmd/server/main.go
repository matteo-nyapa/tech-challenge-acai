package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/chat"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/chat/assistant"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/chat/model"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/httpx"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/mongox"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/observability"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/pb"
	"github.com/twitchtv/twirp"
)

func main() {

	ctx := context.Background()
	shutdown, err := observability.SetupOTel(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to setup OpenTelemetry: %w", err))
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			slog.Error("OpenTelemetry shutdown error", "err", err)
		}
	}()

	hooks, _, err := observability.NewServerMetrics()
	if err != nil {
		panic(fmt.Errorf("failed to init server metrics: %w", err))
	}

	mongo := mongox.MustConnect()
	repo := model.New(mongo)
	assist := assistant.New()
	server := chat.NewServer(repo, assist)

	handler := mux.NewRouter()
	handler.Use(
		httpx.Logger(),
		httpx.Recovery(),
	)

	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "Hi, my name is Clippy!")
	})

	twirpSrv := pb.NewChatServiceServer(
		server,
		twirp.WithServerJSONSkipDefaults(true),
		twirp.WithServerHooks(hooks),
	)
	handler.PathPrefix("/twirp/").Handler(twirpSrv)

	slog.Info("Starting the server...", "addr", ":8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		panic(err)
	}
}
