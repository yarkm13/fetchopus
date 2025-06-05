package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/term"
)

// Credentials securely stores authentication information
// The password is stored as a byte slice to allow secure clearing from memory
type Credentials struct {
	username string
	password []byte
}

// Clear securely wipes the password from memory by overwriting the byte slice
// with zeros before setting the reference to nil to prevent sensitive data leaks
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
func askPassword() []byte {
	fmt.Print("Enter password: ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		log.Fatal(err)
	}
	return password
}
