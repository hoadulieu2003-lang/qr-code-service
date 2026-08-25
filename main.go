package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
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

	router := NewRouter(config, nil)
	log.Printf("Server starting on port %s", config.Port)
	if config.SSL {
		err = http.ListenAndServeTLS(":"+config.Port, "certificate.pem", "key.pem", router)
	} else {
		err = http.ListenAndServe(":"+config.Port, router)
	}
	if err != nil {
		log.Print(err)
	}
}
