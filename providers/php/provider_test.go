package php

import (
	"slices"
	"strings"
	"testing"

	"github.com/termfx/morfx/core"
)

func TestPHPProvider_New(t *testing.T) {
	provider := New()
	if provider == nil {
		t.Fatal("New returned nil")
	}
	if provider.Language() != "php" {
		t.Errorf("Expected language 'php', got '%s'", provider.Language())
	}
}

func TestPHPProvider_Language(t *testing.T) {
	provider := New()
	if provider.Language() != "php" {
		t.Errorf("Expected language 'php', got '%s'", provider.Language())
	}
}

func TestPHPProvider_Extensions(t *testing.T) {
	provider := New()
	extensions := provider.Extensions()

	expected := []string{".php", ".phtml", ".php4", ".php5", ".phps"}
	if len(extensions) != len(expected) {
		t.Errorf("Expected %d extensions, got %d", len(expected), len(extensions))
	}

	for _, ext := range expected {
		found := slices.Contains(extensions, ext)
		if !found {
			t.Errorf("Expected extension '%s' not found", ext)
		}
	}
}

func TestPHPProvider_Query_Functions(t *testing.T) {
	provider := New()
	source := `<?php
function getUserData($id) {
    return "User " . $id;
}

function processUser($user) {
    return strtoupper($user);
}
`

	query := core.AgentQuery{
		Type: "function",
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	if len(result.Matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(result.Matches))
	}

	// Should find both functions
	names := make([]string, len(result.Matches))
	for i, match := range result.Matches {
		names[i] = match.Name
	}

	foundGetUserData := false
	foundProcessUser := false
	for _, name := range names {
		if name == "getUserData" {
			foundGetUserData = true
		}
		if name == "processUser" {
			foundProcessUser = true
		}
	}

	if !foundGetUserData {
		t.Error("Expected to find 'getUserData' function")
	}
	if !foundProcessUser {
		t.Error("Expected to find 'processUser' function")
	}
}

func TestPHPProvider_Query_Classes(t *testing.T) {
	provider := New()
	source := `<?php
class User {
    private $name;
    
    public function __construct($name) {
        $this->name = $name;
    }
    
    public function getName() {
        return $this->name;
    }
}

class UserController {
    public function index() {
        return "Users list";
    }
}
`

	query := core.AgentQuery{
		Type: "class",
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	if len(result.Matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(result.Matches))
	}

	names := make([]string, len(result.Matches))
	for i, match := range result.Matches {
		names[i] = match.Name
	}

	foundUser := false
	foundUserController := false
	for _, name := range names {
		if name == "User" {
			foundUser = true
		}
		if name == "UserController" {
			foundUserController = true
		}
	}

	if !foundUser {
		t.Error("Expected to find 'User' class")
	}
	if !foundUserController {
		t.Error("Expected to find 'UserController' class")
	}
}

func TestPHPProvider_Query_Methods(t *testing.T) {
	provider := New()
	source := `<?php
class Calculator {
    public function add($a, $b) {
        return $a + $b;
    }
    
    public function subtract($a, $b) {
        return $a - $b;
    }
}
`

	query := core.AgentQuery{
		Type: "function", // Methods are also functions in PHP
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	// Should find at least the methods
	if len(result.Matches) < 2 {
		t.Errorf("Expected at least 2 matches, got %d", len(result.Matches))
	}
}

func TestPHPProvider_Transform_Replace(t *testing.T) {
	provider := New()
	source := `<?php
function greet($name) {
    return "Hello " . $name;
}
`

	transform := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "greet",
		},
		Replacement: "function greet($name) {\n    return 'Hi ' . $name;\n}",
	}

	result := provider.Transform(source, transform)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	if result.Modified == "" {
		t.Error("Expected modified code, got empty string")
	}

	if result.Confidence.Score <= 0.5 {
		t.Errorf("Expected confidence > 0.5, got %f", result.Confidence.Score)
	}

	if result.MatchCount == 0 {
		t.Error("Expected at least 1 match, got 0")
	}
}

