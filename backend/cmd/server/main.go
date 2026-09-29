package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"masterbook/internal/admin"
	"masterbook/internal/appointments"
	"masterbook/internal/auth"
	"masterbook/internal/config"
	"masterbook/internal/database"
	"masterbook/internal/httpx"
	"masterbook/internal/masters"
	"masterbook/internal/notifications"
	"masterbook/internal/schedule"
	"masterbook/internal/services"
)

//go:embed web/*
var webFiles embed.FS

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

	if err := os.MkdirAll(filepath.Join(cfg.UploadsDir, "avatars"), 0o755); err != nil {
		log.Fatal(err)
	}

	userRepo := auth.NewPostgresRepository(db)
	authService := auth.NewService(userRepo, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService)
	masterRepo := masters.NewPostgresRepository(db)
	masterService := masters.NewService(masterRepo, cfg.UploadsDir)
	masterHandler := masters.NewHandler(masterService)
	serviceRepo := services.NewPostgresRepository(db)
	serviceService := services.NewService(serviceRepo, masterRepo)
	serviceHandler := services.NewHandler(serviceService)
	scheduleRepo := schedule.NewPostgresRepository(db)
	scheduleService := schedule.NewService(scheduleRepo, masterRepo)
	scheduleHandler := schedule.NewHandler(scheduleService)
	notificationRepo := notifications.NewPostgresRepository(db)
	notificationService := notifications.NewService(notificationRepo)
	notificationHandler := notifications.NewHandler(notificationService)
	appointmentRepo := appointments.NewPostgresRepository(db)
	appointmentService := appointments.NewService(appointmentRepo, notificationService)
	appointmentHandler := appointments.NewHandler(appointmentService)
	adminService := admin.NewService(authService, masterService)
	adminHandler := admin.NewHandler(adminService)

	authMiddleware := auth.AuthMiddleware(cfg.JWTSecret)
	mux := http.NewServeMux()

	authHandler.RegisterRoutes(mux, authMiddleware)
	masterHandler.RegisterRoutes(mux, authMiddleware)
	serviceHandler.RegisterRoutes(mux, authMiddleware)
	scheduleHandler.RegisterRoutes(mux, authMiddleware)
	appointmentHandler.RegisterRoutes(mux, authMiddleware)
	notificationHandler.RegisterRoutes(mux, authMiddleware)
	adminHandler.RegisterRoutes(mux, authMiddleware)

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready"})
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadsDir))))

	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(webFS)))

	server := &http.Server{Addr: ":" + cfg.HTTPPort, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	log.Println("server stopped")
}
