package main

import (
	"fmt"
	"os"
	"testing"
)

// User represents a system user with authentication data
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"-"`
}

// MockUser is a test user structure
type MockUser struct {
	ID   int
	Name string
}

// DatabaseConfig holds database connection settings
const DatabaseURL = "postgres://localhost:5432/mydb"
const APIVersion = "v1.2.0"

var globalCounter int
var isInitialized bool

// NewUser creates a new user instance
func NewUser(name, email string) *User {
	return &User{
		Name:  name,
		Email: email,
	}
}

// TestCreateUser tests user creation functionality
func TestCreateUser(t *testing.T) {
	user := NewUser("John Doe", "john@example.com")
	if user.Name != "John Doe" {
		t.Error("Name not set correctly")
	}
}

// TestUserValidation validates user data
func TestUserEmail(t *testing.T) {
	user := &User{Email: "invalid-email"}
	if !ValidateUser(user) {
		t.Error("Email validation failed")
	}
}

// ValidateUser performs user validation
func ValidateUser(user *User) bool {
	return len(user.Email) > 0 && len(user.Name) > 0
}

// (u *User) SetPassword sets the user password
func (u *User) SetPassword(password string) {
	u.Password = password
}

// (u *User) GetDisplayName returns formatted display name
func (u *User) GetDisplayName() string {
	return fmt.Sprintf("%s <%s>", u.Name, u.Email)
}

// (u *User) IsValid checks if user data is valid
func (u *User) IsValid() bool {
	return ValidateUser(u)
}

func main() {
	user := NewUser("Admin", "admin@example.com")
	user.SetPassword("secret123")

	if user.IsValid() {
		fmt.Println("User is valid:", user.GetDisplayName())
	} else {
		fmt.Println("Invalid user data")
		os.Exit(1)
	}
}
