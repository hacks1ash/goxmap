// Example 01: Standard DTO mapping
//
// This example demonstrates mapping between a database model and a JSON-facing
// DTO, including pointer-to-value and value-to-pointer conversions.
//
// To regenerate:
//
//	go generate ./...
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

//go:generate go run github.com/hacks1ash/goxmap -src DBUser -dst UserResponse

// DBUser represents a row from the users table.
// Nullable columns are modeled as pointers.
type DBUser struct {
	ID        int     `json:"id"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`
	Age       int     `json:"age"`
	Active    bool    `json:"active"`
}

// UserResponse is the JSON DTO sent to API consumers.
// All fields are non-pointer for a clean JSON contract.
type UserResponse struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Age       int    `json:"age"`
	Active    bool   `json:"active"`
}

func main() {
	email := "alice@example.com"
	user := &DBUser{
		ID:        42,
		FirstName: "Alice",
		LastName:  "Smith",
		Email:     &email,
		Phone:     nil, // nullable column, no phone on file
		Age:       30,
		Active:    true,
	}

	dto := MapDBUserToUserResponse(user)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(dto); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