func TestPHPProvider_Transform_Delete(t *testing.T) {
	provider := New()
	source := `<?php
function greet($name) {
    return "Hello " . $name;
}

function farewell($name) {
    return "Goodbye " . $name;
}
`

	transform := core.TransformOp{
		Method: "delete",
		Target: core.AgentQuery{
			Type: "function",
			Name: "greet",
		},
	}

	result := provider.Transform(source, transform)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	if result.Modified == "" {
		t.Error("Expected modified code, got empty string")
	}

	if result.Confidence.Score <= 0.5 {
		t.Errorf("Expected confidence > 0.5, got %f", result.Confidence.Score)
	}

	if result.MatchCount == 0 {
		t.Error("Expected at least 1 match, got 0")
	}
}

func TestPHPProvider_Validate(t *testing.T) {
	provider := New()

	// Test valid code
	validSource := `<?php
class User {
    private $name;
    
    public function __construct($name) {
        $this->name = $name;
    }
    
    public function getName() {
        return $this->name;
    }
}

function processUser($user) {
    return $user->getName();
}
`

	result := provider.Validate(validSource)
	if !result.Valid {
		t.Errorf("Expected valid code to be valid, got errors: %v", result.Errors)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors for valid code, got %d errors", len(result.Errors))
	}
}

// Test ExtractNodeName to improve coverage
func TestPHPProvider_ExtractNodeName(t *testing.T) {
	provider := New()
	source := `<?php

namespace App\Services;

use App\Models\User;

// Function declaration
function calculateTotal($items) {
    return array_sum($items);
}

// Class declaration
class UserService {
    
    // Method definition
    public function createUser($name, $email) {
        return new User($name, $email);
    }
    
    // Static method
    public static function validateEmail($email) {
        return filter_var($email, FILTER_VALIDATE_EMAIL);
    }
    
    // Private method
    private function _hashPassword($password) {
        return password_hash($password, PASSWORD_DEFAULT);
    }
}

// Interface declaration
interface PaymentProcessor {
    public function processPayment($amount);
}

// Trait declaration
trait Timestampable {
    public function touch() {
        $this->updated_at = time();
    }
}

// Class constants
class Config {
    const API_URL = 'https://api.example.com';
    const MAX_RETRIES = 3;
}

// Variables
$userName = 'John Doe';
$userAge = 30;
$isActive = true;

// Function with closure
function processData($data) {
    return array_map(function($item) {
        return strtoupper($item);
    }, $data);
}
`

	// Test function queries
	funcQuery := core.AgentQuery{
		Type: "function",
	}

	result := provider.Query(source, funcQuery)
	if result.Error != nil {
		t.Fatalf("Function query failed: %v", result.Error)
	}

	foundFunctions := make(map[string]bool)
	for _, match := range result.Matches {
		foundFunctions[match.Name] = true
		t.Logf("Found function: %s", match.Name)
	}

	expectedFunctions := []string{
		"calculateTotal",
		"createUser",
		"validateEmail",
		"_hashPassword",
		"processPayment",
		"touch",
		"processData",
	}
	foundAnyFunction := false
	for _, expected := range expectedFunctions {
		if foundFunctions[expected] {
			foundAnyFunction = true
			break
		}
	}

	if !foundAnyFunction {
		t.Error("Expected to find at least one function")
	}

	// Test class queries
	classQuery := core.AgentQuery{
		Type: "class",
	}

	classResult := provider.Query(source, classQuery)
	if classResult.Error != nil {
		t.Fatalf("Class query failed: %v", classResult.Error)
	}

	foundClasses := make(map[string]bool)
	for _, match := range classResult.Matches {
		foundClasses[match.Name] = true
		t.Logf("Found class: %s", match.Name)
	}

	expectedClasses := []string{"UserService", "Config"}
	for _, expected := range expectedClasses {
		if !foundClasses[expected] {
			t.Errorf("Expected to find class '%s'", expected)
		}
	}

	// Test interface queries
	interfaceQuery := core.AgentQuery{
		Type: "interface",
	}

	interfaceResult := provider.Query(source, interfaceQuery)
	if interfaceResult.Error != nil {
		t.Fatalf("Interface query failed: %v", interfaceResult.Error)
	}

	foundInterfaces := make(map[string]bool)
	for _, match := range interfaceResult.Matches {
		foundInterfaces[match.Name] = true
		t.Logf("Found interface: %s", match.Name)
	}

	if !foundInterfaces["PaymentProcessor"] {
		t.Error("Expected to find PaymentProcessor interface")
	}

	// Test trait queries
	traitQuery := core.AgentQuery{
		Type: "trait",
	}

	traitResult := provider.Query(source, traitQuery)
	if traitResult.Error != nil {
		t.Fatalf("Trait query failed: %v", traitResult.Error)
	}

	foundTraits := make(map[string]bool)
	for _, match := range traitResult.Matches {
		foundTraits[match.Name] = true
		t.Logf("Found trait: %s", match.Name)
	}

	if !foundTraits["Timestampable"] {
		t.Error("Expected to find Timestampable trait")
	}
}

