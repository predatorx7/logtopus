package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/predatorx7/logtopus/pkg/auth"
)

func main() {
	clientID := flag.String("client", "", "Client ID to issue key for")
	secret := flag.String("secret", "", "Master secret key (or use AUTH_SECRET env var)")
	flag.Parse()

	if *clientID == "" {
		fmt.Println("Usage: apikey-gen -client <clientID> [-secret <secret>]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	secretKey := *secret
	if secretKey == "" {
		secretKey = os.Getenv("AUTH_SECRET")
	}
	if secretKey == "" {
		log.Fatal("Error: Secret is required via -secret flag or AUTH_SECRET env var")
	}

	key := auth.IssueAPIKey(*clientID, []byte(secretKey))
	fmt.Printf("Issued API Key for '%s':\n%s\n", *clientID, key)
}
