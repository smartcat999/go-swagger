package gin

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/smartcat999/go-swagger/pkg/api"
)

// WithGinHandler is a helper function to set a gin.HandlerFunc as the native handler
// Usage: api.NewAPI(...).WithGinHandler(yourGinHandler)
func WithGinHandler(apiDef *api.APIDefinition, handler gin.HandlerFunc) *api.APIDefinition {
	return apiDef.WithNativeHandler(handler)
}

// APIRouter enhanced route registrar
type APIRouter struct {
	engine          *gin.Engine
	definitions     []api.APIDefinition
	basePath        string
	title           string
	version         string
	description     string
	swaggerDoc      []byte // Cached swagger document
	generated       bool   // Whether swagger has been generated
	securitySchemes map[string]api.SecurityScheme
	globalSecurity  []map[string][]string
}

// NewAPIRouter creates a new API route registrar
func NewAPIRouter(engine *gin.Engine, basePath, title, version, description string) *APIRouter {
	return &APIRouter{
		engine:          engine,
		definitions:     make([]api.APIDefinition, 0),
		basePath:        basePath,
		title:           title,
		version:         version,
		description:     description,
		swaggerDoc:      nil,
		generated:       false,
		securitySchemes: make(map[string]api.SecurityScheme),
		globalSecurity:  make([]map[string][]string, 0),
	}
}

// SetInfo sets basic API information
func (r *APIRouter) SetInfo(title, version, description string) {
	r.title = title
	r.version = version
	r.description = description
}

// SetBasePath sets API base path
func (r *APIRouter) SetBasePath(basePath string) {
	r.basePath = basePath
}

// AddBasicAuth adds Basic Authentication security scheme
func (r *APIRouter) AddBasicAuth(name, description string) {
	r.securitySchemes[name] = api.SecurityScheme{
		Type:        "http",
		Scheme:      "basic",
		Description: description,
	}
}

// AddBearerAuth adds Bearer token security scheme
func (r *APIRouter) AddBearerAuth(name, description, format string) {
	r.securitySchemes[name] = api.SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: format,
		Description:  description,
	}
}

// AddAPIKey adds API key security scheme
func (r *APIRouter) AddAPIKey(name, description, in string) {
	r.securitySchemes[name] = api.SecurityScheme{
		Type:        "apiKey",
		Name:        name,
		In:          in,
		Description: description,
	}
}

// AddOAuth2 adds OAuth2 security scheme
func (r *APIRouter) AddOAuth2(name, description string, flows *api.OAuthFlows) {
	r.securitySchemes[name] = api.SecurityScheme{
		Type:        "oauth2",
		Description: description,
		Flows:       flows,
	}
}

// AddOpenIDConnect adds OpenID Connect security scheme
func (r *APIRouter) AddOpenIDConnect(name, description, url string) {
	r.securitySchemes[name] = api.SecurityScheme{
		Type:             "openIdConnect",
		Description:      description,
		OpenIDConnectURL: url,
	}
}

// SetGlobalSecurity sets global security requirements
func (r *APIRouter) SetGlobalSecurity(requirements []map[string][]string) {
	r.globalSecurity = requirements
}