// Test error handling and edge cases for better coverage
func TestPHPProvider_ErrorHandling(t *testing.T) {
	provider := New()

	// Test malformed PHP code
	malformedSource := `<?php
	class MalformedClass {
		public function badMethod(
		// Missing closing brace and parameters
	`

	result := provider.Validate(malformedSource)
	if result.Valid {
		t.Error("Expected malformed code to be invalid")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected validation errors for malformed code")
	}

	// Test query on malformed code (may or may not error depending on tree-sitter parser robustness)
	query := core.AgentQuery{Type: "class"}
	queryResult := provider.Query(malformedSource, query)
	// Tree-sitter is robust and may still parse partial AST, so we just log the result
	t.Logf("Query on malformed code returned: error=%v, matches=%d", queryResult.Error, len(queryResult.Matches))
}

func TestPHPProvider_ComplexExtractNodeName(t *testing.T) {
	provider := New()
	source := `<?php

namespace App\Services\Payment;

use App\Models\{User, Order, Payment};
use App\Contracts\PaymentProcessorInterface;
use Some\External\Library as ExtLib;

// Abstract class
abstract class BasePaymentProcessor implements PaymentProcessorInterface {
	// Property declarations
	protected $config;
	private $logger;
	public static $instances = [];

	// Constants
	const DEFAULT_TIMEOUT = 30;
	const MAX_RETRIES = 3;

	// Constructor
	public function __construct(array $config) {
		$this->config = $config;
	}

	// Abstract method
	abstract protected function processPayment(Payment $payment): bool;

	// Final method
	final public function validatePayment($payment) {
		return $payment instanceof Payment;
	}

	// Static method
	public static function getInstance($type) {
		return self::$instances[$type] ?? null;
	}

	// Magic methods
	public function __toString() {
		return static::class;
	}

	public function __call($method, $args) {
		throw new \BadMethodCallException("Method {$method} not found");
	}
}

// Concrete implementation
class StripePaymentProcessor extends BasePaymentProcessor {
	protected function processPayment(Payment $payment): bool {
		// Implementation here
		return true;
	}

	// Method with complex parameters
	public function refundPayment(
		Payment $payment,
		float $amount = null,
		array $options = [],
		callable $callback = null
	): RefundResult {
		// Complex method implementation
		return new RefundResult();
	}
}

// Interface with methods
interface RefundableProcessor {
	public function canRefund(Payment $payment): bool;
	public function refund(Payment $payment, float $amount): RefundResult;
}

// Trait with methods
trait LoggableTrait {
	protected $logLevel = 'info';

	protected function log(string $message, string $level = 'info'): void {
		// Logging implementation
	}

	public function setLogLevel(string $level): self {
		$this->logLevel = $level;
		return $this;
	}
}

// Anonymous class assignment
$anonymousProcessor = new class implements PaymentProcessorInterface {
	public function process($data) {
		return true;
	}
};

// Closure assignment
$validationCallback = function($payment) use ($config) {
	return $payment->amount > 0;
};

// Arrow function (PHP 7.4+)
$formatter = fn($amount) => number_format($amount, 2);

// Global variables
$globalConfig = ['timeout' => 30];
$_SECRET_KEY = 'sk_test_123';
`

	// Test namespace queries
	namespaceQuery := core.AgentQuery{Type: "namespace"}
	namespaceResult := provider.Query(source, namespaceQuery)
	if namespaceResult.Error != nil {
		t.Fatalf("Namespace query failed: %v", namespaceResult.Error)
	}

	if len(namespaceResult.Matches) > 0 {
		for _, match := range namespaceResult.Matches {
			t.Logf("Found namespace: %s", match.Name)
		}
	}

	// Test use/import queries
	useQuery := core.AgentQuery{Type: "use"}
	useResult := provider.Query(source, useQuery)
	if useResult.Error != nil {
		t.Fatalf("Use query failed: %v", useResult.Error)
	}

	if len(useResult.Matches) > 0 {
		for _, match := range useResult.Matches {
			t.Logf("Found use statement: %s", match.Name)
		}
	}

	// Test constant queries
	constQuery := core.AgentQuery{Type: "const"}
	constResult := provider.Query(source, constQuery)
	if constResult.Error != nil {
		t.Fatalf("Const query failed: %v", constResult.Error)
	}

	// Test variable queries
	varQuery := core.AgentQuery{Type: "variable"}
	varResult := provider.Query(source, varQuery)
	if varResult.Error != nil {
		t.Fatalf("Variable query failed: %v", varResult.Error)
	}

	if len(varResult.Matches) > 0 {
		for _, match := range varResult.Matches {
			t.Logf("Found variable: %s", match.Name)
		}
	}
}

