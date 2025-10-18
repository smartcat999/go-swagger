package api

import (
	"testing"
	"time"
)

// Test structs for schema generation
type User struct {
	ID        int64     `json:"id" validate:"required"`
	Username  string    `json:"username" validate:"required,min=3,max=20"`
	Email     string    `json:"email" validate:"required,email"`
	Age       int       `json:"age,omitempty" validate:"min=0,max=150"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	Tags      []string  `json:"tags,omitempty"`
}

type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	Country string `json:"country"`
	ZipCode string `json:"zip_code,omitempty"`
}

// TestNewAPIDefinition tests API definition creation
func TestNewAPIDefinition(t *testing.T) {
	api := NewAPIDefinition("GET", "/users", "Get all users")

	if api.Method != "GET" {
		t.Errorf("Expected method GET, got %s", api.Method)
	}
	if api.Path != "/users" {
		t.Errorf("Expected path /users, got %s", api.Path)
	}
	if api.Summary != "Get all users" {
		t.Errorf("Expected summary 'Get all users', got %s", api.Summary)
	}
	if api.Tags == nil {
		t.Error("Expected Tags to be initialized")
	}
	if api.Params == nil {
		t.Error("Expected Params to be initialized")
	}
}

// TestAPIDefinitionChaining tests method chaining
func TestAPIDefinitionChaining(t *testing.T) {
	api := NewAPIDefinition("POST", "/users", "Create user").
		WithDescription("Create a new user account").
		WithTags("users", "management").
		WithParam("limit", "query", "Maximum number of results", false).
		WithOperationID("createUser")

	if api.Description != "Create a new user account" {
		t.Errorf("Expected description to be set")
	}
	if len(api.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(api.Tags))
	}
	if len(api.Params) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(api.Params))
	}
	if api.OperationID != "createUser" {
		t.Errorf("Expected operationID 'createUser', got %s", api.OperationID)
	}
}

// TestSchemaFromStruct tests schema generation from struct
func TestSchemaFromStruct(t *testing.T) {
	user := User{}
	schema, err := SchemaFromStruct(user)
	if err != nil {
		t.Fatalf("SchemaFromStruct failed: %v", err)
	}

	if schema == nil {
		t.Fatal("Expected schema to be generated")
	}

	schemaType, ok := schema["type"].(string)
	if !ok || schemaType != "object" {
		t.Errorf("Expected type 'object', got %v", schemaType)
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	// Check if expected fields exist
	expectedFields := []string{"id", "username", "email", "age", "is_active", "created_at", "tags"}
	for _, field := range expectedFields {
		if _, exists := properties[field]; !exists {
			t.Errorf("Expected field %s to exist in schema", field)
		}
	}

	// Check required fields
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Expected required to be a string slice")
	}

	expectedRequired := map[string]bool{"id": true, "username": true, "email": true, "is_active": true, "created_at": true}
	for _, field := range required {
		if !expectedRequired[field] {
			t.Errorf("Field %s should be required", field)
		}
	}
}

// TestSafeSchemaFromStruct tests safe schema generation
func TestSafeSchemaFromStruct(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "valid struct",
			input:   User{},
			wantErr: false,
		},
		{
			name:    "nil input",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "invalid type",
			input:   "string",
			wantErr: true,
		},
		{
			name:    "pointer to struct",
			input:   &User{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := SafeSchemaFromStruct(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeSchemaFromStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && schema == nil {
				t.Error("Expected schema to be generated")
			}
		})
	}
}

// TestParameterValidation tests parameter validation
func TestParameterValidation(t *testing.T) {
	tests := []struct {
		name      string
		param     Parameter
		value     interface{}
		wantError bool
	}{
		{
			name: "required parameter - valid",
			param: Parameter{
				Name:     "id",
				Required: true,
			},
			value:     "123",
			wantError: false,
		},
		{
			name: "required parameter - missing",
			param: Parameter{
				Name:     "id",
				Required: true,
			},
			value:     "",
			wantError: true,
		},
		{
			name: "optional parameter - empty",
			param: Parameter{
				Name:     "filter",
				Required: false,
			},
			value:     "",
			wantError: false,
		},
		{
			name: "parameter with min validation",
			param: Parameter{
				Name:     "age",
				Required: true,
				Validations: []ValidationRule{
					{Type: "min", Value: 18.0, Message: "Age must be at least 18"},
				},
			},
			value:     "20",
			wantError: false,
		},
		{
			name: "parameter with min validation - fail",
			param: Parameter{
				Name:     "age",
				Required: true,
				Validations: []ValidationRule{
					{Type: "min", Value: 18.0, Message: "Age must be at least 18"},
				},
			},
			value:     "15",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.param.Validate(tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("Parameter.Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestValidationError tests custom error types
func TestValidationError(t *testing.T) {
	err := NewValidationError("email", "format", "Invalid email format", nil)
	if err == nil {
		t.Fatal("Expected error to be created")
	}

	verr, ok := err.(*ValidationError)
	if !ok {
		t.Fatal("Expected ValidationError type")
	}

	if verr.Field != "email" {
		t.Errorf("Expected field 'email', got %s", verr.Field)
	}

	if verr.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

// TestSchemaError tests schema error type
func TestSchemaError(t *testing.T) {
	err := NewSchemaError("invalid_type", "Type must be a struct", nil)
	if err == nil {
		t.Fatal("Expected error to be created")
	}

	serr, ok := err.(*SchemaError)
	if !ok {
		t.Fatal("Expected SchemaError type")
	}

	if serr.Type != "invalid_type" {
		t.Errorf("Expected type 'invalid_type', got %s", serr.Type)
	}

	if serr.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

// TestNestedStructSchema tests schema generation for nested structs
func TestNestedStructSchema(t *testing.T) {
	type UserWithAddress struct {
		User    User    `json:"user"`
		Address Address `json:"address"`
	}

	userAddr := UserWithAddress{}
	schema, err := SchemaFromStruct(userAddr)
	if err != nil {
		t.Fatalf("SchemaFromStruct failed: %v", err)
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	// Check nested user object
	if _, exists := properties["user"]; !exists {
		t.Error("Expected 'user' field in schema")
	}

	// Check nested address object
	if _, exists := properties["address"]; !exists {
		t.Error("Expected 'address' field in schema")
	}
}

// TestSliceSchema tests schema generation for slices
func TestSliceSchema(t *testing.T) {
	type UserList struct {
		Users []User `json:"users"`
	}

	userList := UserList{}
	schema, err := SchemaFromStruct(userList)
	if err != nil {
		t.Fatalf("SchemaFromStruct failed: %v", err)
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	usersField, exists := properties["users"]
	if !exists {
		t.Fatal("Expected 'users' field in schema")
	}

	usersSchema, ok := usersField.(map[string]interface{})
	if !ok {
		t.Fatal("Expected users field to be a map")
	}

	if usersSchema["type"] != "array" {
		t.Errorf("Expected type 'array', got %v", usersSchema["type"])
	}

	if _, hasItems := usersSchema["items"]; !hasItems {
		t.Error("Expected 'items' field in array schema")
	}
}

// TestTimeField tests time.Time field handling
func TestTimeField(t *testing.T) {
	user := User{
		CreatedAt: time.Now(),
	}

	schema, err := SchemaFromStruct(user)
	if err != nil {
		t.Fatalf("SchemaFromStruct failed: %v", err)
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	createdAtField, exists := properties["created_at"]
	if !exists {
		t.Fatal("Expected 'created_at' field in schema")
	}

	createdAtSchema, ok := createdAtField.(map[string]interface{})
	if !ok {
		t.Fatal("Expected created_at field to be a map")
	}

	if createdAtSchema["type"] != "string" {
		t.Errorf("Expected type 'string' for time.Time, got %v", createdAtSchema["type"])
	}

	if createdAtSchema["format"] != "date-time" {
		t.Errorf("Expected format 'date-time' for time.Time, got %v", createdAtSchema["format"])
	}
}

// BenchmarkSchemaGeneration benchmarks schema generation
func BenchmarkSchemaGeneration(b *testing.B) {
	user := User{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = SchemaFromStruct(user)
	}
}

// BenchmarkParameterValidation benchmarks parameter validation
func BenchmarkParameterValidation(b *testing.B) {
	param := Parameter{
		Name:     "email",
		Required: true,
		Validations: []ValidationRule{
			{Type: "min", Value: 5, Message: "Too short"},
			{Type: "max", Value: 100, Message: "Too long"},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = param.Validate("test@example.com")
	}
}
