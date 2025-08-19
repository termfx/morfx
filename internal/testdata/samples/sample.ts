/**
 * Sample TypeScript file for testing cross-language functionality.
 * Contains various TypeScript constructs that map to universal concepts.
 */

import * as fs from 'fs';
import * as path from 'path';

// Type definitions
interface UserData {
    id: number;
    name: string;
    email: string;
    password?: string;
}

// Constants
const DATABASE_URL: string = "postgres://localhost:5432/mydb";
const API_VERSION: string = "v1.2.0";

// Global variables
let globalCounter: number = 0;
var isInitialized: boolean = false;

/**
 * User class represents a system user with authentication data
 */
class User implements UserData {
    public id: number;
    public name: string;
    public email: string;
    private password?: string;

    constructor(id: number, name: string, email: string) {
        this.id = id;
        this.name = name;
        this.email = email;
    }

    /**
     * Set the user password
     */
    setPassword(password: string): void {
        this.password = password;
    }

    /**
     * Return formatted display name
     */
    getDisplayName(): string {
        return `${this.name} <${this.email}>`;
    }

    /**
     * Check if user data is valid
     */
    isValid(): boolean {
        return validateUser(this);
    }

    /**
     * Get user as JSON object
     */
    toJSON(): UserData {
        return {
            id: this.id,
            name: this.name,
            email: this.email,
            password: this.password
        };
    }
}

/**
 * MockUser is a test user structure
 */
class MockUser {
    constructor(
        public readonly id: number,
        public readonly name: string
    ) {}
}

/**
 * Create a new user instance
 */
function newUser(name: string, email: string): User {
    return new User(0, name, email);
}

/**
 * Test user creation functionality
 */
function testCreateUser(): void {
    const user = newUser("John Doe", "john@example.com");
    console.assert(user.name === "John Doe", "Name not set correctly");
}

/**
 * Test user email validation
 */
function testUserEmail(): void {
    const user = new User(1, "Test", "invalid-email");
    console.assert(validateUser(user), "Email validation failed");
}

/**
 * Perform user validation
 */
function validateUser(user: UserData): boolean {
    return user.email.length > 0 && user.name.length > 0;
}

/**
 * Arrow function for user processing with type annotation
 */
const processUser = (user: User): string => {
    if (user.isValid()) {
        return user.getDisplayName();
    }
    return "Invalid user";
};

/**
 * Generic function for data processing
 */
function processData<T>(data: T[], processor: (item: T) => string): string[] {
    return data.map(processor);
}

/**
 * Enum for user roles
 */
enum UserRole {
    ADMIN = "admin",
    USER = "user",
    GUEST = "guest"
}

/**
 * Type alias for user processing function
 */
type UserProcessor = (user: User) => string;

/**
 * Main function demonstrating user functionality
 */
function main(): void {
    const user = newUser("Admin", "admin@example.com");
    user.setPassword("secret123");
    
    if (user.isValid()) {
        console.log(`User is valid: ${user.getDisplayName()}`);
    } else {
        console.log("Invalid user data");
        process.exit(1);
    }
}

// Export for testing
export { 
    User, 
    MockUser, 
    UserData, 
    UserRole, 
    UserProcessor,
    newUser, 
    validateUser, 
    processUser, 
    processData 
};

// Run main if this is the entry point
if (require.main === module) {
    main();
}