// Test complex transformation scenarios
func TestPHPProvider_ComplexTransformations(t *testing.T) {
	provider := New()
	source := `<?php

class UserService {
	public function getUser($id) {
		return User::find($id);
	}

	public function createUser($data) {
		return User::create($data);
	}

	public function deleteUser($id) {
		$user = User::find($id);
		if ($user) {
			$user->delete();
			return true;
		}
		return false;
	}
}
`

	// Test insert_before transformation
	insertBeforeOp := core.TransformOp{
		Method: "insert_before",
		Target: core.AgentQuery{
			Type: "function",
			Name: "createUser",
		},
		Content: "// Validation method\n\tpublic function validateUserData($data) {\n\t\treturn !empty($data['name']) && !empty($data['email']);\n\t}",
	}

	result := provider.Transform(source, insertBeforeOp)
	if result.Error != nil {
		t.Fatalf("Insert before transform failed: %v", result.Error)
	}

	if result.Modified == "" {
		t.Error("Expected modified code for insert_before")
	}

	// Test insert_after transformation
	insertAfterOp := core.TransformOp{
		Method: "insert_after",
		Target: core.AgentQuery{
			Type: "function",
			Name: "getUser",
		},
		Content: "\n\t// Get user by email\n\tpublic function getUserByEmail($email) {\n\t\treturn User::where('email', $email)->first();\n\t}",
	}

	result2 := provider.Transform(source, insertAfterOp)
	if result2.Error != nil {
		t.Fatalf("Insert after transform failed: %v", result2.Error)
	}

	if result2.Modified == "" {
		t.Error("Expected modified code for insert_after")
	}

	// Test append transformation
	appendOp := core.TransformOp{
		Method: "append",
		Target: core.AgentQuery{
			Type: "class",
			Name: "UserService",
		},
		Content: "\n\t// New method added via append\n\tpublic function getUserCount() {\n\t\treturn User::count();\n\t}",
	}

	result3 := provider.Transform(source, appendOp)
	if result3.Error != nil {
		t.Fatalf("Append transform failed: %v", result3.Error)
	}

	if result3.Modified == "" {
		t.Error("Expected modified code for append")
	}
}

