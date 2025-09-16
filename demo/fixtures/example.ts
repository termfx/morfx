// TypeScript example for testing transformations

interface UserData {
    id: number;
    name: string;
    email: string;
    role: UserRole;
    createdAt: Date;
}

type UserRole = 'admin' | 'user' | 'moderator';

enum UserStatus {
    ACTIVE = 'active',
    INACTIVE = 'inactive',
    PENDING = 'pending',
    BANNED = 'banned'
}

class User implements UserData {
    public id: number;
    public name: string;
    public email: string;
    public role: UserRole;
    public createdAt: Date;
    private status: UserStatus = UserStatus.ACTIVE;

    constructor(data: Partial<UserData>) {
        this.id = data.id || 0;
        this.name = data.name || '';
        this.email = data.email || '';
        this.role = data.role || 'user';
        this.createdAt = data.createdAt || new Date();
    }

    public getDisplayName(): string {
        return `${this.name} (${this.role})`;
    }

    public updateEmail(newEmail: string): boolean {
        if (this.validateEmail(newEmail)) {
            this.email = newEmail;
            return true;
        }
        return false;
    }

    private validateEmail(email: string): boolean {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return emailRegex.test(email);
    }

    public getStatus(): UserStatus {
        return this.status;
    }
}

function createUser(name: string, email: string, role: UserRole = 'user'): User {
    return new User({ name, email, role, createdAt: new Date() });
}

export { User, UserData, UserRole, UserStatus, createUser };
