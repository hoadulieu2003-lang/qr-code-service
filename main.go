package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func runHealthcheck(config Config) int {
	protocol := "http"
	if config.SSL {
		protocol = "https"
	}

	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get(fmt.Sprintf("%s://localhost:%s/health", protocol, config.Port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
		return 1
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Health check failed: status %d\n", response.StatusCode)
		return 1
	}

	fmt.Println("OK")
	return 0
}

func main() {
	_ = godotenv.Load("default.env")
	if err := godotenv.Overload(".env"); err != nil {
		log.Print("No .env file provided, will continue with system env")
	}

	config, err := LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) > 1 && os.Args[1] == "--healthcheck" {
		os.Exit(runHealthcheck(config))
	}

	store, err := NewProductStore(config.DataFile)
	if err != nil {
		log.Fatal(err)
	}
	router := NewRouter(config, store)

	// Hardened HTTP Server with explicit timeouts to prevent Slowloris attacks
	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run the server in a separate goroutine
	go func() {
		log.Printf("Bond QR Service starting on port %s", config.Port)
		var srvErr error
		if config.SSL {
			srvErr = srv.ListenAndServeTLS("certificate.pem", "key.pem")
		} else {
			srvErr = srv.ListenAndServe()
		}
		if srvErr != nil && srvErr != http.ErrServerClosed {
			log.Fatalf("Bond server failure: %v", srvErr)
		}
	}()

	// Graceful Shutdown handling on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received termination signal (%v). Commencing graceful shutdown...", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced shutdown with error: %v", err)
	} else {
		log.Print("Server gracefully stopped. All in-flight requests drained.")
	}
}
