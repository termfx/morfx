// JavaScript example for testing transformations

const API_BASE_URL = 'https://api.example.com';
const DEFAULT_TIMEOUT = 5000;

let userCache = new Map();
let isInitialized = false;

/**
 * User class representing a user entity
 */
class User {
    constructor(id, name, email) {
        this.id = id;
        this.name = name;
        this.email = email;
        this.createdAt = new Date();
    }

    getDisplayName() {
        return `${this.name} <${this.email}>`;
    }

    updateEmail(newEmail) {
        if (validateEmail(newEmail)) {
            this.email = newEmail;
            return true;
        }
        return false;
    }
}

/**
 * Creates a new user instance
 */
function createUser(name, email) {
    if (!name || !email) {
        throw new Error('Name and email are required');
    }
    
    const id = generateUserId();
    return new User(id, name, email);
}

/**
 * Validates email format
 */
function validateEmail(email) {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
}

/**
 * Generates a unique user ID
 */
function generateUserId() {
    return Math.random().toString(36).substr(2, 9);
}
