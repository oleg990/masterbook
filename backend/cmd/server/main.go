package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"masterbook/internal/appointments"
	"masterbook/internal/auth"
	"masterbook/internal/config"
	"masterbook/internal/database"
	"masterbook/internal/masters"
	"masterbook/internal/notifications"
	"masterbook/internal/schedule"
	"masterbook/internal/services"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	authHandler := &auth.Handler{
		DB:        db,
		JWTSecret: cfg.JWTSecret,
	}

	mastersHandler := &masters.Handler{
		DB: db,
	}

	servicesHandler := &services.Handler{
		DB: db,
	}

	scheduleHandler := &schedule.Handler{
		DB: db,
	}

	appointmentsHandler := &appointments.Handler{
		DB: db,
	}

	notificationsHandler := &notifications.Handler{
		DB: db,
	}

	mux := http.NewServeMux()

	authMiddleware := auth.AuthMiddleware(cfg.JWTSecret)

	mux.HandleFunc(
		"POST /api/v1/auth/register",
		authHandler.Register,
	)

	mux.HandleFunc(
		"POST /api/v1/auth/login",
		authHandler.Login,
	)

	mux.Handle(
		"GET /api/v1/me",
		authMiddleware(
			http.HandlerFunc(authHandler.Me),
		),
	)

	mux.Handle(
		"POST /api/v1/master/profile",
		authMiddleware(
			http.HandlerFunc(mastersHandler.CreateProfile),
		),
	)

	mux.Handle(
		"GET /api/v1/master/profile",
		authMiddleware(
			http.HandlerFunc(mastersHandler.GetProfile),
		),
	)

	mux.Handle(
		"POST /api/v1/master/services",
		authMiddleware(
			http.HandlerFunc(servicesHandler.Create),
		),
	)

	mux.Handle(
		"GET /api/v1/masters/{masterID}/services",
		http.HandlerFunc(servicesHandler.List),
	)

	// Health check
	mux.HandleFunc(
		"/api/health",
		healthHandler,
	)

	mux.Handle(
		"PUT /api/v1/master/schedule",
		authMiddleware(
			http.HandlerFunc(scheduleHandler.SetWorkingHour),
		),
	)

	mux.Handle(
		"GET /api/v1/masters/{masterID}/schedule",
		http.HandlerFunc(scheduleHandler.GetWorkingHours),
	)

	mux.Handle(
		"POST /api/v1/appointments",
		authMiddleware(
			http.HandlerFunc(appointmentsHandler.Create),
		),
	)

	mux.Handle(
		"GET /api/v1/appointments",
		authMiddleware(
			http.HandlerFunc(appointmentsHandler.ListMine),
		),
	)

	mux.Handle(
		"PATCH /api/v1/appointments/{id}/cancel",
		authMiddleware(
			http.HandlerFunc(appointmentsHandler.Cancel),
		),
	)

	mux.Handle(
		"GET /api/v1/masters/{masterID}/availability",
		http.HandlerFunc(appointmentsHandler.Availability),
	)

	mux.Handle(
		"GET /api/v1/master/appointments",
		authMiddleware(
			http.HandlerFunc(appointmentsHandler.ListMasterAppointments),
		),
	)

	mux.Handle(
		"PATCH /api/v1/master/appointments/{id}/confirm",
		authMiddleware(
			http.HandlerFunc(appointmentsHandler.Confirm),
		),
	)

	mux.Handle(
		"PATCH /api/v1/master/appointments/{id}/complete",
		authMiddleware(
			http.HandlerFunc(appointmentsHandler.Complete),
		),
	)

	mux.Handle(
		"PATCH /api/v1/master/appointments/{id}/cancel",
		authMiddleware(
			http.HandlerFunc(appointmentsHandler.CancelByMaster),
		),
	)

	mux.Handle(
		"GET /api/v1/notifications",
		authMiddleware(
			http.HandlerFunc(notificationsHandler.List),
		),
	)

	mux.Handle(
		"PATCH /api/v1/notifications/{id}/read",
		authMiddleware(
			http.HandlerFunc(notificationsHandler.MarkAsRead),
		),
	)

	// Readiness check
	mux.HandleFunc("/api/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status":   "not ready",
				"database": "unavailable",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status":   "ready",
			"database": "ok",
		})
	})

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("MasterBook server started on :%s", cfg.HTTPPort)

		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()

	log.Println("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	log.Println("server stopped")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}
