# Go-Swagger SDK

A lightweight and easy-to-use Go SDK for generating OpenAPI 3.0 documentation with Gin framework integration.

## Features

- 🚀 Easy API definition with fluent interface
- 📝 Automatic OpenAPI 3.0 specification generation
- ✅ Built-in request/response validation
- 🔒 Multiple security schemes support (Basic Auth, Bearer, API Key, OAuth2)
- 🎯 Parameter validation (query, path, header, cookie)
- 🔄 Type-safe schema generation from Go structs
- ⚡ High performance with caching
- 🧪 Well-tested with comprehensive test coverage

## Installation

```bash
go get github.com/smartcat999/go-swagger
```

## Quick Start

### 1. Basic Usage

```go
package main

import (
    "net/http"
    "github.com/gin-gonic/gin"
    ginSwagger "github.com/smartcat999/go-swagger/pkg/gin"
    "github.com/smartcat999/go-swagger/pkg/api"
)

// Define your request/response models
type CreateUserRequest struct {
    Username string `json:"username" validate:"required,min=3"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age,omitempty"`
}

type UserResponse struct {
    ID       int64  `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
}

func main() {
    // Create Gin engine
    engine := gin.Default()
    
    // Create API router
    router := ginSwagger.NewAPIRouter(
        engine,
        "/api/v1",           // base path
        "My API",            // title
        "1.0.0",             // version
        "My API Description", // description
    )
    
    // Define and register APIs
    createUserAPI := api.NewAPIDefinition("POST", "/users", "Create a new user").
        WithDescription("Create a new user account").
        WithTags("users").
        WithRequest(CreateUserRequest{}).
        WithResponse(UserResponse{}).
        WithHandler(createUserHandler)
    
    router.Register(createUserAPI)
    
    // Generate and cache Swagger documentation
    if err := router.GenerateSwagger(); err != nil {
        panic(err)
    }
    
    // Register Swagger endpoint
    engine.GET("/swagger.json", router.SwaggerHandler)
    
    // Start server
    engine.Run(":8080")
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    // Your handler logic here
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(`{"id":1,"username":"john","email":"john@example.com"}`))
}
```

### 2. Parameter Validation

```go
// Add query parameters with validation
getUserAPI := api.NewAPIDefinition("GET", "/users", "Get users").
    WithParam("limit", "query", "Maximum number of results", false,
        api.NewValidationRule("min", 1, "Limit must be at least 1"),
        api.NewValidationRule("max", 100, "Limit cannot exceed 100"),
    ).
    WithParam("offset", "query", "Number of results to skip", false).
    WithHandler(getUsersHandler)

// Add path parameters
getUserByIDAPI := api.NewAPIDefinition("GET", "/users/:id", "Get user by ID").
    WithParam("id", "path", "User ID", true).
    WithHandler(getUserByIDHandler)
```

### 3. Security Schemes

```go
// Add Basic Authentication
router.AddBasicAuth("basicAuth", "HTTP Basic Authentication")

// Add Bearer Token Authentication
router.AddBearerAuth("bearerAuth", "JWT Bearer Token", "JWT")

// Add API Key Authentication
router.AddAPIKey("apiKey", "API Key in header", "header")

// Add OAuth2
flows := &api.OAuthFlows{
    AuthorizationCode: &api.OAuthFlow{
        AuthorizationURL: "https://example.com/oauth/authorize",
        TokenURL:         "https://example.com/oauth/token",
        Scopes: map[string]string{
            "read":  "Read access",
            "write": "Write access",
        },
    },
}
router.AddOAuth2("oauth2", "OAuth 2.0", flows)

// Set global security requirements
router.SetGlobalSecurity([]map[string][]string{
    {"bearerAuth": []string{}},
})

// Or set security per API
api := api.NewAPIDefinition("POST", "/admin/users", "Create admin user").
    WithSecurity("bearerAuth", []string{"admin"}).
    WithHandler(handler)
```

### 4. Advanced API Definition

```go
api := api.NewAPIDefinition("PUT", "/users/:id", "Update user").
    WithOperationID("updateUser").
    WithDescription("Update an existing user's information").
    WithTags("users", "management").
    WithParam("id", "path", "User ID", true).
    WithRequest(UpdateUserRequest{}).
    WithResponse(UserResponse{}).
    WithDeprecated(false).
    WithExternalDocs("User Guide", "https://docs.example.com/users").
    WithExample("success", api.Example{
        Summary: "Successful update",
        Value: map[string]interface{}{
            "id": 1,
            "username": "john_doe",
            "email": "john@example.com",
        },
    }).
    WithHandler(updateUserHandler)

router.Register(api)
```

### 5. Register Multiple APIs as a Group

```go
userAPIs := []api.APIDefinition{
    *api.NewAPIDefinition("GET", "/users", "List users").WithHandler(listHandler),
    *api.NewAPIDefinition("POST", "/users", "Create user").WithHandler(createHandler),
    *api.NewAPIDefinition("GET", "/users/:id", "Get user").WithHandler(getHandler),
    *api.NewAPIDefinition("PUT", "/users/:id", "Update user").WithHandler(updateHandler),
    *api.NewAPIDefinition("DELETE", "/users/:id", "Delete user").WithHandler(deleteHandler),
}

if err := router.RegisterGroup("users", userAPIs); err != nil {
    panic(err)
}
```

### 6. Custom Validation Rules

```go
api := api.NewAPIDefinition("POST", "/users", "Create user").
    WithParam("age", "query", "User age", true,
        api.NewValidationRule("min", 18.0, "Must be at least 18 years old"),
        api.NewValidationRule("max", 120.0, "Age cannot exceed 120"),
    ).
    WithParam("email", "query", "User email", true,
        api.NewValidationRule("email", nil, "Invalid email format"),
    ).
    WithParam("username", "query", "Username", true,
        api.NewValidationRule("pattern", "^[a-zA-Z0-9_]+$", "Username can only contain letters, numbers, and underscores"),
    ).
    WithHandler(handler)
```

## Schema Generation

The SDK automatically generates OpenAPI schemas from Go structs:

```go
type User struct {
    ID        int64     `json:"id" validate:"required"`
    Username  string    `json:"username" validate:"required,min=3,max=20"`
    Email     string    `json:"email" validate:"required,email"`
    Age       int       `json:"age,omitempty" validate:"min=0,max=150"`
    IsActive  bool      `json:"is_active"`
    CreatedAt time.Time `json:"created_at"`
    Tags      []string  `json:"tags,omitempty"`
    Profile   Profile   `json:"profile,omitempty"`
}

type Profile struct {
    Bio       string `json:"bio,omitempty"`
    Avatar    string `json:"avatar,omitempty"`
    Website   string `json:"website,omitempty"`
}
```

Supported struct tags:
- `json`: Field name in JSON (use `-` to exclude, `omitempty` for optional fields)
- `validate`: Validation rules (affects required fields in OpenAPI)
- `doc`: Field description in OpenAPI schema
- `example`: Example value in OpenAPI schema
- `format`: OpenAPI format (e.g., "date-time", "email", "uri")

## Testing

Run all tests:

```bash
cd go-swagger
go test ./...
```

Run tests with coverage:

```bash
go test -v -cover ./...
```

Run benchmarks:

```bash
go test -bench=. ./...
```

## API Documentation

After starting your server, access the generated OpenAPI specification at:

```
http://localhost:8080/swagger.json
```

You can use this with Swagger UI or other OpenAPI tools:

```bash
# Using Swagger UI Docker
docker run -p 8081:8080 -e SWAGGER_JSON=/swagger.json \
  -v $(pwd)/swagger.json:/swagger.json \
  swaggerapi/swagger-ui
```

Then open http://localhost:8081 in your browser.

## Error Handling

The SDK provides custom error types for better error handling:

```go
schema, err := api.SafeSchemaFromStruct(invalidType)
if err != nil {
    switch e := err.(type) {
    case *api.ValidationError:
        // Handle validation error
        fmt.Printf("Validation failed for field %s: %s\n", e.Field, e.Message)
    case *api.SchemaError:
        // Handle schema error
        fmt.Printf("Schema error (%s): %s\n", e.Type, e.Message)
    case *api.ErrInvalidType:
        // Handle invalid type error
        fmt.Printf("Invalid type: %s\n", e.Type)
    default:
        // Handle other errors
        fmt.Printf("Error: %v\n", err)
    }
}
```

## Performance Considerations

1. **Swagger Generation**: Call `GenerateSwagger()` once at startup, not on every request
2. **Validation**: Parameter validation is performed on every request
3. **Schema Caching**: Schemas are cached after first generation
4. **Handler Registration**: Register all handlers before starting the server

## Best Practices

1. Always define request/response types for better documentation
2. Use meaningful operation IDs for easier client generation
3. Group related APIs using tags
4. Add descriptions and examples for better API documentation
5. Validate input parameters to ensure data quality
6. Use appropriate HTTP status codes
7. Handle errors gracefully

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

