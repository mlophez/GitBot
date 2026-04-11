package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitbot/internal"
	"gitbot/internal/adapter"
	"gitbot/internal/app"
	"gitbot/internal/config"
	"gitbot/internal/event"
	"gitbot/internal/event/provider"
	"gitbot/internal/event/queue"
	"gitbot/internal/notification"
	"gitbot/internal/server"
	"gitbot/internal/types"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	c := config.Load()

	/* Event processing */
	aService := app.NewService(c.ClientSet)
	eService := event.NewService(c.SecurityRules, aService)
	eventQueue := queue.NewMemoryQueue[types.QueueItem]()
	bitbucket := provider.NewBitbucketProvider(c.BitbucketBearerToken)
	processor := internal.NewEventProcessor(eventQueue, eService.Process, c.ClusterName)

	/* Apps API */
	appManager := adapter.NewArgoAppManager(c.ClientSet)

	/* Routes */
	router := http.NewServeMux()
	router.HandleFunc("GET /status", internal.Status)
	router.HandleFunc("POST /api/v1/webhook/bitbucket", internal.EventCreate(eventQueue, bitbucket))
	router.HandleFunc("POST /api/v1/notification", notification.HandleNotification(c.ClientSet, bitbucket))
	router.HandleFunc("GET /api/v1/apps", internal.ListApps(appManager))
	router.HandleFunc("POST /api/v1/apps/{id}/lock", internal.LockApp(appManager))
	router.HandleFunc("POST /api/v1/apps/{id}/unlock", internal.UnlockApp(appManager))

	/* HTTP server */
	srv := &http.Server{Addr: ":" + c.HttpPort, Handler: server.RequestID(router)}
	go func() {
		slog.Info("Starting server in port :" + c.HttpPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Error starting server", "error", err)
			os.Exit(1)
		}
	}()

	go processor.Start()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	slog.Info("Server shutdown...")
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
	}

	slog.Info("Shutdown event queue...")
	processor.Stop(ctx)

	slog.Info("Server stopped")
	time.Sleep(3 * time.Second)
	os.Exit(0)
}
