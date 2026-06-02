// Package main is the entrypoint for the kubeops-agent HTTP server.
// It wires all dependencies (config, adapters, queue, routes) and starts
// the HTTP server and event-processing loop.
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

	"gitbot/internal/app"
	"gitbot/internal/config"
	"gitbot/internal/event"
	"gitbot/internal/status"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	c := config.NewEnvConfigLoader().Load()

	/* Apps API — local cluster is always managed directly.
	   If the config defines remote agent clusters, a multi-cluster manager is built
	   so that events are processed across all clusters and responses are centralised. */
	localManager := app.NewArgoAppManager(c.ClientSet, c.ClusterName)
	appManager := buildAppManager(c, localManager)

	/* Providers: Bitbucket, GitHub, GitLab, etc. */
	bitbucket := event.NewBitbucketClient(c.BitbucketBearerToken, c.BitbucketBotUUID)

	/* Queue */
	eventQueue := event.NewMemoryQueue[event.QueueItem]()

	/* Routes */
	protected := func(h http.Handler) http.Handler { return apiTokenAuth(c.APIToken, h) }

	router := http.NewServeMux()
	router.HandleFunc("GET /{$}", serveIndex)
	router.HandleFunc("GET /api/v1/status", status.Status)
	router.HandleFunc("POST /api/v1/webhook/bitbucket", event.EventCreate(eventQueue, bitbucket, c.WebhookToken))
	router.Handle("POST /api/v1/notification", protected(event.NotificationHandle(appManager, bitbucket)))
	router.Handle("GET /api/v1/apps", protected(app.ListApps(appManager)))
	router.Handle("POST /api/v1/apps/{id}/lock", protected(app.LockApp(appManager)))
	router.Handle("POST /api/v1/apps/{id}/unlock", protected(app.UnlockApp(appManager)))
	router.HandleFunc("POST /api/v1/admission/apps/validate", app.ValidateApp(c.BotKubernetesUsername))

	/* HTTP server */
	var handler http.Handler = requestID(router)
	if c.ContextRoot != "" {
		handler = http.StripPrefix(c.ContextRoot, handler)
	}
	srv := &http.Server{Addr: ":" + c.HttpPort, Handler: handler}
	go func() {
		slog.Info("Starting server", "port", c.HttpPort, "contextRoot", c.ContextRoot)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Error starting server", "error", err)
			os.Exit(1)
		}
	}()

	/* Event processing */
	processor := newEventProcessor(eventQueue, event.EventProcess(appManager), c.ClusterName)
	go processor.start()

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
	processor.stop(ctx)

	slog.Info("Server stopped")
	time.Sleep(3 * time.Second)
	os.Exit(0)
}

// buildAppManager returns a MultiClusterAppManager when the config defines remote
// agent clusters, or the local manager directly when there are none.
// Single-cluster deployments are unaffected by this change.
func buildAppManager(c *config.Config, local app.AppManager) app.AppManager {
	var remotes []config.ClusterConfig
	for _, cl := range c.Clusters {
		if cl.Auth.Type == "agent" {
			remotes = append(remotes, cl)
		}
	}
	if len(remotes) == 0 {
		return local
	}

	multi := app.NewMultiClusterAppManager()

	// Register the local cluster. Prefer the name from config; fall back to CLUSTER_NAME.
	localName := c.ClusterName
	for _, cl := range c.Clusters {
		if cl.Auth.Type == "serviceaccount" {
			localName = cl.Name
			break
		}
	}
	multi.Add(localName, local)

	// Register each remote agent cluster.
	for _, cl := range remotes {
		multi.Add(cl.Name, app.NewRemoteAppManager(cl.Auth.URL, cl.Name, cl.Auth.InsecureSkipTLSVerify, c.APIToken))
	}

	return multi
}