// Test pattern matching and wildcards
func TestPHPProvider_PatternMatching(t *testing.T) {
	provider := New()
	source := `<?php

class UserController {
	public function getUserData($id) {
		return User::find($id);
	}

	public function getUserProfile($id) {
		return Profile::findByUserId($id);
	}

	public function getAdminData($id) {
		return Admin::find($id);
	}

	public function setUserActive($id) {
		$user = User::find($id);
		$user->active = true;
		$user->save();
	}
}
`

	// Test wildcard pattern matching
	getUserQuery := core.AgentQuery{
		Type: "function",
		Name: "getUser*", // Should match getUserData and getUserProfile
	}

	result := provider.Query(source, getUserQuery)
	if result.Error != nil {
		t.Fatalf("Wildcard query failed: %v", result.Error)
	}

	// Should find getUserData and getUserProfile, but not getAdminData or setUserActive
	expectedMatches := 2
	if len(result.Matches) != expectedMatches {
		t.Errorf("Expected %d matches for 'getUser*', got %d", expectedMatches, len(result.Matches))
	}

	for _, match := range result.Matches {
		if !strings.HasPrefix(match.Name, "getUser") {
			t.Errorf("Unexpected match '%s' for pattern 'getUser*'", match.Name)
		}
	}

	// Test suffix wildcard
	dataQuery := core.AgentQuery{
		Type: "function",
		Name: "*Data", // Should match getUserData and getAdminData
	}

	result2 := provider.Query(source, dataQuery)
	if result2.Error != nil {
		t.Fatalf("Suffix wildcard query failed: %v", result2.Error)
	}

	expectedSuffixMatches := 2
	if len(result2.Matches) != expectedSuffixMatches {
		t.Errorf("Expected %d matches for '*Data', got %d", expectedSuffixMatches, len(result2.Matches))
	}

	// Test middle wildcard
	middleQuery := core.AgentQuery{
		Type: "function",
		Name: "get*Data", // Should match getUserData and getAdminData
	}

	result3 := provider.Query(source, middleQuery)
	if result3.Error != nil {
		t.Fatalf("Middle wildcard query failed: %v", result3.Error)
	}

	expectedMiddleMatches := 2
	if len(result3.Matches) != expectedMiddleMatches {
		t.Errorf("Expected %d matches for 'get*Data', got %d", expectedMiddleMatches, len(result3.Matches))
	}
}

// Test edge cases in ExtractNodeName
func TestPHPProvider_ExtractNodeNameEdgeCases(t *testing.T) {
	provider := New()
	source := `<?php

// Property declaration with multiple variables
class TestClass {
	public $prop1, $prop2, $prop3;
	private static $staticProp;
	protected $protectedProp = 'default';

	// Method without name field (edge case)
	public function() {} // This should be invalid but test parsing

	// Method with complex name
	public function __construct() {}
	public function __destruct() {}
	public function __call($method, $args) {}
}

// Namespace use with aliases
use Some\Very\Long\Namespace\ClassName as ShortName;
use Another\Namespace\{ClassA, ClassB, ClassC};

// Variable declarations
$simpleVar = 'value';
$$dynamicVar = 'dynamic';
$complexVar = new SomeClass();

// Function with no name (anonymous - should not happen in global scope)
`

	// Test property queries
	propQuery := core.AgentQuery{Type: "variable"}
	propResult := provider.Query(source, propQuery)
	if propResult.Error != nil {
		t.Fatalf("Property query failed: %v", propResult.Error)
	}

	for _, match := range propResult.Matches {
		t.Logf("Found property/variable: %s", match.Name)
		// Most property names should not have $ prefix after extraction, but some edge cases might
		if strings.HasPrefix(match.Name, "$") {
			t.Logf("Note: Property name includes $ prefix: %s", match.Name)
		}
	}
}

