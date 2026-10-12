package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		client := http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("http://127.0.0.1:8080/api/health")
		if err != nil {
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			os.Exit(1)
		}
		return
	}
	location, err := time.LoadLocation(env("TZ", "Asia/Shanghai"))
	if err != nil {
		log.Fatal(err)
	}
	dataPath := env("DB_PATH", "data/aprs.sqlite")
	if err := os.MkdirAll(filepath.Dir(dataPath), 0700); err != nil {
		log.Fatal(err)
	}
	app, err := newApp(dataPath, location, env("PUBLIC_URL", ""), env("WEB_DIR", "web/build"))
	if err != nil {
		log.Fatal(err)
	}
	defer app.db.Close()
	server := &http.Server{Addr: env("LISTEN_ADDR", ":8080"), Handler: app.handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	log.Printf("Listening on %s; date timezone %s", server.Addr, location)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
