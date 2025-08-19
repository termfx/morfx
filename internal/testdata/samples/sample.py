"""
Sample Python file for testing cross-language functionality.
Contains various Python constructs that map to universal concepts.
"""

import os
import json
from typing import Optional, Dict, Any
from dataclasses import dataclass

# Constants following Python naming convention
DATABASE_URL = "postgres://localhost:5432/mydb"
API_VERSION = "v1.2.0"

# Global variables
global_counter = 0
is_initialized = False


@dataclass
class User:
    """User represents a system user with authentication data."""
    id: int
    name: str
    email: str
    password: Optional[str] = None

    def set_password(self, password: str) -> None:
        """Set the user password."""
        self.password = password

    def get_display_name(self) -> str:
        """Return formatted display name."""
        return f"{self.name} <{self.email}>"

    def is_valid(self) -> bool:
        """Check if user data is valid."""
        return validate_user(self)


class MockUser:
    """MockUser is a test user structure."""
    
    def __init__(self, user_id: int, name: str):
        self.id = user_id
        self.name = name


def new_user(name: str, email: str) -> User:
    """Create a new user instance."""
    return User(id=0, name=name, email=email)


def test_create_user():
    """Test user creation functionality."""
    user = new_user("John Doe", "john@example.com")
    assert user.name == "John Doe", "Name not set correctly"


def test_user_email():
    """Test user email validation."""
    user = User(id=1, name="Test", email="invalid-email")
    assert validate_user(user), "Email validation failed"


def validate_user(user: User) -> bool:
    """Perform user validation."""
    return len(user.email) > 0 and len(user.name) > 0


def main():
    """Main function demonstrating user functionality."""
    user = new_user("Admin", "admin@example.com")
    user.set_password("secret123")
    
    if user.is_valid():
        print(f"User is valid: {user.get_display_name()}")
    else:
        print("Invalid user data")
        os.exit(1)


if __name__ == "__main__":
    main()