// Test invalid transformation operations
func TestPHPProvider_InvalidTransformations(t *testing.T) {
	provider := New()
	source := `<?php

class TestClass {
	public function testMethod() {
		return 'test';
	}
}
`

	// Test unknown transformation method
	invalidOp := core.TransformOp{
		Method: "unknown_method",
		Target: core.AgentQuery{
			Type: "function",
			Name: "testMethod",
		},
		Replacement: "// replacement",
	}

	result := provider.Transform(source, invalidOp)
	if result.Error == nil {
		t.Error("Expected error for unknown transform method")
	}

	// Test transformation on non-existent target
	noTargetOp := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "nonExistentMethod",
		},
		Replacement: "// replacement",
	}

	result2 := provider.Transform(source, noTargetOp)
	if result2.Error == nil {
		t.Error("Expected error for transformation with no targets")
	}

	// Test transformation with empty target list
	emptyTargetOp := core.TransformOp{
		Method: "delete",
		Target: core.AgentQuery{
			Type: "class",
			Name: "NonExistentClass",
		},
	}

	result3 := provider.Transform(source, emptyTargetOp)
	if result3.Error == nil {
		t.Error("Expected error for transformation with no matching targets")
	}
}

// Test confidence scoring scenarios
func TestPHPProvider_ConfidenceScoring(t *testing.T) {
	provider := New()
	source := `<?php

class PublicAPI {
	public function publicMethod() {
		return 'public';
	}

	private function _privateMethod() {
		return 'private';
	}

	public function methodOne() {
		return 'one';
	}

	public function methodTwo() {
		return 'two';
	}

	public function methodThree() {
		return 'three';
	}

	public function methodFour() {
		return 'four';
	}

	public function methodFive() {
		return 'five';
	}

	public function methodSix() {
		return 'six';
	}
}
`

	// Test single target replacement (should have high confidence)
	singleTargetOp := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "publicMethod",
		},
		Replacement: "public function publicMethod() { return 'modified'; }",
	}

	result := provider.Transform(source, singleTargetOp)
	if result.Error != nil {
		t.Fatalf("Single target transform failed: %v", result.Error)
	}

	if result.Confidence.Score <= 0.8 {
		t.Errorf("Expected high confidence for single target, got %f", result.Confidence.Score)
	}

	// Test multiple targets replacement (should have lower confidence)
	multipleTargetOp := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "method*", // Matches 6 methods
		},
		Replacement: "public function replacedMethod() { return 'replaced'; }",
	}

	result2 := provider.Transform(source, multipleTargetOp)
	if result2.Error != nil {
		t.Fatalf("Multiple target transform failed: %v", result2.Error)
	}

	if result2.Confidence.Score >= 0.8 {
		t.Errorf("Expected lower confidence for multiple targets, got %f", result2.Confidence.Score)
	}

	// Test delete operation on exported method (should reduce confidence)
	deleteExportedOp := core.TransformOp{
		Method: "delete",
		Target: core.AgentQuery{
			Type: "function",
			Name: "publicMethod", // Public/exported method
		},
	}

	result3 := provider.Transform(source, deleteExportedOp)
	if result3.Error != nil {
		t.Fatalf("Delete exported transform failed: %v", result3.Error)
	}

	// Should have reduced confidence due to deleting exported API and being a delete operation
	if result3.Confidence.Score >= 0.95 {
		t.Errorf("Expected reduced confidence for deleting exported method, got %f", result3.Confidence.Score)
	}

	// Test wildcard pattern (should reduce confidence)
	wildcardOp := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "*Method", // Wildcard pattern
		},
		Replacement: "public function newMethod() { return 'new'; }",
	}

	result4 := provider.Transform(source, wildcardOp)
	if result4.Error != nil {
		t.Fatalf("Wildcard transform failed: %v", result4.Error)
	}

	// Should have reduced confidence due to wildcard pattern
	if result4.Confidence.Score >= 0.9 {
		t.Errorf("Expected reduced confidence for wildcard pattern, got %f", result4.Confidence.Score)
	}
}
