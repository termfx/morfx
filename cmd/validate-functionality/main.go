package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// FunctionalTest represents a single functional test
type FunctionalTest struct {
	Name        string
	Description string
	Language    string
	Query       string
	FilePattern string
	ExpectedMin int // Minimum expected matches
	ExpectedMax int // Maximum expected matches (-1 for no limit)
}

// TestResult represents the result of a functional test
type TestResult struct {
	Test     FunctionalTest
	Passed   bool
	Output   string
	Error    string
	Duration time.Duration
}

func main() {
	fmt.Println("🚀 MORFX Functional Validation")
	fmt.Println("==============================")
	fmt.Println()

	// First, build morfx
	fmt.Println("Building morfx...")
	err := buildMorfx()
	if err != nil {
		fmt.Printf("❌ Failed to build morfx: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ morfx built successfully")
	fmt.Println()

	// Create test files
	err = createTestFiles()
	if err != nil {
		fmt.Printf("❌ Failed to create test files: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Test files created")
	fmt.Println()

	// Define functional tests
	tests := []FunctionalTest{
		// DSL Flexibility Tests - Python syntax
		{
			Name:        "Python DSL - Function",
			Description: "Test 'def:' syntax for Python functions",
			Language:    "python",
			Query:       "def:test*",
			FilePattern: "testdata/sample.py",
			ExpectedMin: 1,
			ExpectedMax: 5,
		},
		{
			Name:        "Python DSL - Class",
			Description: "Test 'class:' syntax for Python classes",
			Language:    "python",
			Query:       "class:User",
			FilePattern: "testdata/sample.py",
			ExpectedMin: 0,
			ExpectedMax: 2,
		},

		// DSL Flexibility Tests - Go syntax
		{
			Name:        "Go DSL - Function",
			Description: "Test 'func:' syntax for Go functions",
			Language:    "go",
			Query:       "func:Test*",
			FilePattern: "testdata/sample.go",
			ExpectedMin: 1,
			ExpectedMax: 5,
		},
		{
			Name:        "Go DSL - Struct",
			Description: "Test 'struct:' syntax for Go structs",
			Language:    "go",
			Query:       "struct:User",
			FilePattern: "testdata/sample.go",
			ExpectedMin: 0,
			ExpectedMax: 2,
		},

		// Operator Efficiency Tests - Single operators
		{
			Name:        "Single AND Operator",
			Description: "Test single '&' operator works correctly",
			Language:    "go",
			Query:       "func:Test* & !func:TestHelper",
			FilePattern: "testdata/sample.go",
			ExpectedMin: 1,
			ExpectedMax: 5,
		},
		{
			Name:        "Single OR Operator",
			Description: "Test single '|' operator works correctly",
			Language:    "go",
			Query:       "func:main | func:Test*",
			FilePattern: "testdata/sample.go",
			ExpectedMin: 1,
			ExpectedMax: 10,
		},
		{
			Name:        "Single NOT Operator",
			Description: "Test single '!' operator works correctly",
			Language:    "go",
			Query:       "!func:TestHelper",
			FilePattern: "testdata/sample.go",
			ExpectedMin: 1,
			ExpectedMax: 20,
		},

		// Operator Efficiency Tests - Double operators (aliases)
		{
			Name:        "Double AND Operator",
			Description: "Test double '&&' operator works as alias",
			Language:    "go",
			Query:       "func:Test* && !func:TestHelper",
			FilePattern: "testdata/sample.go",
			ExpectedMin: 1,
			ExpectedMax: 5,
		},
		{
			Name:        "Double OR Operator",
			Description: "Test double '||' operator works as alias",
			Language:    "go",
			Query:       "func:main || func:Test*",
			FilePattern: "testdata/sample.go",
			ExpectedMin: 1,
			ExpectedMax: 10,
		},
		{
			Name:        "Word AND Operator",
			Description: "Test 'and' operator works as alias",
			Language:    "go",
			Query:       "func:Test* and !func:TestHelper",
			FilePattern: "testdata/sample.go",
			ExpectedMin: 1,
			ExpectedMax: 5,
		},
		{
			Name:        "Word OR Operator",
			Description: "Test 'or' operator works as alias",
			Language:    "go",
			Query:       "func:main or func:Test*",
			FilePattern: "testdata/sample.go",
			ExpectedMin: 1,
			ExpectedMax: 10,
		},

		// JavaScript tests
		{
			Name:        "JavaScript DSL - Function",
			Description: "Test 'function:' syntax for JavaScript functions",
			Language:    "javascript",
			Query:       "function:test* | const:API",
			FilePattern: "testdata/sample.js",
			ExpectedMin: 1,
			ExpectedMax: 10,
		},
	}

	// Run tests
	results := make([]TestResult, 0, len(tests))
	passed := 0

	for i, test := range tests {
		fmt.Printf("Running test %d/%d: %s\n", i+1, len(tests), test.Name)
		result := runFunctionalTest(test)
		results = append(results, result)

		if result.Passed {
			passed++
			fmt.Printf("  ✅ PASSED (%.2fs)\n", result.Duration.Seconds())
		} else {
			fmt.Printf("  ❌ FAILED (%.2fs)\n", result.Duration.Seconds())
			if result.Error != "" {
				fmt.Printf("     Error: %s\n", result.Error)
			}
		}
	}

	fmt.Println()
	fmt.Println("FUNCTIONAL VALIDATION SUMMARY")
	fmt.Println("=============================")
	fmt.Printf("Total Tests: %d\n", len(tests))
	fmt.Printf("Passed: %d\n", passed)
	fmt.Printf("Failed: %d\n", len(tests)-passed)
	fmt.Printf("Pass Rate: %.1f%%\n", float64(passed)/float64(len(tests))*100.0)

	// Print failed tests details
	if passed < len(tests) {
		fmt.Println()
		fmt.Println("FAILED TESTS:")
		for _, result := range results {
			if !result.Passed {
				fmt.Printf("❌ %s\n", result.Test.Name)
				fmt.Printf("   Query: %s\n", result.Test.Query)
				if result.Error != "" {
					fmt.Printf("   Error: %s\n", result.Error)
				}
				if result.Output != "" && len(result.Output) < 500 {
					fmt.Printf("   Output: %s\n", strings.TrimSpace(result.Output))
				}
			}
		}
	}

	// Clean up test files
	cleanupTestFiles()

	if passed == len(tests) {
		fmt.Println()
		fmt.Println("✅ All functional validation tests PASSED!")
		os.Exit(0)
	} else {
		fmt.Println()
		fmt.Printf("❌ %d of %d functional tests FAILED!\n", len(tests)-passed, len(tests))
		os.Exit(1)
	}
}

func buildMorfx() error {
	cmd := exec.Command("go", "build", "-o", "morfx", "./cmd/morfx")
	return cmd.Run()
}

func createTestFiles() error {
	err := os.MkdirAll("testdata", 0o755)
	if err != nil {
		return err
	}

	// Create Go test file
	goContent := `package main

import "fmt"

type User struct {
	Name string
	ID   int
}

type Admin struct {
	User
	Permissions []string
}

func main() {
	fmt.Println("Hello World")
}

func TestUser() {
	user := User{Name: "test", ID: 1}
	fmt.Println(user)
}

func TestAdmin() {
	admin := Admin{}
	fmt.Println(admin)
}

func TestHelper() {
	// This is a helper function
}

func NewUser(name string) *User {
	return &User{Name: name}
}

func (u *User) GetName() string {
	return u.Name
}

func (u *User) SetName(name string) {
	u.Name = name
}
`

	err = os.WriteFile("testdata/sample.go", []byte(goContent), 0o644)
	if err != nil {
		return err
	}

	// Create Python test file
	pythonContent := `#!/usr/bin/env python3

class User:
    def __init__(self, name):
        self.name = name
    
    def get_name(self):
        return self.name

class Admin(User):
    def __init__(self, name, permissions):
        super().__init__(name)
        self.permissions = permissions

def test_user_creation():
    user = User("test")
    assert user.get_name() == "test"

def test_admin_creation():
    admin = Admin("admin", ["read", "write"])
    assert admin.get_name() == "admin"

def helper_function():
    pass

def main():
    print("Hello World")
    test_user_creation()
    test_admin_creation()

if __name__ == "__main__":
    main()
`

	err = os.WriteFile("testdata/sample.py", []byte(pythonContent), 0o644)
	if err != nil {
		return err
	}

	// Create JavaScript test file
	jsContent := `const API_URL = "https://api.example.com";
const API_VERSION = "v1";

class User {
    constructor(name) {
        this.name = name;
    }

    getName() {
        return this.name;
    }
}

class Admin extends User {
    constructor(name, permissions) {
        super(name);
        this.permissions = permissions;
    }
}

function testUserCreation() {
    const user = new User("test");
    console.log(user.getName());
}

function testAdminCreation() {
    const admin = new Admin("admin", ["read", "write"]);
    console.log(admin.getName());
}

function main() {
    console.log("Hello World");
    testUserCreation();
    testAdminCreation();
}

module.exports = { User, Admin, testUserCreation, testAdminCreation };
`

	err = os.WriteFile("testdata/sample.js", []byte(jsContent), 0o644)
	if err != nil {
		return err
	}

	return nil
}

func runFunctionalTest(test FunctionalTest) TestResult {
	start := time.Now()

	// Build command
	var cmd *exec.Cmd
	if test.Language != "" {
		cmd = exec.Command("./morfx", "-lang", test.Language, test.Query, test.FilePattern)
	} else {
		cmd = exec.Command("./morfx", test.Query, test.FilePattern)
	}

	// Run command
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	result := TestResult{
		Test:     test,
		Output:   string(output),
		Duration: duration,
	}

	if err != nil {
		result.Error = err.Error()
		result.Passed = false
		return result
	}

	// Check if we got expected results
	outputStr := string(output)
	matchesFound := countMatches(outputStr)

	// Validate match count is within expected range
	if matchesFound >= test.ExpectedMin && (test.ExpectedMax == -1 || matchesFound <= test.ExpectedMax) {
		result.Passed = true
	} else {
		result.Passed = false
		result.Error = fmt.Sprintf("Expected %d-%d matches, got %d",
			test.ExpectedMin, test.ExpectedMax, matchesFound)
	}

	return result
}

func countMatches(output string) int {
	// Count lines that look like match results
	// Format: "1. Function 'TestUser' at line 15:6"
	lines := strings.Split(output, "\n")
	count := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ". ") && strings.Contains(line, " at line ") {
			count++
		}
	}

	return count
}

func cleanupTestFiles() {
	os.RemoveAll("testdata")
	os.Remove("morfx")
}
