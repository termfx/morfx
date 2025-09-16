package python

import (
	"slices"
	"strings"
	"testing"

	"github.com/termfx/morfx/core"
)

func TestPythonProvider_New(t *testing.T) {
	provider := New()
	if provider == nil {
		t.Fatal("New returned nil")
	}
	if provider.Language() != "python" {
		t.Errorf("Expected language 'python', got '%s'", provider.Language())
	}
}

func TestPythonProvider_Language(t *testing.T) {
	provider := New()
	if provider.Language() != "python" {
		t.Errorf("Expected language 'python', got '%s'", provider.Language())
	}
}

func TestPythonProvider_Extensions(t *testing.T) {
	provider := New()
	extensions := provider.Extensions()

	expected := []string{".py", ".pyw", ".pyi"}
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

func TestPythonProvider_Query_Functions(t *testing.T) {
	provider := New()
	source := `
def get_user_data(user_id):
    return f"User {user_id}"

def process_user(user):
    return user.upper()

async def fetch_data():
    return await some_api_call()
`

	query := core.AgentQuery{
		Type: "function",
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	if len(result.Matches) != 3 {
		t.Errorf("Expected 3 matches, got %d", len(result.Matches))
	}

	// Should find all functions including async
	names := make([]string, len(result.Matches))
	for i, match := range result.Matches {
		names[i] = match.Name
	}

	foundGetUserData := false
	foundProcessUser := false
	foundFetchData := false
	for _, name := range names {
		if name == "get_user_data" {
			foundGetUserData = true
		}
		if name == "process_user" {
			foundProcessUser = true
		}
		if name == "fetch_data" {
			foundFetchData = true
		}
	}

	if !foundGetUserData {
		t.Error("Expected to find 'get_user_data' function")
	}
	if !foundProcessUser {
		t.Error("Expected to find 'process_user' function")
	}
	if !foundFetchData {
		t.Error("Expected to find 'fetch_data' async function")
	}
}

func TestPythonProvider_Query_Classes(t *testing.T) {
	provider := New()
	source := `
class User:
    def __init__(self, name):
        self.name = name
    
    def get_name(self):
        return self.name

class UserManager:
    def __init__(self):
        self.users = []
    
    def add_user(self, user):
        self.users.append(user)
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
	foundUserManager := false
	for _, name := range names {
		if name == "User" {
			foundUser = true
		}
		if name == "UserManager" {
			foundUserManager = true
		}
	}

	if !foundUser {
		t.Error("Expected to find 'User' class")
	}
	if !foundUserManager {
		t.Error("Expected to find 'UserManager' class")
	}
}

func TestPythonProvider_Query_Methods(t *testing.T) {
	provider := New()
	source := `
class Calculator:
    def add(self, a, b):
        return a + b
    
    def subtract(self, a, b):
        return a - b
    
    async def async_divide(self, a, b):
        return a / b
`

	query := core.AgentQuery{
		Type: "function", // Methods are also functions in Python
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	// Should find at least the methods
	if len(result.Matches) < 3 {
		t.Errorf("Expected at least 3 matches, got %d", len(result.Matches))
	}
}

func TestPythonProvider_Transform_Replace(t *testing.T) {
	provider := New()
	source := `
def greet(name):
    return f"Hello {name}"
`

	transform := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "greet",
		},
		Replacement: "def greet(name):\n    return f'Hi {name}'",
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

func TestPythonProvider_Transform_Delete(t *testing.T) {
	provider := New()
	source := `
def greet(name):
    return f"Hello {name}"

def farewell(name):
    return f"Goodbye {name}"
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

func TestPythonProvider_Validate(t *testing.T) {
	provider := New()

	// Test valid code
	validSource := `
class User:
    def __init__(self, name):
        self.name = name
    
    def get_name(self):
        return self.name

def process_user(user):
    return user.get_name().upper()

async def fetch_users():
    return await api_call()
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
func TestPythonProvider_ExtractNodeName(t *testing.T) {
	provider := New()
	source := `
import os
import sys
from datetime import datetime
from typing import List, Dict

# Function definition
def calculate_total(items: List[float]) -> float:
    """Calculate total of all items"""
    return sum(items)

# Async function
async def fetch_data(url: str) -> dict:
    """Fetch data from URL"""
    return {"data": "example"}

# Class definition
class UserManager:
    """Manages user operations"""
    
    def __init__(self, db_connection):
        self.db = db_connection
        self._cache = {}
    
    # Public method
    def create_user(self, name: str, email: str) -> dict:
        return {"name": name, "email": email}
    
    # Private method
    def _validate_email(self, email: str) -> bool:
        return "@" in email
    
    # Static method
    @staticmethod
    def hash_password(password: str) -> str:
        return f"hashed_{password}"
    
    # Class method
    @classmethod
    def from_config(cls, config: dict):
        return cls(config.get('db'))
    
    # Property
    @property
    def user_count(self) -> int:
        return len(self._cache)
    
    # Async method
    async def async_save(self, user_data: dict):
        await self._async_validate(user_data)
    
    # Dunder method
    def __str__(self) -> str:
        return f"UserManager({len(self._cache)} users)"

# Decorator function
def retry(max_attempts: int = 3):
    def decorator(func):
        def wrapper(*args, **kwargs):
            for attempt in range(max_attempts):
                try:
                    return func(*args, **kwargs)
                except Exception as e:
                    if attempt == max_attempts - 1:
                        raise e
            return None
        return wrapper
    return decorator

# Lambda functions
square = lambda x: x ** 2
add_numbers = lambda a, b: a + b

# Class with inheritance
class AdminUser(UserManager):
    def __init__(self, db_connection, permissions):
        super().__init__(db_connection)
        self.permissions = permissions
    
    def grant_access(self, resource: str) -> bool:
        return resource in self.permissions

# Variables
user_name = "John Doe" 
user_age = 30
is_active = True

# Global function with nested function
def process_data(data_list: List[str]) -> List[str]:
    def clean_item(item: str) -> str:
        return item.strip().lower()
    
    return [clean_item(item) for item in data_list]
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
		"calculate_total", "fetch_data", "__init__", "create_user",
		"_validate_email", "hash_password", "from_config", "user_count",
		"async_save", "__str__", "retry", "grant_access", "process_data", "clean_item",
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

	expectedClasses := []string{"UserManager", "AdminUser"}
	for _, expected := range expectedClasses {
		if !foundClasses[expected] {
			t.Errorf("Expected to find class '%s'", expected)
		}
	}

	// Test variable queries
	varQuery := core.AgentQuery{
		Type: "variable",
	}

	varResult := provider.Query(source, varQuery)
	if varResult.Error != nil {
		t.Fatalf("Variable query failed: %v", varResult.Error)
	}

	foundVars := make(map[string]bool)
	for _, match := range varResult.Matches {
		foundVars[match.Name] = true
		t.Logf("Found variable: %s", match.Name)
	}

	expectedVars := []string{"user_name", "user_age", "is_active", "square", "add_numbers"}
	foundAnyVar := false
	for _, expected := range expectedVars {
		if foundVars[expected] {
			foundAnyVar = true
			break
		}
	}

	if !foundAnyVar {
		t.Error("Expected to find at least one variable")
	}
}

// Test error handling and malformed code
func TestPythonProvider_ErrorHandling(t *testing.T) {
	provider := New()

	// Test malformed Python code
	malformedSource := `
class MalformedClass:
    def bad_method(self
        # Missing closing parenthesis and colon
        return "incomplete"
`

	result := provider.Validate(malformedSource)
	if result.Valid {
		t.Error("Expected malformed code to be invalid")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected validation errors for malformed code")
	}

	// Test query on malformed code
	query := core.AgentQuery{Type: "class"}
	queryResult := provider.Query(malformedSource, query)
	if queryResult.Error == nil {
		t.Error("Expected query error on malformed code")
	}
}

// Test complex Python constructs
func TestPythonProvider_ComplexConstructs(t *testing.T) {
	provider := New()
	source := `
#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Complex Python module for testing AST parsing.
Contains various Python constructs and patterns.
"""

import os
import sys
import asyncio
from typing import Dict, List, Optional, Union, TypeVar, Generic, Protocol
from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from enum import Enum, auto
from collections.abc import Iterable
from contextlib import contextmanager
import functools
import itertools
from pathlib import Path

# Type variables and aliases
T = TypeVar('T')
K = TypeVar('K')
V = TypeVar('V')
UserDict = Dict[str, Union[str, int, bool]]
HandlerFunc = callable[[str], bool]

# Enum classes
class Status(Enum):
    PENDING = auto()
    PROCESSING = auto()
    COMPLETED = auto()
    FAILED = auto()

class Priority(Enum):
    LOW = 1
    MEDIUM = 2
    HIGH = 3
    CRITICAL = 4

# Dataclass with complex fields
@dataclass
class User:
    id: int
    name: str
    email: str
    is_active: bool = True
    metadata: Dict[str, any] = field(default_factory=dict)
    tags: List[str] = field(default_factory=list)

    def __post_init__(self):
        if not self.email:
            raise ValueError("Email is required")

    @property
    def display_name(self) -> str:
        return f"{self.name} <{self.email}>"

    @classmethod
    def from_dict(cls, data: dict) -> 'User':
        return cls(**data)

    def to_dict(self) -> dict:
        return {
            'id': self.id,
            'name': self.name,
            'email': self.email,
            'is_active': self.is_active,
            'metadata': self.metadata,
            'tags': self.tags
        }

# Protocol (structural typing)
class Drawable(Protocol):
    def draw(self) -> None: ...
    def get_bounds(self) -> tuple[int, int, int, int]: ...

# Abstract base class
class BaseRepository(ABC, Generic[T]):
    """Abstract repository pattern implementation."""

    def __init__(self, connection_string: str):
        self.connection_string = connection_string
        self._cache: Dict[str, T] = {}

    @abstractmethod
    async def find_by_id(self, id: str) -> Optional[T]:
        """Find entity by ID."""
        pass

    @abstractmethod
    async def find_all(self) -> List[T]:
        """Find all entities."""
        pass

    @abstractmethod
    async def save(self, entity: T) -> T:
        """Save entity."""
        pass

    @abstractmethod
    async def delete(self, id: str) -> bool:
        """Delete entity by ID."""
        pass

    # Concrete method
    def clear_cache(self) -> None:
        """Clear internal cache."""
        self._cache.clear()

    @property
    def cache_size(self) -> int:
        return len(self._cache)

# Concrete repository implementation
class UserRepository(BaseRepository[User]):
    """User repository implementation."""

    def __init__(self, connection_string: str, pool_size: int = 10):
        super().__init__(connection_string)
        self.pool_size = pool_size
        self._connection_pool = None

    async def find_by_id(self, id: str) -> Optional[User]:
        """Find user by ID with caching."""
        if id in self._cache:
            return self._cache[id]

        # Simulate database lookup
        user_data = await self._fetch_user_from_db(id)
        if user_data:
            user = User.from_dict(user_data)
            self._cache[id] = user
            return user
        return None

    async def find_all(self) -> List[User]:
        """Find all users."""
        user_data_list = await self._fetch_all_users_from_db()
        return [User.from_dict(data) for data in user_data_list]

    async def save(self, entity: User) -> User:
        """Save user entity."""
        saved_data = await self._save_user_to_db(entity.to_dict())
        saved_user = User.from_dict(saved_data)
        self._cache[str(saved_user.id)] = saved_user
        return saved_user

    async def delete(self, id: str) -> bool:
        """Delete user by ID."""
        success = await self._delete_user_from_db(id)
        if success and id in self._cache:
            del self._cache[id]
        return success

    async def find_by_email(self, email: str) -> Optional[User]:
        """Find user by email address."""
        user_data = await self._fetch_user_by_email_from_db(email)
        return User.from_dict(user_data) if user_data else None

    async def find_active_users(self) -> List[User]:
        """Find all active users."""
        all_users = await self.find_all()
        return [user for user in all_users if user.is_active]

    # Private methods
    async def _fetch_user_from_db(self, id: str) -> Optional[dict]:
        """Simulate database fetch."""
        await asyncio.sleep(0.1)  # Simulate network delay
        return {'id': int(id), 'name': f'User {id}', 'email': f'user{id}@example.com'}

    async def _fetch_all_users_from_db(self) -> List[dict]:
        """Simulate fetching all users."""
        await asyncio.sleep(0.2)
        return [{'id': i, 'name': f'User {i}', 'email': f'user{i}@example.com'} for i in range(1, 11)]

    async def _save_user_to_db(self, user_data: dict) -> dict:
        """Simulate saving to database."""
        await asyncio.sleep(0.1)
        return user_data

    async def _delete_user_from_db(self, id: str) -> bool:
        """Simulate deletion from database."""
        await asyncio.sleep(0.1)
        return True

    async def _fetch_user_by_email_from_db(self, email: str) -> Optional[dict]:
        """Simulate fetch by email."""
        await asyncio.sleep(0.1)
        return {'id': 1, 'name': 'User', 'email': email}

# Service class with dependency injection
class UserService:
    """User service with business logic."""

    def __init__(self, repository: UserRepository, logger=None):
        self.repository = repository
        self.logger = logger or self._create_default_logger()
        self._event_handlers: List[HandlerFunc] = []

    async def get_user(self, user_id: str) -> Optional[User]:
        """Get user by ID with logging."""
        self.logger.info(f"Fetching user {user_id}")
        try:
            user = await self.repository.find_by_id(user_id)
            if user:
                self.logger.info(f"Found user {user.name}")
                await self._notify_handlers(f"user_fetched:{user_id}")
            else:
                self.logger.warning(f"User {user_id} not found")
            return user
        except Exception as e:
            self.logger.error(f"Error fetching user {user_id}: {e}")
            raise

    async def create_user(self, user_data: dict) -> User:
        """Create new user with validation."""
        self.logger.info(f"Creating user {user_data.get('name')}")

        # Validation
        if not self._validate_user_data(user_data):
            raise ValueError("Invalid user data")

        user = User.from_dict(user_data)
        saved_user = await self.repository.save(user)

        self.logger.info(f"Created user {saved_user.id}")
        await self._notify_handlers(f"user_created:{saved_user.id}")

        return saved_user

    async def update_user(self, user_id: str, updates: dict) -> Optional[User]:
        """Update existing user."""
        user = await self.repository.find_by_id(user_id)
        if not user:
            return None

        # Apply updates
        for key, value in updates.items():
            if hasattr(user, key):
                setattr(user, key, value)

        updated_user = await self.repository.save(user)
        await self._notify_handlers(f"user_updated:{user_id}")
        return updated_user

    async def delete_user(self, user_id: str) -> bool:
        """Delete user by ID."""
        success = await self.repository.delete(user_id)
        if success:
            await self._notify_handlers(f"user_deleted:{user_id}")
        return success

    def add_event_handler(self, handler: HandlerFunc) -> None:
        """Add event handler."""
        self._event_handlers.append(handler)

    def remove_event_handler(self, handler: HandlerFunc) -> None:
        """Remove event handler."""
        if handler in self._event_handlers:
            self._event_handlers.remove(handler)

    async def _notify_handlers(self, event: str) -> None:
        """Notify all event handlers."""
        for handler in self._event_handlers:
            try:
                handler(event)
            except Exception as e:
                self.logger.error(f"Error in event handler: {e}")

    def _validate_user_data(self, data: dict) -> bool:
        """Validate user data."""
        required_fields = ['name', 'email']
        return all(field in data and data[field] for field in required_fields)

    def _create_default_logger(self):
        """Create default logger."""
        import logging
        logger = logging.getLogger(__name__)
        logger.setLevel(logging.INFO)
        return logger

# Utility functions with decorators
@functools.lru_cache(maxsize=128)
def expensive_computation(n: int) -> int:
    """Expensive computation with caching."""
    if n <= 1:
        return n
    return expensive_computation(n-1) + expensive_computation(n-2)

@contextmanager
def database_transaction():
    """Context manager for database transactions."""
    print("Starting transaction")
    try:
        yield
        print("Committing transaction")
    except Exception as e:
        print(f"Rolling back transaction: {e}")
        raise
    finally:
        print("Closing transaction")

def retry(max_attempts: int = 3, delay: float = 1.0):
    """Retry decorator."""
    def decorator(func):
        @functools.wraps(func)
        async def async_wrapper(*args, **kwargs):
            for attempt in range(max_attempts):
                try:
                    return await func(*args, **kwargs)
                except Exception as e:
                    if attempt == max_attempts - 1:
                        raise e
                    await asyncio.sleep(delay)
            return None

        @functools.wraps(func)
        def sync_wrapper(*args, **kwargs):
            for attempt in range(max_attempts):
                try:
                    return func(*args, **kwargs)
                except Exception as e:
                    if attempt == max_attempts - 1:
                        raise e
                    time.sleep(delay)
            return None

        # Return appropriate wrapper based on function type
        return async_wrapper if asyncio.iscoroutinefunction(func) else sync_wrapper
    return decorator

# Generator functions
def fibonacci_generator(n: int):
    """Generate Fibonacci sequence."""
    a, b = 0, 1
    for _ in range(n):
        yield a
        a, b = b, a + b

async def async_data_generator(data_source: Iterable[str]):
    """Async generator for processing data."""
    for item in data_source:
        await asyncio.sleep(0.01)  # Simulate async processing
        yield item.upper()

# Lambda functions and functional programming
filter_active_users = lambda users: [u for u in users if u.is_active]
map_user_names = lambda users: [u.name for u in users]
reduce_user_ages = lambda users: functools.reduce(lambda acc, u: acc + getattr(u, 'age', 0), users, 0)

# Complex list comprehensions and generator expressions
def process_user_data(users: List[User]) -> Dict[str, List[str]]:
    """Process user data with comprehensions."""
    # Nested list comprehension
    user_emails_by_domain = {
        domain: [u.email for u in users if u.email.endswith(f'@{domain}')]
        for domain in {u.email.split('@')[1] for u in users if '@' in u.email}
    }

    # Generator expression with filtering
    active_user_names = (u.name for u in users if u.is_active and len(u.name) > 3)

    # Dictionary comprehension with conditions
    user_metadata_summary = {
        u.id: len(u.metadata)
        for u in users
        if u.metadata and u.is_active
    }

    return {
        'emails_by_domain': user_emails_by_domain,
        'active_names': list(active_user_names),
        'metadata_summary': user_metadata_summary
    }

# Exception handling and custom exceptions
class UserServiceError(Exception):
    """Base exception for user service."""
    pass

class UserNotFoundError(UserServiceError):
    """Exception for when user is not found."""

    def __init__(self, user_id: str):
        self.user_id = user_id
        super().__init__(f"User {user_id} not found")

class UserValidationError(UserServiceError):
    """Exception for user validation errors."""

    def __init__(self, field: str, message: str):
        self.field = field
        self.message = message
        super().__init__(f"Validation error for {field}: {message}")

# Global variables and constants
DEFAULT_PAGE_SIZE = 50
MAX_RETRIES = 3
API_VERSION = "v1"
CONFIG = {
    'database_url': 'postgresql://localhost:5432/testdb',
    'redis_url': 'redis://localhost:6379',
    'debug': True
}

# Module-level functions
def setup_logging(level: str = 'INFO') -> None:
    """Setup logging configuration."""
    import logging
    logging.basicConfig(
        level=getattr(logging, level.upper()),
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )

def main() -> None:
    """Main entry point."""
    setup_logging()
    print("Starting application...")

    # Example usage
    async def run_example():
        repo = UserRepository(CONFIG['database_url'])
        service = UserService(repo)

        # Create and fetch user
        user_data = {'name': 'John Doe', 'email': 'john@example.com'}
        user = await service.create_user(user_data)
        fetched_user = await service.get_user(str(user.id))

        print(f"Created and fetched user: {fetched_user.display_name}")

    # Run async example
    asyncio.run(run_example())

if __name__ == '__main__':
    main()
`

	// Test complex function queries
	funcQuery := core.AgentQuery{Type: "function"}
	funcResult := provider.Query(source, funcQuery)
	if funcResult.Error != nil {
		t.Fatalf("Function query failed: %v", funcResult.Error)
	}

	expectedFunctions := []string{
		"__post_init__", "from_dict", "to_dict", "find_by_id", "find_all", "save", "delete",
		"clear_cache", "__init__", "find_by_email", "find_active_users", "get_user",
		"create_user", "update_user", "delete_user", "add_event_handler", "expensive_computation",
		"database_transaction", "retry", "fibonacci_generator", "async_data_generator",
		"process_user_data", "setup_logging", "main", "run_example",
	}

	foundFunctions := make(map[string]bool)
	for _, match := range funcResult.Matches {
		foundFunctions[match.Name] = true
		t.Logf("Found function: %s", match.Name)
	}

	// Check that we found at least some expected functions
	foundCount := 0
	for _, expected := range expectedFunctions {
		if foundFunctions[expected] {
			foundCount++
		}
	}

	if foundCount < 10 {
		t.Errorf("Expected to find at least 10 functions, found %d", foundCount)
	}

	// Test class queries
	classQuery := core.AgentQuery{Type: "class"}
	classResult := provider.Query(source, classQuery)
	if classResult.Error != nil {
		t.Fatalf("Class query failed: %v", classResult.Error)
	}

	expectedClasses := []string{
		"Status",
		"Priority",
		"User",
		"BaseRepository",
		"UserRepository",
		"UserService",
		"UserServiceError",
		"UserNotFoundError",
		"UserValidationError",
	}
	foundClasses := make(map[string]bool)
	for _, match := range classResult.Matches {
		foundClasses[match.Name] = true
		t.Logf("Found class: %s", match.Name)
	}

	foundClassCount := 0
	for _, expected := range expectedClasses {
		if foundClasses[expected] {
			foundClassCount++
		}
	}

	if foundClassCount < 5 {
		t.Errorf("Expected to find at least 5 classes, found %d", foundClassCount)
	}

	// Test import queries
	importQuery := core.AgentQuery{Type: "import"}
	importResult := provider.Query(source, importQuery)
	if importResult.Error != nil {
		t.Fatalf("Import query failed: %v", importResult.Error)
	}

	for _, match := range importResult.Matches {
		t.Logf("Found import: %s", match.Name)
	}

	// Test decorator queries
	decoratorQuery := core.AgentQuery{Type: "decorator"}
	decoratorResult := provider.Query(source, decoratorQuery)
	if decoratorResult.Error != nil {
		t.Fatalf("Decorator query failed: %v", decoratorResult.Error)
	}

	for _, match := range decoratorResult.Matches {
		t.Logf("Found decorator: %s", match.Name)
	}
}

// Test complex transformations
func TestPythonProvider_ComplexTransformations(t *testing.T) {
	provider := New()
	source := `
class DataProcessor:
    def __init__(self, config):
        self.config = config

    def process_data(self, data):
        return [item.upper() for item in data]

    def validate_data(self, data):
        return all(isinstance(item, str) for item in data)

    async def async_process(self, data):
        await asyncio.sleep(0.1)
        return self.process_data(data)
`

	// Test insert_before transformation
	insertBeforeOp := core.TransformOp{
		Method: "insert_before",
		Target: core.AgentQuery{
			Type: "function",
			Name: "validate_data",
		},
		Content: "    def clean_data(self, data):\n        \"\"\"Clean input data.\"\"\"\n        return [item.strip() for item in data if item.strip()]",
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
			Name: "process_data",
		},
		Content: "\n    def process_data_advanced(self, data, transform_func=None):\n        \"\"\"Advanced data processing with optional transform.\"\"\"\n        processed = self.process_data(data)\n        if transform_func:\n            processed = [transform_func(item) for item in processed]\n        return processed",
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
			Name: "DataProcessor",
		},
		Content: "\n    @property\n    def status(self):\n        \"\"\"Get processor status.\"\"\"\n        return 'active'\n    \n    def shutdown(self):\n        \"\"\"Shutdown the processor.\"\"\"\n        self.config = None",
	}

	result3 := provider.Transform(source, appendOp)
	if result3.Error != nil {
		t.Fatalf("Append transform failed: %v", result3.Error)
	}

	if result3.Modified == "" {
		t.Error("Expected modified code for append")
	}
}

// Test pattern matching with Python-specific patterns
func TestPythonProvider_PatternMatching(t *testing.T) {
	provider := New()
	source := `
class UserManager:
    def get_user_data(self, user_id):
        return {'id': user_id}

    def get_user_profile(self, user_id):
        return {'profile': 'data'}

    def get_admin_data(self, admin_id):
        return {'admin': admin_id}

    def set_user_active(self, user_id):
        pass

    def _get_internal_data(self, id):
        return {'internal': id}

    def __get_private_data(self, id):
        return {'private': id}
`

	// Test wildcard pattern matching
	getUserQuery := core.AgentQuery{
		Type: "function",
		Name: "get_user*", // Should match get_user_data and get_user_profile
	}

	result := provider.Query(source, getUserQuery)
	if result.Error != nil {
		t.Fatalf("Wildcard query failed: %v", result.Error)
	}

	// Should find get_user_data and get_user_profile, but not get_admin_data
	expectedMatches := 2
	if len(result.Matches) != expectedMatches {
		t.Errorf("Expected %d matches for 'get_user*', got %d", expectedMatches, len(result.Matches))
	}

	for _, match := range result.Matches {
		if !strings.HasPrefix(match.Name, "get_user") {
			t.Errorf("Unexpected match '%s' for pattern 'get_user*'", match.Name)
		}
	}

	// Test suffix wildcard
	dataQuery := core.AgentQuery{
		Type: "function",
		Name: "*_data", // Should match get_user_data, get_admin_data, etc.
	}

	result2 := provider.Query(source, dataQuery)
	if result2.Error != nil {
		t.Fatalf("Suffix wildcard query failed: %v", result2.Error)
	}

	if len(result2.Matches) < 2 {
		t.Errorf("Expected at least 2 matches for '*_data', got %d", len(result2.Matches))
	}

	// Test private method pattern
	privateQuery := core.AgentQuery{
		Type: "function",
		Name: "_*", // Should match private methods
	}

	result3 := provider.Query(source, privateQuery)
	if result3.Error != nil {
		t.Fatalf("Private method query failed: %v", result3.Error)
	}

	for _, match := range result3.Matches {
		if !strings.HasPrefix(match.Name, "_") {
			t.Errorf("Expected private method name to start with underscore: %s", match.Name)
		}
	}
}

// Test edge cases in ExtractNodeName
func TestPythonProvider_ExtractNodeNameEdgeCases(t *testing.T) {
	provider := New()
	source := `
# Multiple assignment
a, b, c = 1, 2, 3
x = y = z = 42

# Augmented assignment
counter += 1
data *= 2

# Global and nonlocal statements
global global_var
nonlocal outer_var

# Import statements with various forms
import os
import sys as system
from pathlib import Path
from typing import Dict, List, Optional
from collections.abc import Iterable, Mapping
from . import local_module
from ..parent import parent_module
from .sibling import function as imported_func

# Function with complex decorators
@property
@functools.lru_cache(maxsize=128)
@classmethod
def cached_class_method(cls):
    return 'cached'

@staticmethod
@retry(max_attempts=3)
def static_retry_method():
    return 'retry'

@functools.wraps(original_func)
@app.route('/api/users')
@auth.requires_permission('read:users')
async def api_endpoint(request):
    return 'api response'

# Class with complex inheritance
class MultipleInheritance(BaseClass, MixinClass, ABC):
    pass

# Lambda assignments
square = lambda x: x ** 2
filter_func = lambda items: [item for item in items if item > 0]
map_func = lambda data: {k: v.upper() for k, v in data.items()}

# Nested function definitions
def outer_function(param):
    def inner_function(inner_param):
        def deeply_nested(deep_param):
            return f"{param}-{inner_param}-{deep_param}"
        return deeply_nested
    return inner_function

# Generator with complex expressions
def complex_generator(data):
    for item in data:
        if isinstance(item, dict):
            for key, value in item.items():
                if key.startswith('_'):
                    continue
                yield f"{key}:{value}"
        elif isinstance(item, (list, tuple)):
            for sub_item in item:
                yield str(sub_item)
        else:
            yield str(item)
`

	// Test variable queries with complex assignments
	varQuery := core.AgentQuery{Type: "variable"}
	varResult := provider.Query(source, varQuery)
	if varResult.Error != nil {
		t.Fatalf("Variable query failed: %v", varResult.Error)
	}

	for _, match := range varResult.Matches {
		t.Logf("Found variable: %s", match.Name)
	}

	// Test import queries
	importQuery := core.AgentQuery{Type: "import"}
	importResult := provider.Query(source, importQuery)
	if importResult.Error != nil {
		t.Fatalf("Import query failed: %v", importResult.Error)
	}

	for _, match := range importResult.Matches {
		t.Logf("Found import: %s", match.Name)
	}

	// Test decorator queries
	decoratorQuery := core.AgentQuery{Type: "decorator"}
	decoratorResult := provider.Query(source, decoratorQuery)
	if decoratorResult.Error != nil {
		t.Fatalf("Decorator query failed: %v", decoratorResult.Error)
	}

	for _, match := range decoratorResult.Matches {
		t.Logf("Found decorator: %s", match.Name)
	}
}

// Test invalid transformations
func TestPythonProvider_InvalidTransformations(t *testing.T) {
	provider := New()
	source := `
class TestService:
    def test_method(self):
        return 'test'
`

	// Test unknown transformation method
	invalidOp := core.TransformOp{
		Method: "unknown_method",
		Target: core.AgentQuery{
			Type: "function",
			Name: "test_method",
		},
		Replacement: "# replacement",
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
			Name: "non_existent_method",
		},
		Replacement: "# replacement",
	}

	result2 := provider.Transform(source, noTargetOp)
	if result2.Error == nil {
		t.Error("Expected error for transformation with no targets")
	}
}

