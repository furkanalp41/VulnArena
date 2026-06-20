package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

func New(router *chi.Mux, port string, logger *slog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         ":" + port,
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger: logger,
	}
}

func (s *Server) Start() error {
	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// shutdownDone is closed only after Shutdown(ctx) has fully drained
	// in-flight requests, so Start() can block on it before returning and
	// callers do not tear down DB/Redis pools mid-drain.
	shutdownDone := make(chan struct{})

	go func() {
		defer close(shutdownDone)

		sig := <-quit
		s.logger.Info("shutting down server", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.logger.Error("server forced shutdown", slog.String("error", err.Error()))
		}
	}()

	s.logger.Info("starting server", slog.String("addr", s.httpServer.Addr))

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	// ListenAndServe returned ErrServerClosed because Shutdown was called.
	// Wait for the in-flight drain to complete before returning so the
	// caller does not close shared resources while requests are finishing.
	<-shutdownDone

	s.logger.Info("server stopped")
	return nil
}