// Register registers an API route
func (r *APIRouter) Register(api *api.APIDefinition) error {
	if api == nil {
		return fmt.Errorf("api definition cannot be nil")
	}

	// Check if we have either a native handler or standard handler
	if api.NativeHandler == nil && api.Handler == nil {
		return fmt.Errorf("handler cannot be nil for path: %s", api.Path)
	}

	// Validate path
	if api.Path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Validate method
	method := strings.ToUpper(api.Method)
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		// Valid method
	default:
		return fmt.Errorf("unsupported HTTP method: %s", api.Method)
	}

	// Create middleware chain for parameter validation
	handler := func(c *gin.Context) {
		// Validate path parameters
		for _, param := range api.Params {
			if param.In == "path" {
				value := c.Param(param.Name)
				if param.Required && value == "" {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
						"error": fmt.Sprintf("missing required path parameter: %s", param.Name),
					})
					return
				}
				if value != "" {
					if err := param.Validate(value); err != nil {
						c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
							"error": fmt.Sprintf("invalid path parameter %s: %v", param.Name, err),
						})
						return
					}
				}
			}
		}

		// Validate query parameters
		for _, param := range api.Params {
			if param.In == "query" {
				value := c.Query(param.Name)
				if param.Required && value == "" {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
						"error": fmt.Sprintf("missing required query parameter: %s", param.Name),
					})
					return
				}
				if value != "" {
					if err := param.Validate(value); err != nil {
						c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
							"error": fmt.Sprintf("invalid query parameter %s: %v", param.Name, err),
						})
						return
					}
				}
			}
		}

		// Validate header parameters
		for _, param := range api.Params {
			if param.In == "header" {
				value := c.GetHeader(param.Name)
				if param.Required && value == "" {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
						"error": fmt.Sprintf("missing required header: %s", param.Name),
					})
					return
				}
				if value != "" {
					if err := param.Validate(value); err != nil {
						c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
							"error": fmt.Sprintf("invalid header %s: %v", param.Name, err),
						})
						return
					}
				}
			}
		}

		// Validate cookie parameters
		for _, param := range api.Params {
			if param.In == "cookie" {
				value, err := c.Cookie(param.Name)
				if err != nil {
					if param.Required {
						c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
							"error": fmt.Sprintf("missing required cookie: %s", param.Name),
						})
						return
					}
					continue
				}
				if value != "" {
					if err := param.Validate(value); err != nil {
						c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
							"error": fmt.Sprintf("invalid cookie %s: %v", param.Name, err),
						})
						return
					}
				}
			}
		}

		// Validate request body if needed
		if api.Request != nil && (method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch) {
			contentType := c.GetHeader("Content-Type")
			if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
				c.AbortWithStatusJSON(http.StatusUnsupportedMediaType, gin.H{
					"error": "Content-Type must be application/json",
				})
				return
			}

			var requestBody interface{}
			if err := c.ShouldBindJSON(&requestBody); err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("invalid request body: %v", err),
				})
				return
			}

			// Note: Request body validation would require schema generation
			// For now, we'll skip detailed validation and rely on JSON unmarshaling
		}

		// Call the actual handler
		// Prefer NativeHandler (gin.HandlerFunc) over standard http.HandlerFunc
		if api.NativeHandler != nil {
			if ginHandler, ok := api.NativeHandler.(gin.HandlerFunc); ok {
				ginHandler(c)
				return
			}
		}

		// Fallback to standard HTTP handler
		if api.Handler != nil {
			api.Handler(c.Writer, c.Request)
		}
	}

	// Register to gin engine
	fullPath := fmt.Sprintf("%s%s", r.basePath, api.Path)
	r.engine.Handle(method, fullPath, handler)

	// Save API definition information
	r.definitions = append(r.definitions, *api)
	return nil
}

// RegisterGroup registers a group of related APIs
func (r *APIRouter) RegisterGroup(tag string, apis []api.APIDefinition) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}

	if len(apis) == 0 {
		return fmt.Errorf("apis cannot be empty")
	}

	for i, api := range apis {
		if len(api.Tags) == 0 {
			api.Tags = []string{tag}
		}
		if err := r.Register(&api); err != nil {
			return fmt.Errorf("failed to register API %d: %w", i, err)
		}
	}

	return nil
}

// GenerateSwagger generates and caches the swagger document, returns the generated document
func (r *APIRouter) GenerateSwagger() (*api.OpenAPIDoc, error) {
	if r.title == "" {
		return nil, fmt.Errorf("API title is required")
	}
	if r.version == "" {
		return nil, fmt.Errorf("API version is required")
	}

	// Build OpenAPI document
	doc, err := r.BuildOpenAPI()
	if err != nil {
		return nil, fmt.Errorf("failed to build OpenAPI document: %w", err)
	}

	// Validate required fields
	if len(doc.Paths) == 0 {
		return nil, fmt.Errorf("no API paths defined")
	}

	// Add default server if none defined
	if len(doc.Servers) == 0 {
		doc.Servers = []api.OpenAPIServer{
			{
				URL:         r.basePath,
				Description: "Default server",
			},
		}
	}

	// Initialize components if nil
	if doc.Components == nil {
		doc.Components = &api.Components{
			Schemas:         make(map[string]interface{}),
			SecuritySchemes: make(map[string]api.SecurityScheme),
			Parameters:      make(map[string]api.Parameter),
			RequestBodies:   make(map[string]api.RequestBody),
			Responses:       make(map[string]api.Response),
			Headers:         make(map[string]api.Header),
			Examples:        make(map[string]api.Example),
		}
	}

	// Add default responses for all operations
	for path := range doc.Paths {
		pathItem := doc.Paths[path]
		operations := []*api.Operation{
			pathItem.Get,
			pathItem.Post,
			pathItem.Put,
			pathItem.Delete,
			pathItem.Patch,
		}

		for _, op := range operations {
			if op != nil {
				// Add default error responses if not present
				if op.Responses == nil {
					op.Responses = make(map[string]api.Response)
				}
				if _, ok := op.Responses["400"]; !ok {
					op.Responses["400"] = api.Response{
						Description: "Bad Request - Invalid input parameters",
					}
				}
				if _, ok := op.Responses["401"]; !ok && len(op.Security) > 0 {
					op.Responses["401"] = api.Response{
						Description: "Unauthorized - Authentication required",
					}
				}
				if _, ok := op.Responses["403"]; !ok && len(op.Security) > 0 {
					op.Responses["403"] = api.Response{
						Description: "Forbidden - Insufficient permissions",
					}
				}
				if _, ok := op.Responses["500"]; !ok {
					op.Responses["500"] = api.Response{
						Description: "Internal Server Error",
					}
				}

				// Generate operationId if not set
				if op.OperationID == "" {
					op.OperationID = generateOperationID(path, op)
				}
			}
		}
		doc.Paths[path] = pathItem
	}

	// Marshal document
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAPI document: %w", err)
	}

	r.swaggerDoc = data
	r.generated = true
	return doc, nil
}

