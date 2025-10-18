package gin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/smartcat999/go-swagger/pkg/api"
)

// Test structs
type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age,omitempty"`
}

type UserResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Message  string `json:"message,omitempty"`
}

// TestNewAPIRouter tests router creation
func TestNewAPIRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewAPIRouter(engine, "/api/v1", "Test API", "1.0.0", "Test API Description")

	if router == nil {
		t.Fatal("Expected router to be created")
	}
	if router.basePath != "/api/v1" {
		t.Errorf("Expected basePath '/api/v1', got %s", router.basePath)
	}
	if router.title != "Test API" {
		t.Errorf("Expected title 'Test API', got %s", router.title)
	}
	if router.version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", router.version)
	}
}

// TestSetInfo tests setting API info
func TestSetInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewAPIRouter(engine, "/api", "Old Title", "1.0.0", "Old Description")

	router.SetInfo("New Title", "2.0.0", "New Description")

	if router.title != "New Title" {
		t.Errorf("Expected title 'New Title', got %s", router.title)
	}
	if router.version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got %s", router.version)
	}
	if router.description != "New Description" {
		t.Errorf("Expected description 'New Description', got %s", router.description)
	}
}

// TestRegisterAPI tests API registration
func TestRegisterAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewAPIRouter(engine, "/api", "Test API", "1.0.0", "Test")

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"success"}`))
	}

	apiDef := api.NewAPIDefinition("GET", "/users", "Get users").
		WithHandler(testHandler)

	err := router.Register(apiDef)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if len(router.definitions) != 1 {
		t.Errorf("Expected 1 definition, got %d", len(router.definitions))
	}
}

// TestRegisterAPIValidation tests API registration validation
func TestRegisterAPIValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewAPIRouter(engine, "/api", "Test API", "1.0.0", "Test")

	tests := []struct {
		name    string
		api     *api.APIDefinition
		wantErr bool
	}{
		{
			name:    "nil API",
			api:     nil,
			wantErr: true,
		},
		{
			name: "missing handler",
			api: &api.APIDefinition{
				Method:  "GET",
				Path:    "/test",
				Handler: nil,
			},
			wantErr: true,
		},
		{
			name: "empty path",
			api: &api.APIDefinition{
				Method:  "GET",
				Path:    "",
				Handler: func(w http.ResponseWriter, r *http.Request) {},
			},
			wantErr: true,
		},
		{
			name: "invalid method",
			api: &api.APIDefinition{
				Method:  "INVALID",
				Path:    "/test",
				Handler: func(w http.ResponseWriter, r *http.Request) {},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := router.Register(tt.api)
			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSecuritySchemes tests security scheme management
func TestSecuritySchemes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewAPIRouter(engine, "/api", "Test API", "1.0.0", "Test")

	// Test Basic Auth
	router.AddBasicAuth("basicAuth", "Basic authentication")
	if len(router.securitySchemes) != 1 {
		t.Errorf("Expected 1 security scheme, got %d", len(router.securitySchemes))
	}

	// Test Bearer Auth
	router.AddBearerAuth("bearerAuth", "JWT Bearer token", "JWT")
	if len(router.securitySchemes) != 2 {
		t.Errorf("Expected 2 security schemes, got %d", len(router.securitySchemes))
	}

	// Test API Key
	router.AddAPIKey("apiKey", "API Key authentication", "header")
	if len(router.securitySchemes) != 3 {
		t.Errorf("Expected 3 security schemes, got %d", len(router.securitySchemes))
	}

	// Verify scheme types
	if router.securitySchemes["basicAuth"].Type != "http" {
		t.Error("Expected basicAuth type to be 'http'")
	}
	if router.securitySchemes["bearerAuth"].Scheme != "bearer" {
		t.Error("Expected bearerAuth scheme to be 'bearer'")
	}
	if router.securitySchemes["apiKey"].Type != "apiKey" {
		t.Error("Expected apiKey type to be 'apiKey'")
	}
}

// TestGenerateSwagger tests Swagger document generation
func TestGenerateSwagger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewAPIRouter(engine, "/api", "Test API", "1.0.0", "Test Description")

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	// Register a test API
	apiDef := api.NewAPIDefinition("GET", "/users", "Get users").
		WithDescription("Get all users").
		WithTags("users").
		WithResponse(UserResponse{}).
		WithHandler(testHandler)

	err := router.Register(apiDef)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Generate swagger
	apiDoc, err := router.GenerateSwagger()
	if err != nil {
		t.Fatalf("GenerateSwagger failed: %v", err)
	}
	if apiDoc == nil {
		t.Fatal("Expected non-nil OpenAPI document")
	}

	if !router.generated {
		t.Error("Expected generated flag to be true")
	}
	if router.swaggerDoc == nil {
		t.Error("Expected swagger document to be generated")
	}

	// Verify JSON is valid
	var doc map[string]interface{}
	err = json.Unmarshal(router.swaggerDoc, &doc)
	if err != nil {
		t.Fatalf("Generated swagger is not valid JSON: %v", err)
	}

	// Verify OpenAPI version
	if doc["openapi"] != "3.0.0" {
		t.Errorf("Expected openapi version '3.0.0', got %v", doc["openapi"])
	}
}

// TestGenerateSwaggerValidation tests swagger generation validation
func TestGenerateSwaggerValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		title       string
		version     string
		description string
		addAPI      bool
		wantErr     bool
	}{
		{
			name:        "missing title",
			title:       "",
			version:     "1.0.0",
			description: "Test",
			addAPI:      true,
			wantErr:     true,
		},
		{
			name:        "missing version",
			title:       "Test API",
			version:     "",
			description: "Test",
			addAPI:      true,
			wantErr:     true,
		},
		{
			name:        "no APIs defined",
			title:       "Test API",
			version:     "1.0.0",
			description: "Test",
			addAPI:      false,
			wantErr:     true,
		},
		{
			name:        "valid configuration",
			title:       "Test API",
			version:     "1.0.0",
			description: "Test",
			addAPI:      true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new engine for each test to avoid route conflicts
			engine := gin.New()
			router := NewAPIRouter(engine, "/api", tt.title, tt.version, tt.description)

			if tt.addAPI {
				testHandler := func(w http.ResponseWriter, r *http.Request) {}
				apiDef := api.NewAPIDefinition("GET", "/test", "Test").
					WithHandler(testHandler)
				_ = router.Register(apiDef)
			}

			doc, err := router.GenerateSwagger()
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSwagger() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && doc == nil {
				t.Error("Expected non-nil document when no error")
			}
		})
	}
}

// TestSwaggerHandler tests the swagger HTTP handler
func TestSwaggerHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewAPIRouter(engine, "/api", "Test API", "1.0.0", "Test")

	testHandler := func(w http.ResponseWriter, r *http.Request) {}
	apiDef := api.NewAPIDefinition("GET", "/test", "Test").
		WithHandler(testHandler)
	_ = router.Register(apiDef)
	_, _ = router.GenerateSwagger()

	// Test swagger handler
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/swagger.json", nil)

	router.SwaggerHandler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("Expected content-type 'application/json; charset=utf-8', got %s", contentType)
	}

	// Verify it's valid JSON
	var doc map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &doc)
	if err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}
}

// TestSwaggerHandlerNotGenerated tests swagger handler when not generated
func TestSwaggerHandlerNotGenerated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewAPIRouter(engine, "/api", "Test API", "1.0.0", "Test")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/swagger.json", nil)

	router.SwaggerHandler(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

// TestParameterValidationMiddleware tests parameter validation in handlers
func TestParameterValidationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewAPIRouter(engine, "/api", "Test API", "1.0.0", "Test")

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"success"}`))
	}

	// API with required query parameter
	apiDef := api.NewAPIDefinition("GET", "/users", "Get users").
		WithParam("limit", "query", "Result limit", true).
		WithHandler(testHandler)

	err := router.Register(apiDef)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	tests := []struct {
		name       string
		url        string
		wantStatus int
	}{
		{
			name:       "missing required parameter",
			url:        "/api/users",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid request",
			url:        "/api/users?limit=10",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tt.url, nil)
			engine.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

// TestRequestBodyValidation tests request body validation
func TestRequestBodyValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewAPIRouter(engine, "/api", "Test API", "1.0.0", "Test")

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"username":"test","email":"test@example.com"}`))
	}

	apiDef := api.NewAPIDefinition("POST", "/users", "Create user").
		WithRequest(CreateUserRequest{}).
		WithHandler(testHandler)

	err := router.Register(apiDef)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	tests := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
	}{
		{
			name:        "valid JSON",
			body:        `{"username":"john","email":"john@example.com","age":25}`,
			contentType: "application/json",
			wantStatus:  http.StatusCreated,
		},
		{
			name:        "invalid JSON",
			body:        `{"username":"john","email":`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "wrong content type",
			body:        `{"username":"john"}`,
			contentType: "text/plain",
			wantStatus:  http.StatusUnsupportedMediaType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/users", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			engine.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestRegisterGroup tests API group registration
func TestRegisterGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewAPIRouter(engine, "/api", "Test API", "1.0.0", "Test")

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	apis := []api.APIDefinition{
		*api.NewAPIDefinition("GET", "/users", "List users").WithHandler(testHandler),
		*api.NewAPIDefinition("POST", "/users", "Create user").WithHandler(testHandler),
		*api.NewAPIDefinition("GET", "/users/:id", "Get user").WithHandler(testHandler),
	}

	err := router.RegisterGroup("users", apis)
	if err != nil {
		t.Fatalf("RegisterGroup failed: %v", err)
	}

	if len(router.definitions) != 3 {
		t.Errorf("Expected 3 definitions, got %d", len(router.definitions))
	}

	// Verify all APIs have the tag
	for _, def := range router.definitions {
		if len(def.Tags) == 0 || def.Tags[0] != "users" {
			t.Error("Expected all APIs to have 'users' tag")
		}
	}
}

// BenchmarkRegisterAPI benchmarks API registration
func BenchmarkRegisterAPI(b *testing.B) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewAPIRouter(engine, "/api", "Test API", "1.0.0", "Test")

	testHandler := func(w http.ResponseWriter, r *http.Request) {}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		apiDef := api.NewAPIDefinition("GET", "/test", "Test").
			WithHandler(testHandler)
		_ = router.Register(apiDef)
	}
}

// BenchmarkGenerateSwagger benchmarks swagger generation
func BenchmarkGenerateSwagger(b *testing.B) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := NewAPIRouter(engine, "/api", "Test API", "1.0.0", "Test")

	testHandler := func(w http.ResponseWriter, r *http.Request) {}

	// Register some APIs
	for i := 0; i < 10; i++ {
		apiDef := api.NewAPIDefinition("GET", "/test", "Test").
			WithHandler(testHandler)
		_ = router.Register(apiDef)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = router.GenerateSwagger()
	}
}
