package main

import (
	"context"
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
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

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

	authMiddleware := auth.AuthMiddleware(cfg.JWTSecret)

	mux := http.NewServeMux()

	// -------------------------
	// Auth
	// -------------------------

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

	// -------------------------
	// Masters
	// -------------------------

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

	// Публичный список мастеров
	mux.Handle(
		"GET /api/v1/masters",
		http.HandlerFunc(mastersHandler.List),
	)

	// -------------------------
	// Services
	// -------------------------

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

	// -------------------------
	// Schedule
	// -------------------------

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

	// -------------------------
	// Client appointments
	// -------------------------

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

	// -------------------------
	// Availability
	// -------------------------

	mux.Handle(
		"GET /api/v1/masters/{masterID}/availability",
		http.HandlerFunc(appointmentsHandler.GetAvailability),
	)

	// -------------------------
	// Master appointments
	// -------------------------

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

	// -------------------------
	// Notifications
	// -------------------------

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

	// -------------------------
	// Health
	// -------------------------

	mux.HandleFunc(
		"GET /api/health",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		},
	)

	mux.HandleFunc(
		"GET /api/ready",
		func(w http.ResponseWriter, r *http.Request) {
			if err := db.Ping(r.Context()); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"status":"not ready"}`))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ready"}`))
		},
	)

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", server.Addr)

		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)

	signal.Notify(
		stop,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-stop

	log.Println("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	log.Println("server stopped")
}
