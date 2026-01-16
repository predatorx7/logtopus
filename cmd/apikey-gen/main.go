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
	verifyKey := flag.String("verify", "", "API Key to verify")
	secret := flag.String("secret", "", "Master secret key (or use AUTH_SECRET env var)")
	flag.Parse()

	secretKey := *secret
	if secretKey == "" {
		secretKey = os.Getenv("AUTH_SECRET")
	}
	if secretKey == "" {
		log.Fatal("Error: Secret is required via -secret flag or AUTH_SECRET env var")
	}

	// Verification Mode
	if *verifyKey != "" {
		valid, extractedClient, err := auth.VerifyAPIKey(*verifyKey, []byte(secretKey))
		if err != nil {
			fmt.Printf("Invalid Key: %v\n", err)
			os.Exit(1)
		}
		if valid {
			fmt.Printf("Valid API Key for Client ID: %s\n", extractedClient)
			os.Exit(0)
		} else {
			fmt.Println("Invalid API Key (Signature mismatch)")
			os.Exit(1)
		}
	}

	// Generation Mode
	if *clientID == "" {
		fmt.Println("Usage:")
		fmt.Println("  Generate: apikey-gen -client <clientID> [-secret <secret>]")
		fmt.Println("  Verify:   apikey-gen -verify <apiKey> [-secret <secret>]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	key := auth.IssueAPIKey(*clientID, []byte(secretKey))
	fmt.Printf("Issued API Key for '%s':\n%s\n", *clientID, key)
}