// Test confidence scoring with Python-specific scenarios
func TestPythonProvider_ConfidenceScoring(t *testing.T) {
	provider := New()
	source := `
class PublicAPI:
    def public_method(self):
        """Public API method."""
        return 'public'

    def _private_method(self):
        """Private method (single underscore)."""
        return 'private'

    def __dunder_method__(self):
        """Dunder method (double underscore)."""
        return 'dunder'

    def method_one(self): pass
    def method_two(self): pass
    def method_three(self): pass
    def method_four(self): pass
    def method_five(self): pass
    def method_six(self): pass
`

	// Test deleting public method (should reduce confidence)
	deletePublicOp := core.TransformOp{
		Method: "delete",
		Target: core.AgentQuery{
			Type: "function",
			Name: "public_method", // Public method (no underscore prefix)
		},
	}

	result := provider.Transform(source, deletePublicOp)
	if result.Error != nil {
		t.Fatalf("Delete public transform failed: %v", result.Error)
	}

	// Should have reduced confidence due to deleting public API
	if result.Confidence.Score >= 0.7 {
		t.Errorf("Expected reduced confidence for deleting public method, got %f", result.Confidence.Score)
	}

	// Test replacing private method (should have higher confidence)
	replacePrivateOp := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "_private_method", // Private method
		},
		Replacement: "    def _private_method(self):\n        return 'modified private'",
	}

	result2 := provider.Transform(source, replacePrivateOp)
	if result2.Error != nil {
		t.Fatalf("Replace private transform failed: %v", result2.Error)
	}

	// Should have higher confidence since it's not exported
	if result2.Confidence.Score <= 0.8 {
		t.Errorf("Expected higher confidence for modifying private method, got %f", result2.Confidence.Score)
	}

	// Test wildcard operation affecting many methods
	wildcardOp := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "method_*", // Affects 6 methods
		},
		Replacement: "    def replacement_method(self): pass",
	}

	result3 := provider.Transform(source, wildcardOp)
	if result3.Error != nil {
		t.Fatalf("Wildcard transform failed: %v", result3.Error)
	}

	// Should have reduced confidence due to affecting many targets
	if result3.Confidence.Score >= 0.8 {
		t.Errorf(
			"Expected reduced confidence for wildcard affecting multiple methods, got %f",
			result3.Confidence.Score,
		)
	}
}
