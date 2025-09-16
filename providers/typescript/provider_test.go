package typescript

import (
	"slices"
	"strings"
	"testing"

	"github.com/termfx/morfx/core"
)

func TestTypeScriptProvider_New(t *testing.T) {
	provider := New()
	if provider == nil {
		t.Fatal("New returned nil")
	}
	if provider.Language() != "typescript" {
		t.Errorf("Expected language 'typescript', got '%s'", provider.Language())
	}
}

func TestTypeScriptProvider_Language(t *testing.T) {
	provider := New()
	if provider.Language() != "typescript" {
		t.Errorf("Expected language 'typescript', got '%s'", provider.Language())
	}
}

func TestTypeScriptProvider_Extensions(t *testing.T) {
	provider := New()
	extensions := provider.Extensions()

	expected := []string{".ts", ".tsx", ".d.ts"}
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

func TestTypeScriptProvider_Query_Functions(t *testing.T) {
	provider := New()
	source := `
function getUserData(id: number): string {
    return "User " + id;
}

const processUser = (user: string): string => {
    return user.toUpperCase();
};

async function fetchData(): Promise<any> {
    return await fetch('/api/data');
}
`

	query := core.AgentQuery{
		Type: "function",
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	// Should find at least one function
	if len(result.Matches) < 1 {
		t.Errorf("Expected at least 1 match, got %d", len(result.Matches))
	}

	// Should find getUserData function
	names := make([]string, len(result.Matches))
	for i, match := range result.Matches {
		names[i] = match.Name
	}

	foundGetUserData := false
	for _, name := range names {
		if name == "getUserData" {
			foundGetUserData = true
		}
	}

	if !foundGetUserData {
		t.Error("Expected to find 'getUserData' function")
	}
}

func TestTypeScriptProvider_Query_Classes(t *testing.T) {
	provider := New()
	source := `
class User {
    private name: string;
    
    constructor(name: string) {
        this.name = name;
    }
    
    getName(): string {
        return this.name;
    }
}

class UserService {
    private users: User[] = [];
    
    addUser(user: User): void {
        this.users.push(user);
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
	foundUserService := false
	for _, name := range names {
		if name == "User" {
			foundUser = true
		}
		if name == "UserService" {
			foundUserService = true
		}
	}

	if !foundUser {
		t.Error("Expected to find 'User' class")
	}
	if !foundUserService {
		t.Error("Expected to find 'UserService' class")
	}
}

func TestTypeScriptProvider_Query_Interfaces(t *testing.T) {
	provider := New()
	source := `
interface IUser {
    id: number;
    name: string;
    email: string;
}

interface IUserService {
    getUser(id: number): IUser;
    createUser(user: IUser): void;
}
`

	query := core.AgentQuery{
		Type: "interface",
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	if len(result.Matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(result.Matches))
	}
}

func TestTypeScriptProvider_Query_Types(t *testing.T) {
	provider := New()
	source := `
type UserType = {
    id: number;
    name: string;
};

type StatusType = 'active' | 'inactive' | 'pending';
`

	query := core.AgentQuery{
		Type: "type",
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	if len(result.Matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(result.Matches))
	}
}

func TestTypeScriptProvider_Transform_Replace(t *testing.T) {
	provider := New()
	source := `
function greet(name: string): string {
    return "Hello " + name;
}
`

	transform := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "greet",
		},
		Replacement: "function greet(name: string): string {\n    return 'Hi ' + name;\n}",
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

func TestTypeScriptProvider_Transform_Delete(t *testing.T) {
	provider := New()
	source := `
function greet(name: string): string {
    return "Hello " + name;
}

function farewell(name: string): string {
    return "Goodbye " + name;
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

func TestTypeScriptProvider_Validate(t *testing.T) {
	provider := New()

	// Test valid code
	validSource := `
interface IUser {
    id: number;
    name: string;
}

class UserService {
    private users: IUser[] = [];
    
    addUser(user: IUser): void {
        this.users.push(user);
    }
    
    getUser(id: number): IUser | undefined {
        return this.users.find(user => user.id === id);
    }
}

const service = new UserService();
`

	result := provider.Validate(validSource)
	if !result.Valid {
		t.Errorf("Expected valid code to be valid, got errors: %v", result.Errors)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors for valid code, got %d errors", len(result.Errors))
	}
}

// Simple test to improve ExtractNodeName coverage
func TestTypeScriptProvider_ExtractNodeNameSimple(t *testing.T) {
	provider := New()

	// Test basic function
	funcSource := `function testFunc() { return 42; }`
	funcQuery := core.AgentQuery{Type: "function"}
	funcResult := provider.Query(funcSource, funcQuery)

	if funcResult.Error != nil {
		t.Fatalf("Function query failed: %v", funcResult.Error)
	}

	if len(funcResult.Matches) > 0 {
		t.Logf("Found function: %s", funcResult.Matches[0].Name)
	}

	// Test basic class
	classSource := `class TestClass { constructor() {} }`
	classQuery := core.AgentQuery{Type: "class"}
	classResult := provider.Query(classSource, classQuery)

	if classResult.Error != nil {
		t.Fatalf("Class query failed: %v", classResult.Error)
	}

	if len(classResult.Matches) > 0 {
		t.Logf("Found class: %s", classResult.Matches[0].Name)
	}

	// Test interface
	interfaceSource := `interface ITest { prop: string; }`
	interfaceQuery := core.AgentQuery{Type: "interface"}
	interfaceResult := provider.Query(interfaceSource, interfaceQuery)

	if interfaceResult.Error != nil {
		t.Fatalf("Interface query failed: %v", interfaceResult.Error)
	}

	if len(interfaceResult.Matches) > 0 {
		t.Logf("Found interface: %s", interfaceResult.Matches[0].Name)
	}

	// Test type alias
	typeSource := `type MyType = string | number;`
	typeQuery := core.AgentQuery{Type: "type"}
	typeResult := provider.Query(typeSource, typeQuery)

	if typeResult.Error != nil {
		t.Fatalf("Type query failed: %v", typeResult.Error)
	}

	if len(typeResult.Matches) > 0 {
		t.Logf("Found type: %s", typeResult.Matches[0].Name)
	}
}

// Test error handling and malformed code
func TestTypeScriptProvider_ErrorHandling(t *testing.T) {
	provider := New()

	// Test malformed TypeScript code
	malformedSource := `
	class MalformedClass {
		constructor(public name: string
		// Missing closing parenthesis and brace
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

// Test complex TypeScript constructs
func TestTypeScriptProvider_ComplexConstructs(t *testing.T) {
	provider := New()
	source := `
import { Observable, Subject } from 'rxjs';
import type { User, UserProfile } from './types';
import * as utils from './utils';
export { ApiClient } from './api';
export type { RequestOptions } from './types';

// Interface with generics
interface Repository<T, K = string> {
	findById(id: K): Promise<T | null>;
	findAll(): Promise<T[]>;
	create(entity: T): Promise<T>;
	update(id: K, updates: Partial<T>): Promise<T>;
	delete(id: K): Promise<boolean>;
}

// Type aliases
type UserID = string;
type UserStatus = 'active' | 'inactive' | 'pending';
type EventHandler<T> = (event: T) => void;
type ApiResponse<T> = {
	data: T;
	status: number;
	message?: string;
};

// Enum declaration
enum UserRole {
	ADMIN = 'admin',
	MODERATOR = 'moderator',
	USER = 'user',
	GUEST = 'guest'
}

// Class with generics and decorators
@injectable()
@singleton()
class UserService<T extends User = User> implements Repository<T, UserID> {
	private readonly _users: Map<UserID, T> = new Map();
	private _eventSubject = new Subject<UserEvent>();

	// Constructor with parameter properties
	constructor(
		private readonly apiClient: ApiClient,
		private logger: Logger,
		public readonly config: ServiceConfig
	) {}

	// Method with generics
	async findById(id: UserID): Promise<T | null> {
		try {
			const cached = this._users.get(id);
			if (cached) return cached;

			const response = await this.apiClient.get<T>('/users/' + id);
			return response.data;
		} catch (error) {
			this.logger.error('Failed to find user', { id, error });
			return null;
		}
	}

	// Arrow function method
	findAll = async (): Promise<T[]> => {
		const response = await this.apiClient.get<T[]>('/users');
		return response.data;
	};

	// Method with overloads
	create(userData: Omit<T, 'id'>): Promise<T>;
	create(userData: Omit<T, 'id'>, options: CreateOptions): Promise<T>;
	async create(userData: Omit<T, 'id'>, options?: CreateOptions): Promise<T> {
		const response = await this.apiClient.post<T>('/users', userData);
		const user = response.data;
		this._users.set(user.id, user);
		return user;
	}

	// Async method
	async update(id: UserID, updates: Partial<T>): Promise<T> {
		const response = await this.apiClient.patch<T>('/users/' + id, updates);
		const user = response.data;
		this._users.set(id, user);
		return user;
	}

	// Method with complex return type
	async delete(id: UserID): Promise<boolean> {
		try {
			await this.apiClient.delete('/users/' + id);
			this._users.delete(id);
			return true;
		} catch {
			return false;
		}
	}

	// Static method
	static validateUser(user: unknown): user is User {
		return typeof user === 'object' && user !== null && 'id' in user;
	}

	// Getter
	get userCount(): number {
		return this._users.size;
	}

	// Setter
	set maxUsers(value: number) {
		this.config.maxUsers = value;
	}

	// Private method
	private validateUserData(data: Partial<T>): boolean {
		return data.name !== undefined && data.email !== undefined;
	}

	// Protected method
	protected notifyUserEvent(event: UserEvent): void {
		this._eventSubject.next(event);
	}
}

// Abstract class
abstract class BaseController {
	protected abstract handleRequest(req: Request): Promise<Response>;

	public async processRequest(req: Request): Promise<Response> {
		try {
			return await this.handleRequest(req);
		} catch (error) {
			return new Response('Error', { status: 500 });
		}
	}
}

// Extended class
class UserController extends BaseController {
	constructor(private userService: UserService) {
		super();
	}

	protected async handleRequest(req: Request): Promise<Response> {
		const users = await this.userService.findAll();
		return Response.json(users);
	}
}

// Function declarations
function createUser(data: UserData): User {
	return { ...data, id: generateId() };
}

function* userGenerator(): Generator<User, void, unknown> {
	let id = 1;
	while (true) {
		yield { id: id++, name: 'User ' + id, email: 'user' + id + '@example.com' };
	}
}

// Async function
async function fetchUserProfile(userId: UserID): Promise<UserProfile> {
	const response = await fetch('/api/users/' + userId + '/profile');
	return response.json();
}

// Arrow functions
const validateEmail = (email: string): boolean => {
	return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
};

const formatUserName = (user: User): string => user.firstName + ' ' + user.lastName;

// Function with generics
function mapArray<T, U>(array: T[], mapper: (item: T) => U): U[] {
	return array.map(mapper);
}

// Namespace declaration
namespace UserUtils {
	export function formatName(user: User): string {
		return user.firstName + ' ' + user.lastName;
	}

	export const DEFAULT_ROLE = UserRole.USER;
}

// Module declaration
declare module 'custom-module' {
	export interface CustomUser extends User {
		customField: string;
	}
}

// Variable declarations
const apiUrl: string = 'https://api.example.com';
let currentUser: User | null = null;
var globalConfig = { debug: true };

// Class expression
const DynamicClass = class implements Repository<User> {
	async findById(id: string): Promise<User | null> {
		return null;
	}

	async findAll(): Promise<User[]> {
		return [];
	}

	async create(entity: User): Promise<User> {
		return entity;
	}

	async update(id: string, updates: Partial<User>): Promise<User> {
		return {} as User;
	}

	async delete(id: string): Promise<boolean> {
		return true;
	}
};
`

	// Test complex function queries
	funcQuery := core.AgentQuery{Type: "function"}
	funcResult := provider.Query(source, funcQuery)
	if funcResult.Error != nil {
		t.Fatalf("Function query failed: %v", funcResult.Error)
	}

	expectedFunctions := []string{
		"findById", "findAll", "create", "update", "delete", "validateUser",
		"handleRequest", "processRequest", "createUser", "userGenerator",
		"fetchUserProfile", "validateEmail", "formatUserName", "mapArray",
		"formatName",
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

	if foundCount < 5 {
		t.Errorf("Expected to find at least 5 functions, found %d", foundCount)
	}

	// Test interface queries
	interfaceQuery := core.AgentQuery{Type: "interface"}
	interfaceResult := provider.Query(source, interfaceQuery)
	if interfaceResult.Error != nil {
		t.Fatalf("Interface query failed: %v", interfaceResult.Error)
	}

	if len(interfaceResult.Matches) < 1 {
		t.Error("Expected to find at least 1 interface")
	}

	// Test type queries
	typeQuery := core.AgentQuery{Type: "type"}
	typeResult := provider.Query(source, typeQuery)
	if typeResult.Error != nil {
		t.Fatalf("Type query failed: %v", typeResult.Error)
	}

	if len(typeResult.Matches) < 1 {
		t.Error("Expected to find at least 1 type alias")
	}

	// Test enum queries
	enumQuery := core.AgentQuery{Type: "enum"}
	enumResult := provider.Query(source, enumQuery)
	if enumResult.Error != nil {
		t.Fatalf("Enum query failed: %v", enumResult.Error)
	}

	if len(enumResult.Matches) < 1 {
		t.Error("Expected to find at least 1 enum")
	}

	// Test variable queries
	varQuery := core.AgentQuery{Type: "variable"}
	varResult := provider.Query(source, varQuery)
	if varResult.Error != nil {
		t.Fatalf("Variable query failed: %v", varResult.Error)
	}

	for _, match := range varResult.Matches {
		t.Logf("Found variable: %s", match.Name)
	}

	// Test import/export queries
	importQuery := core.AgentQuery{Type: "import"}
	importResult := provider.Query(source, importQuery)
	if importResult.Error != nil {
		t.Fatalf("Import query failed: %v", importResult.Error)
	}

	for _, match := range importResult.Matches {
		t.Logf("Found import/export: %s", match.Name)
	}
}

// Test complex transformations
func TestTypeScriptProvider_ComplexTransformations(t *testing.T) {
	provider := New()
	source := `
class ApiService {
	private baseUrl: string;

	constructor(baseUrl: string) {
		this.baseUrl = baseUrl;
	}

	async get<T>(endpoint: string): Promise<T> {
		const response = await fetch(this.baseUrl + endpoint);
		return response.json();
	}

	async post<T>(endpoint: string, data: any): Promise<T> {
		const response = await fetch(this.baseUrl + endpoint, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(data)
		});
		return response.json();
	}
}
`

	// Test insert_before transformation
	insertBeforeOp := core.TransformOp{
		Method: "insert_before",
		Target: core.AgentQuery{
			Type: "function",
			Name: "post",
		},
		Content: "\t// Validation method\n\tprivate validateEndpoint(endpoint: string): boolean {\n\t\treturn endpoint.startsWith('/');\n\t}",
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
			Name: "get",
		},
		Content: "\n\t// Enhanced get method with caching\n\tasync getCached<T>(endpoint: string): Promise<T> {\n\t\t// Implementation with caching\n\t\treturn this.get<T>(endpoint);\n\t}",
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
			Name: "ApiService",
		},
		Content: "\n\t// New method added via append\n\tasync delete(endpoint: string): Promise<void> {\n\t\tawait fetch(this.baseUrl + endpoint, { method: 'DELETE' });\n\t}",
	}

	result3 := provider.Transform(source, appendOp)
	if result3.Error != nil {
		t.Fatalf("Append transform failed: %v", result3.Error)
	}

	if result3.Modified == "" {
		t.Error("Expected modified code for append")
	}
}

// Test pattern matching with TypeScript-specific patterns
func TestTypeScriptProvider_PatternMatching(t *testing.T) {
	provider := New()
	source := `
interface IUserService {
	getUser(id: string): Promise<User>;
	getUserProfile(id: string): Promise<UserProfile>;
	getAdminData(id: string): Promise<Admin>;
	setUserActive(id: string): Promise<void>;
}

class UserServiceImpl implements IUserService {
	async getUser(id: string): Promise<User> {
		return {} as User;
	}

	async getUserProfile(id: string): Promise<UserProfile> {
		return {} as UserProfile;
	}

	async getAdminData(id: string): Promise<Admin> {
		return {} as Admin;
	}

	async setUserActive(id: string): Promise<void> {
		// Implementation
	}
}
`

	// Test wildcard pattern matching on methods
	getUserQuery := core.AgentQuery{
		Type: "function",
		Name: "getUser*", // Should match getUser and getUserProfile
	}

	result := provider.Query(source, getUserQuery)
	if result.Error != nil {
		t.Fatalf("Wildcard query failed: %v", result.Error)
	}

	// Should find getUser and getUserProfile methods
	expectedMatches := 4 // 2 in interface + 2 in class
	if len(result.Matches) != expectedMatches {
		t.Errorf("Expected %d matches for 'getUser*', got %d", expectedMatches, len(result.Matches))
	}

	for _, match := range result.Matches {
		if !strings.HasPrefix(match.Name, "getUser") {
			t.Errorf("Unexpected match '%s' for pattern 'getUser*'", match.Name)
		}
	}
}

// Test edge cases in ExtractNodeName
func TestTypeScriptProvider_ExtractNodeNameEdgeCases(t *testing.T) {
	provider := New()
	source := `
// Arrow functions with various forms
const arrowFunc1 = () => 'result';
const arrowFunc2 = (x: number) => x * 2;
const arrowFunc3 = async (data: any) => {
	return processData(data);
};

// Function expressions
const funcExpr1 = function() { return 'anonymous'; };
const funcExpr2 = function namedFuncExpr() { return 'named'; };

// Method definitions with computed property names
class TestClass {
	['computed' + 'Method']() {
		return 'computed';
	}

	[Symbol.iterator]() {
		return this;
	}

	// Getters and setters
	get value() {
		return this._value;
	}

	set value(val: any) {
		this._value = val;
	}
}

// Anonymous class expression
const AnonymousClass = class {
	method() {
		return 'anonymous class method';
	}
};

// Import/export statements with various forms
import defaultExport from './module';
import { namedExport1, namedExport2 as alias } from './module';
import * as namespace from './module';
export { something } from './other-module';
export default class ExportedClass {}
`

	// Test arrow function queries
	funcQuery := core.AgentQuery{Type: "function"}
	funcResult := provider.Query(source, funcQuery)
	if funcResult.Error != nil {
		t.Fatalf("Function query failed: %v", funcResult.Error)
	}

	for _, match := range funcResult.Matches {
		t.Logf("Found function: %s (type: %s)", match.Name, match.Type)
		// Anonymous functions should have "anonymous" as name
		if match.Name == "" {
			t.Error("Function name should not be empty")
		}
	}

	// Test import queries
	importQuery := core.AgentQuery{Type: "import"}
	importResult := provider.Query(source, importQuery)
	if importResult.Error != nil {
		t.Fatalf("Import query failed: %v", importResult.Error)
	}

	for _, match := range importResult.Matches {
		t.Logf("Found import/export: %s", match.Name)
	}
}

// Test invalid transformations
func TestTypeScriptProvider_InvalidTransformations(t *testing.T) {
	provider := New()
	source := `
class TestService {
	public testMethod(): string {
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
}

// Test confidence scoring with TypeScript-specific scenarios
func TestTypeScriptProvider_ConfidenceScoring(t *testing.T) {
	provider := New()
	source := `
export class PublicAPI {
	public PublicMethod(): string {
		return 'public';
	}

	private _privateMethod(): string {
		return 'private';
	}

	public MethodOne(): void {}
	public MethodTwo(): void {}
	public MethodThree(): void {}
	public MethodFour(): void {}
	public MethodFive(): void {}
	public MethodSix(): void {}
}
`

	// Test deleting exported method (should reduce confidence)
	deleteExportedOp := core.TransformOp{
		Method: "delete",
		Target: core.AgentQuery{
			Type: "function",
			Name: "PublicMethod", // PascalCase = exported in TypeScript
		},
	}

	result := provider.Transform(source, deleteExportedOp)
	if result.Error != nil {
		t.Fatalf("Delete exported transform failed: %v", result.Error)
	}

	// Should have reduced confidence due to deleting exported API
	if result.Confidence.Score >= 0.7 {
		t.Errorf("Expected reduced confidence for deleting exported method, got %f", result.Confidence.Score)
	}

	// Test replacing private method (should have higher confidence)
	replacePrivateOp := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "_privateMethod", // Not exported
		},
		Replacement: "private _privateMethod(): string { return 'modified'; }",
	}

	result2 := provider.Transform(source, replacePrivateOp)
	if result2.Error != nil {
		t.Fatalf("Replace private transform failed: %v", result2.Error)
	}

	// Should have higher confidence since it's not exported
	if result2.Confidence.Score <= 0.8 {
		t.Errorf("Expected higher confidence for modifying private method, got %f", result2.Confidence.Score)
	}
}
