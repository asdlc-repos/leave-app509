package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/leave-app/notification-dispatcher/internal/client"
	"github.com/leave-app/notification-dispatcher/internal/dispatcher"
	"github.com/leave-app/notification-dispatcher/internal/mailer"
)

type config struct {
	Port         int
	APIURL       string
	ServiceToken string
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPass     string
	MailFrom     string
	RunOnce      bool
	Interval     time.Duration
}

func loadConfig() config {
	c := config{
		Port:         envInt("PORT", 9090),
		APIURL:       strings.TrimRight(envStr("LEAVE_API_URL", "http://leave-api:9090"), "/"),
		ServiceToken: envStr("SERVICE_TOKEN", "dev-service-token"),
		SMTPHost:     envStr("SMTP_HOST", ""),
		SMTPPort:     envInt("SMTP_PORT", 587),
		SMTPUser:     envStr("SMTP_USERNAME", ""),
		SMTPPass:     envStr("SMTP_PASSWORD", ""),
		MailFrom:     envStr("MAIL_FROM", "noreply@leave-app.local"),
		RunOnce:      envBool("RUN_ONCE", true),
		Interval:     envDuration("POLL_INTERVAL", 5*time.Minute),
	}
	return c
}

func envStr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	cfg := loadConfig()

	log.Printf("notification-dispatcher starting: api=%s smtp_host=%q run_once=%v interval=%s",
		cfg.APIURL, cfg.SMTPHost, cfg.RunOnce, cfg.Interval)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	healthSrv := startHealthServer(cfg.Port)

	api := client.New(cfg.APIURL, cfg.ServiceToken)
	mail := mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.MailFrom)
	disp := dispatcher.New(api, mail)

	// Small delay so the health server is ready before the first run.
	time.Sleep(200 * time.Millisecond)

	runOnce := func() {
		runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		if err := disp.Run(runCtx); err != nil {
			log.Printf("dispatch error: %v", err)
		}
	}

	runOnce()

	if !cfg.RunOnce {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Printf("shutdown signal received")
				goto done
			case <-ticker.C:
				runOnce()
			}
		}
	}

done:
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("health server shutdown error: %v", err)
	}
	log.Printf("notification-dispatcher exiting cleanly")
}

func startHealthServer(port int) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("notification-dispatcher"))
			return
		}
		http.NotFound(w, r)
	})

	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("health server listening on :%d", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("health server error: %v", err)
		}
	}()
	return srv
}
