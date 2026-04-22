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

	"github.com/asdlc/leave-api/internal/auth"
	"github.com/asdlc/leave-api/internal/handlers"
	"github.com/asdlc/leave-api/internal/middleware"
	"github.com/asdlc/leave-api/internal/store"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	port := envOr("PORT", "9090")
	jwtSecret := envOr("JWT_SECRET", "dev-leave-api-secret-change-me")
	serviceToken := envOr("SERVICE_TOKEN", "dev-service-token")

	signer := auth.NewSigner(jwtSecret, 30*time.Minute)
	st := store.New()
	h := handlers.New(st, signer)

	mux := http.NewServeMux()

	// Public endpoints
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("POST /auth/logout", h.Logout)

	authMW := middleware.Auth(signer)
	svcOrAuthMW := middleware.ServiceTokenOrAuth(signer, serviceToken)

	// Auth-protected endpoints
	protect := func(handler http.HandlerFunc) http.Handler {
		return authMW(handler)
	}

	mux.Handle("GET /auth/me", protect(h.Me))

	mux.Handle("GET /users", protect(h.ListUsers))
	mux.Handle("GET /users/{id}/balances", protect(h.GetBalances))
	mux.Handle("POST /users/{id}/balances/adjust", protect(h.AdjustBalance))

	mux.Handle("GET /leave-types", protect(h.ListLeaveTypes))
	mux.Handle("POST /leave-types", protect(h.CreateLeaveType))
	mux.Handle("PUT /leave-types/{id}", protect(h.UpdateLeaveType))

	mux.Handle("GET /leave-requests", protect(h.ListLeaveRequests))
	mux.Handle("POST /leave-requests", protect(h.CreateLeaveRequest))
	mux.Handle("GET /leave-requests/{id}", protect(h.GetLeaveRequest))
	mux.Handle("POST /leave-requests/{id}/cancel", protect(h.CancelLeaveRequest))
	mux.Handle("POST /leave-requests/{id}/approve", protect(h.ApproveLeaveRequest))
	mux.Handle("POST /leave-requests/{id}/reject", protect(h.RejectLeaveRequest))
	mux.Handle("POST /leave-requests/{id}/attachments", protect(h.UploadAttachment))

	mux.Handle("GET /calendar", protect(h.Calendar))
	mux.Handle("GET /manager/queue", protect(h.ManagerQueue))
	mux.Handle("GET /manager/team-calendar", protect(h.ManagerTeamCalendar))

	mux.Handle("GET /policies/blackouts", protect(h.ListBlackouts))
	mux.Handle("POST /policies/blackouts", protect(h.CreateBlackout))

	mux.Handle("GET /reports/utilization", protect(h.UtilizationReport))
	mux.Handle("GET /audit/{employeeId}", protect(h.AuditForEmployee))

	// Notifications: email queue readable by dispatcher (service token) or HR; in-app per user.
	mux.Handle("GET /notifications/pending", svcOrAuthMW(http.HandlerFunc(h.PendingNotifications)))
	mux.Handle("POST /notifications/{id}/mark-sent", svcOrAuthMW(http.HandlerFunc(h.MarkNotificationSent)))
	mux.Handle("GET /notifications/inapp", protect(h.InAppNotifications))

	handler := middleware.CORS(mux)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Background reminder generator (runs every 15 minutes; endpoint also generates on poll).
	stopReminders := make(chan struct{})
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		h.GenerateReminders()
		for {
			select {
			case <-ticker.C:
				h.GenerateReminders()
			case <-stopReminders:
				return
			}
		}
	}()

	go func() {
		log.Printf("leave-api listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Println("shutting down…")
	close(stopReminders)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
