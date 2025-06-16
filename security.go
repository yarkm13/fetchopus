package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/term"
)

type Credentials struct {
	username string
	password []byte
}

func (c *Credentials) Clear() {
	secureWipe(c.password)
	c.password = nil
}

// secureWipe safely clears sensitive data from memory
// It overwrites the slice with zeros and then sets it to nil
func secureWipe(data []byte) {
	if data == nil {
		return
	}
	for i := range data {
		data[i] = 0
	}
}

// askPassword securely reads a password from the terminal without echoing it
// Returns the password as a byte slice to allow secure handling in memory
// Supports very long strings by using a buffer-based approach
func askPassword() []byte {
	fmt.Print("Enter password: ")

	// Use a buffered reader approach for handling long inputs
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		log.Fatalf("Failed to set terminal to raw mode: %v", err)
	}
	defer term.Restore(fd, oldState)

	var password []byte
	buffer := make([]byte, 4096) // Large buffer for base64 encoded keys

	for {
		n, err := os.Stdin.Read(buffer[:])
		if err != nil {
			log.Fatalf("Error reading password: %v", err)
		}

		if n > 0 && (buffer[n-1] == '\r' || buffer[n-1] == '\n') {
			password = append(password, buffer[:n-1]...)
			break
		}

		password = append(password, buffer[:n]...)

		if len(password) > 65536 {
			log.Println("Warning: Very large string detected, truncating")
			break
		}
	}

	fmt.Println()
	return password
}
