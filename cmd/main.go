package main

import (
	"context"
	"encoding/json"
	"errors"
	"gitbot/internal/config"
	"gitbot/internal/event"

	"gitbot/internal/event/provider"

	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	kubeconfig = "/home/mlr/Documents/Code/gitbot/kubeconfig"
)

func NewKubernetes() *kubernetes.Clientset {
	config, err := func() (*rest.Config, error) {
		_, exists := os.LookupEnv("KUBERNETES_SERVICE_HOST")
		if exists {
			return rest.InClusterConfig()
		} else {
			return clientcmd.BuildConfigFromFlags("", kubeconfig)
		}
	}()
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}

func main() {
	// Configure log
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// Configure kubernetes
	clientset := NewKubernetes()

	// Configure webhook queue
	queue := event.NewMemoryQueue()

	// New Router for Http Server
	router := http.NewServeMux()
	router.HandleFunc("GET /status", status)

	// Setup Git Providers Routes
	bitbucket := provider.NewBitbucketProvider(config.Get("BITBUCKET_BEARER_TOKEN"))
	router.HandleFunc("POST /api/v1/webhook/bitbucket", event.Handle(queue, bitbucket))

	// Starting Http Server
	srv := &http.Server{Addr: ":" + config.Get("HTTP_PORT"), Handler: router}
	go func() {
		slog.Info("Starting server in port :" + config.Get("HTTP_PORT"))
		err := srv.ListenAndServe()

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Error starting server: %s", err)
			os.Exit(1)
		}
	}()

	// Start Event Queue Worker
	StopWorker := event.StartWorker(queue, clientset)

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
	StopWorker(ctx)

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
