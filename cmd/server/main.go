package main

import (
	"context"
	"github.com/you/go-jobsity-flights/internal/auth"
	"github.com/you/go-jobsity-flights/internal/config"
	"github.com/you/go-jobsity-flights/internal/httpx"
	"github.com/you/go-jobsity-flights/internal/providers"
	"github.com/you/go-jobsity-flights/internal/service"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	// Loading config
	cfg := config.Load()

	// Creating flight provider slice out of config parameters
	prov := []providers.FlightProvider{
		providers.NewAmadeus(cfg),
		providers.NewDuffel(cfg),
		providers.NewRapidBooking(cfg),
	}

	// Creating services
	searchSvc := service.NewSearchService(prov, cfg.SearchTimeout, cfg.CacheTTL)
	histSvc := service.NewHistoryService()

	publicMux := http.NewServeMux()

	// Public: login to get JWT
	publicMux.HandleFunc("/auth/login", auth.LoginHandler(cfg))

	// Protected group with JWT
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("/flights/search", httpx.SearchHandler(searchSvc))
	protectedMux.HandleFunc("/flights/history", httpx.HistoryHandler(histSvc))
	protectedMux.HandleFunc("/sse/", httpx.SubscribeSSEHandler(searchSvc)) // /sse/AMS/BCN?date=2025-10-01
	protectedMux.HandleFunc("/ws/", httpx.SubscribeWSHandler(searchSvc))

	// handler to control authenticated routes
	root := auth.JWTMiddleware(publicMux, protectedMux, cfg)

	// Creation of HTTP server
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           root,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      0,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Running http server on a secondary thread
	go func() {
		log.Printf("\n➡️ Server listening on http://localhost%s\n", srv.Addr)
		if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
			log.Println("🔐 TLS enabled")
			log.Fatal(srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile))
		} else {
			log.Fatal(srv.ListenAndServe())
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
