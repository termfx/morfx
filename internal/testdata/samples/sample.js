/**
 * Sample JavaScript file for testing cross-language functionality.
 * Contains various JavaScript constructs that map to universal concepts.
 */

import fs from 'fs';
import path from 'path';

// Constants
const DATABASE_URL = "postgres://localhost:5432/mydb";
const API_VERSION = "v1.2.0";

// Global variables
let globalCounter = 0;
var isInitialized = false;

/**
 * User class represents a system user with authentication data
 */
class User {
    constructor(id, name, email) {
        this.id = id;
        this.name = name;
        this.email = email;
        this.password = null;
    }

    /**
     * Set the user password
     * @param {string} password - The password to set
     */
    setPassword(password) {
        this.password = password;
    }

    /**
     * Return formatted display name
     * @returns {string} The formatted display name
     */
    getDisplayName() {
        return `${this.name} <${this.email}>`;
    }

    /**
     * Check if user data is valid
     * @returns {boolean} True if user is valid
     */
    isValid() {
        return validateUser(this);
    }
}

/**
 * MockUser is a test user structure
 */
class MockUser {
    constructor(id, name) {
        this.id = id;
        this.name = name;
    }
}

/**
 * Create a new user instance
 * @param {string} name - User name
 * @param {string} email - User email
 * @returns {User} New user instance
 */
function newUser(name, email) {
    return new User(0, name, email);
}

/**
 * Test user creation functionality
 */
function testCreateUser() {
    const user = newUser("John Doe", "john@example.com");
    console.assert(user.name === "John Doe", "Name not set correctly");
}

/**
 * Test user email validation
 */
function testUserEmail() {
    const user = new User(1, "Test", "invalid-email");
    console.assert(validateUser(user), "Email validation failed");
}

/**
 * Perform user validation
 * @param {User} user - User to validate
 * @returns {boolean} True if valid
 */
function validateUser(user) {
    return user.email.length > 0 && user.name.length > 0;
}

/**
 * Arrow function for user processing
 */
const processUser = (user) => {
    if (user.isValid()) {
        return user.getDisplayName();
    }
    return "Invalid user";
};

/**
 * Main function demonstrating user functionality
 */
function main() {
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
export { User, MockUser, newUser, validateUser, processUser };

// Run main if this is the entry point
if (import.meta.url === `file://${process.argv[1]}`) {
    main();
}