// Package входная точка
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vandermeer0/pr-reviewer/internal/config"
	"github.com/vandermeer0/pr-reviewer/internal/infrastructure/repository/postgresql"
	"github.com/vandermeer0/pr-reviewer/internal/transport/httpapi"
	"github.com/vandermeer0/pr-reviewer/internal/usecase"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := postgresql.NewPool(ctx, cfg.DB.ConnString())
	if err != nil {
		log.Fatalf("failed to create postgres pool: %v", err)
	}
	defer pool.Close()

	userRepo := postgresql.NewUserRepository(pool)
	teamRepo := postgresql.NewTeamRepository(pool)
	prRepo := postgresql.NewPullRequestRepository(pool)

	teamSvc := usecase.NewTeamService(userRepo, teamRepo)
	userSvc := usecase.NewUserService(userRepo)
	prSvc := usecase.NewPullRequestService(prRepo, userRepo, teamRepo)
	statsSvc := usecase.NewStatsService(pool)
	teamMaintSvc := usecase.NewTeamMaintenanceService(pool)

	apiServer := httpapi.NewServer(teamSvc, userSvc, prSvc, statsSvc, teamMaintSvc)
	mux := http.NewServeMux()
	apiServer.RegisterRoutes(mux)

	addr := ":8080"
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("pr-reviewer HTTP server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server error: %v", err)
		}
	}()

	log.Println("pr-reviewer app initialized")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
}
