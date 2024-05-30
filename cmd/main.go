package main

import (
	"context"
	"encoding/json"
	"errors"
	"gitbot/internal/app"
	"gitbot/internal/config"
	"gitbot/internal/event"
	"gitbot/internal/event/provider"
	"gitbot/internal/event/queue"

	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
}

func main() {
	// Configure log
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// Load config
	c := config.Load()

	/* Events */
	// Service
	aService := app.NewService(c.ClientSet)
	queue := queue.NewMemoryQueue[event.QueueItem]()
	eService := event.NewService(c.SecurityRules, aService)
	// Worker
	worker := event.NewWorker(queue, eService, c.ClusterName)
	// Handlers
	bitbucket := provider.NewBitbucketProvider(c.BitbucketBearerToken)
	bitbucketHandler := event.NewHandler(queue, bitbucket)

	/* Routes */
	router := http.NewServeMux()
	router.HandleFunc("GET /status", status)
	router.HandleFunc("POST /api/v1/webhook/bitbucket", bitbucketHandler.Handle())

	// Starting Http Server
	srv := &http.Server{Addr: ":" + c.HttpPort, Handler: router}
	go func() {
		slog.Info("Starting server in port :" + c.HttpPort)
		err := srv.ListenAndServe()

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Error starting server: %s", err)
			os.Exit(1)
		}
	}()

	// Start Event Queue Worker
	go worker.Start()

	// Wait for shutdown signal
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done

	// Stopping gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop http server
	slog.Info("Server shutdown...")
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server Shutdown Failed:%+v", err)
	}

	// Stop event queue worker
	slog.Info("Shutdown event queue...")
	worker.Stop(ctx)

	slog.Info("Server Stopped...")
	time.Sleep(3 * time.Second)
	os.Exit(0)
}

func status(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{"status": "OK"}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(jsonResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
