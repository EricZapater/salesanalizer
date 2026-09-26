package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Ús: go run ./cmd/hashpass/main.go <la_teva_contrasenya>")
		os.Exit(1)
	}

	password := os.Args[1]
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generant hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Hash bcrypt generat:")
	fmt.Println(string(hash))
	fmt.Println("\nCopia aquest valor a ADMIN_PASSWORD_HASH al teu .env.backend:")
	fmt.Printf("ADMIN_PASSWORD_HASH='%s'\n", string(hash))
}
