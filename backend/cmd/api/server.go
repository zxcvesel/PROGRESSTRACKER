package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func runServer(handler http.Handler) error {
	host := os.Getenv("PROGRESS_TRACKER_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("PROGRESS_TRACKER_PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              net.JoinHostPort(host, port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignal)

	serverError := make(chan error, 1)
	go func() {
		log.Printf("Backend is running on http://%s", server.Addr)
		serverError <- server.ListenAndServe()
	}()

	select {
	case err := <-serverError:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-shutdownSignal:
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownContext)
	}
}
