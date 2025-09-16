package main

import (
	"fmt"
	"log"
)

// User represents a user in the system
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Database interface for data operations
type Database interface {
	GetUser(id int) (*User, error)
	SaveUser(user *User) error
}

const (
	DefaultPort = 8080
	APIVersion  = "v1"
)

var (
	globalConfig map[string]string
	userCache    []*User
)

// NewUser creates a new user instance
func NewUser(name, email string) *User {
	return &User{
		Name:  name,
		Email: email,
	}
}

// GetUserByID retrieves a user by ID
func GetUserByID(id int) (*User, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid user ID: %d", id)
	}

	// Simulate database lookup
	for _, user := range userCache {
		if user.ID == id {
			return user, nil
		}
	}

	return nil, fmt.Errorf("user not found: %d", id)
}

// UpdateUserEmail updates a user's email address with validation
func UpdateUserEmail(userID int, newEmail string) error {
	// Validate email first
	if !ValidateEmail(newEmail) {
		return fmt.Errorf("invalid email format: %s", newEmail)
	}

	user, err := GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	oldEmail := user.Email
	user.Email = newEmail
	log.Printf("Updated email for user %d from %s to %s", userID, oldEmail, newEmail)
	return nil
}

// ValidateEmail checks if an email is valid
func ValidateEmail(email string) bool {
	return len(email) > 0 && contains(email, "@")
}

// contains is a helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr)
}

func main() {
	fmt.Println("Testing Go provider transformations")

	user := NewUser("Test User", "test@example.com")
	fmt.Printf("Created user: %+v\n", user)

	if ValidateEmail(user.Email) {
		fmt.Println("Email is valid")
	}
}