// generateOperationID generates a unique operation ID based on the path and operation
func generateOperationID(path string, op *api.Operation) string {
	// Remove path parameters
	path = regexp.MustCompile(`\{[^}]+\}`).ReplaceAllString(path, "")
	// Remove special characters
	path = regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(path, "_")
	// Remove consecutive underscores
	path = regexp.MustCompile(`_+`).ReplaceAllString(path, "_")
	// Remove leading and trailing underscores
	path = strings.Trim(path, "_")

	// Get the first tag if available
	prefix := "operation"
	if len(op.Tags) > 0 {
		prefix = strings.ToLower(op.Tags[0])
	}

	// Generate operation ID
	return fmt.Sprintf("%s_%s", prefix, path)
}

// SwaggerHandler provides swagger.json endpoint
func (r *APIRouter) SwaggerHandler(c *gin.Context) {
	if !r.generated || r.swaggerDoc == nil {
		// This should not happen if GenerateSwagger was called at startup
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Swagger documentation not available",
			"message": "Documentation was not generated at startup",
		})
		return
	}

	// Set cache headers for better performance
	c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	c.Header("ETag", fmt.Sprintf(`"%x"`, md5.Sum(r.swaggerDoc)))

	c.Data(http.StatusOK, "application/json; charset=utf-8", r.swaggerDoc)
}

// GetDefinitions returns all registered API definitions
func (r *APIRouter) GetDefinitions() []api.APIDefinition {
	return r.definitions
}

// BuildOpenAPI builds OpenAPI specification document
func (r *APIRouter) BuildOpenAPI() (*api.OpenAPIDoc, error) {
	doc := &api.OpenAPIDoc{
		OpenAPI: "3.0.0",
		Info: api.OpenAPIInfo{
			Title:       r.title,
			Version:     r.version,
			Description: r.description,
		},
		Servers: []api.OpenAPIServer{
			{
				URL:         r.basePath,
				Description: "API Server",
			},
		},
		Paths: make(map[string]api.PathItem),
		Components: &api.Components{
			SecuritySchemes: r.securitySchemes,
		},
	}

	// Add global security requirements if any
	if len(r.globalSecurity) > 0 {
		doc.Security = r.globalSecurity
	}

	// Generate OpenAPI paths for each API definition
	for _, apiDef := range r.definitions {
		pathItem := doc.Paths[apiDef.Path]

		operation := &api.Operation{
			Summary:     apiDef.Summary,
			Description: apiDef.Description,
			Tags:        apiDef.Tags,
			Responses:   make(map[string]api.Response),
		}

		// Generate parameter definitions
		if len(apiDef.Params) > 0 {
			operation.Parameters = apiDef.Params
		}

		// Generate request body schema
		if apiDef.Request != nil {
			schema, err := api.SafeSchemaFromStruct(apiDef.Request)
			if err != nil {
				return nil, fmt.Errorf("failed to generate request schema: %w", err)
			}
			if schema != nil {
				operation.RequestBody = &api.RequestBody{
					Content: map[string]api.Content{
						"application/json": {
							Schema: schema,
						},
					},
				}
			}
		}

		// Generate response schema
		if apiDef.Response != nil {
			schema, err := api.SafeSchemaFromStruct(apiDef.Response)
			if err != nil {
				return nil, fmt.Errorf("failed to generate response schema: %w", err)
			}
			if schema != nil {
				operation.Responses["200"] = api.Response{
					Description: "Success",
					Content: map[string]api.Content{
						"application/json": {
							Schema: schema,
						},
					},
				}
			}
		}

		// Add default error responses
		operation.Responses["400"] = api.Response{
			Description: "Bad Request",
		}
		operation.Responses["500"] = api.Response{
			Description: "Internal Server Error",
		}

		// Set operation based on HTTP method
		switch apiDef.Method {
		case http.MethodGet:
			pathItem.Get = operation
		case http.MethodPost:
			pathItem.Post = operation
		case http.MethodPut:
			pathItem.Put = operation
		case http.MethodDelete:
			pathItem.Delete = operation
		case http.MethodPatch:
			pathItem.Patch = operation
		}

		doc.Paths[apiDef.Path] = pathItem
	}

	return doc, nil
}
