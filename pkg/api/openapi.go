package api

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// APIDefinition stores complete API definition information
type APIDefinition struct {
	Method        string                 // HTTP method
	Path          string                 // Route path
	OperationID   string                 // Unique operation ID
	Summary       string                 // API summary
	Description   string                 // API detailed description
	Tags          []string               // API tag groups
	Request       interface{}            // Request structure
	Response      interface{}            // Response structure
	Params        []Parameter            // Path parameters, query parameters, etc.
	Handler       http.HandlerFunc       // Standard HTTP handler (fallback)
	NativeHandler interface{}            // Framework-specific handler (e.g., gin.HandlerFunc, echo.HandlerFunc)
	Deprecated    bool                   // Whether the API is deprecated
	Security      []map[string][]string  // Security requirements
	ExternalDocs  *ExternalDocumentation // External documentation
	Examples      map[string]Example     // Request/response examples
	Servers       []OpenAPIServer        // Operation-specific servers
}

// ValidationRule defines a validation rule for a parameter
type ValidationRule struct {
	Type    string      // Validation type (e.g., "required", "min", "max", "pattern")
	Value   interface{} // Validation value
	Message string      // Error message
}

// Parameter defines parameter information
type Parameter struct {
	Name            string                 `json:"name"`        // Parameter name
	In              string                 `json:"in"`          // Parameter location: path, query, header, cookie
	Description     string                 `json:"description"` // Parameter description
	Required        bool                   `json:"required"`    // Whether required
	Deprecated      bool                   `json:"deprecated,omitempty"`
	AllowEmptyValue bool                   `json:"allowEmptyValue,omitempty"`
	Style           string                 `json:"style,omitempty"`   // How the parameter value will be serialized
	Explode         bool                   `json:"explode,omitempty"` // Whether arrays and objects should generate separate parameters
	Schema          map[string]interface{} `json:"schema,omitempty"`  // JSON Schema definition
	Example         interface{}            `json:"example,omitempty"`
	Examples        map[string]Example     `json:"examples,omitempty"`
	Content         map[string]Content     `json:"content,omitempty"`
	Validations     []ValidationRule       `json:"-"` // Validation rules
}

// Validate validates a parameter value against its validation rules
func (p *Parameter) Validate(value interface{}) error {
	if value == nil {
		if p.Required {
			return fmt.Errorf("parameter %s is required", p.Name)
		}
		return nil
	}

	// Convert value to string if it's not already
	strValue, ok := value.(string)
	if !ok {
		strValue = fmt.Sprintf("%v", value)
	}

	// Check required
	if p.Required && strValue == "" {
		return fmt.Errorf("parameter %s is required", p.Name)
	}

	// If value is empty and not required, skip other validations
	if strValue == "" && !p.Required {
		return nil
	}

	for _, rule := range p.Validations {
		switch rule.Type {
		case "min":
			if minVal, ok := rule.Value.(float64); ok {
				// For numeric values
				numVal, parseErr := strconv.ParseFloat(strValue, 64)
				if parseErr == nil {
					if numVal < minVal {
						return fmt.Errorf(rule.Message)
					}
				}
			} else if minLen, ok := rule.Value.(int); ok {
				// For string length
				if len(strValue) < minLen {
					return fmt.Errorf(rule.Message)
				}
			}

		case "max":
			if maxVal, ok := rule.Value.(float64); ok {
				// For numeric values
				numVal, parseErr := strconv.ParseFloat(strValue, 64)
				if parseErr == nil {
					if numVal > maxVal {
						return fmt.Errorf(rule.Message)
					}
				}
			} else if maxLen, ok := rule.Value.(int); ok {
				// For string length
				if len(strValue) > maxLen {
					return fmt.Errorf(rule.Message)
				}
			}

		case "pattern":
			if pattern, ok := rule.Value.(string); ok {
				matched, matchErr := regexp.MatchString(pattern, strValue)
				if matchErr != nil || !matched {
					return fmt.Errorf(rule.Message)
				}
			}

		case "enum":
			if values, ok := rule.Value.([]interface{}); ok {
				found := false
				for _, v := range values {
					if fmt.Sprintf("%v", v) == strValue {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf(rule.Message)
				}
			}

		case "email":
			if !strings.Contains(strValue, "@") || !strings.Contains(strValue, ".") {
				return fmt.Errorf(rule.Message)
			}

		case "url":
			if _, urlErr := url.ParseRequestURI(strValue); urlErr != nil {
				return fmt.Errorf(rule.Message)
			}
		}
	}

	return nil
}

// OpenAPIDoc represents the OpenAPI document structure
type OpenAPIDoc struct {
	OpenAPI      string                 `json:"openapi"`
	Info         OpenAPIInfo            `json:"info"`
	Servers      []OpenAPIServer        `json:"servers"`
	Paths        map[string]PathItem    `json:"paths"`
	Components   *Components            `json:"components,omitempty"`
	Security     []map[string][]string  `json:"security,omitempty"`
	Tags         []Tag                  `json:"tags,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
}

// Components holds various reusable objects for the OpenAPI Specification
type Components struct {
	Schemas         map[string]interface{}    `json:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
	Parameters      map[string]Parameter      `json:"parameters,omitempty"`
	RequestBodies   map[string]RequestBody    `json:"requestBodies,omitempty"`
	Responses       map[string]Response       `json:"responses,omitempty"`
	Headers         map[string]Header         `json:"headers,omitempty"`
	Examples        map[string]Example        `json:"examples,omitempty"`
}

// SecurityScheme defines a security scheme that can be used by the operations
type SecurityScheme struct {
	Type             string      `json:"type"`
	Description      string      `json:"description,omitempty"`
	Name             string      `json:"name,omitempty"`             // Required for apiKey
	In               string      `json:"in,omitempty"`               // Required for apiKey
	Scheme           string      `json:"scheme,omitempty"`           // Required for http
	BearerFormat     string      `json:"bearerFormat,omitempty"`     // Optional for http ("bearer")
	Flows            *OAuthFlows `json:"flows,omitempty"`            // Required for oauth2
	OpenIDConnectURL string      `json:"openIdConnectUrl,omitempty"` // Required for openIdConnect
}

// OAuthFlows allows configuration of the supported OAuth Flows
type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty"`
}

// OAuthFlow configuration details for a supported OAuth Flow
type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes"`
}

// Tag adds metadata to a single tag that is used by Operation
type Tag struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
}

// ExternalDocumentation allows referencing an external resource for extended documentation
type ExternalDocumentation struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

// Header represents a header parameter
type Header struct {
	Description string                 `json:"description,omitempty"`
	Required    bool                   `json:"required,omitempty"`
	Deprecated  bool                   `json:"deprecated,omitempty"`
	Schema      map[string]interface{} `json:"schema,omitempty"`
}

// Example represents an example value for a parameter or property
type Example struct {
	Summary       string      `json:"summary,omitempty"`
	Description   string      `json:"description,omitempty"`
	Value         interface{} `json:"value,omitempty"`
	ExternalValue string      `json:"externalValue,omitempty"`
}

type OpenAPIInfo struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
}

type Operation struct {
	Summary      string                 `json:"summary"`
	Description  string                 `json:"description"`
	OperationID  string                 `json:"operationId,omitempty"`
	Tags         []string               `json:"tags"`
	Parameters   []Parameter            `json:"parameters,omitempty"`
	RequestBody  *RequestBody           `json:"requestBody,omitempty"`
	Responses    map[string]Response    `json:"responses"`
	Deprecated   bool                   `json:"deprecated,omitempty"`
	Security     []map[string][]string  `json:"security,omitempty"`
	Servers      []OpenAPIServer        `json:"servers,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
}

type RequestBody struct {
	Content map[string]Content `json:"content"`
}

type Content struct {
	Schema map[string]interface{} `json:"schema"`
}

type Response struct {
	Description string             `json:"description"`
	Content     map[string]Content `json:"content,omitempty"`
}

// Error types
var (
	ErrRequired         = fmt.Errorf("value is required")
	ErrInvalidFormat    = fmt.Errorf("invalid format")
	ErrInvalidValue     = fmt.Errorf("invalid value")
	ErrUnsupportedType  = fmt.Errorf("unsupported type")
	ErrValidationFailed = fmt.Errorf("validation failed")
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Type    string
	Message string
	Cause   error
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation failed for field %s: %s", e.Field, e.Message)
	}
	return e.Message
}

func (e *ValidationError) Unwrap() error {
	return e.Cause
}

// SchemaError represents a schema-related error
type SchemaError struct {
	Type    string
	Message string
	Cause   error
}

func (e *SchemaError) Error() string {
	return fmt.Sprintf("schema error (%s): %s", e.Type, e.Message)
}

func (e *SchemaError) Unwrap() error {
	return e.Cause
}

// ErrInvalidType represents an error when an invalid type is provided
type ErrInvalidType struct {
	Type string
}

func (e *ErrInvalidType) Error() string {
	return fmt.Sprintf("invalid type provided: %s", e.Type)
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, errType, message string, cause error) error {
	return &ValidationError{
		Field:   field,
		Type:    errType,
		Message: message,
		Cause:   cause,
	}
}

// NewSchemaError creates a new SchemaError
func NewSchemaError(errType, message string, cause error) error {
	return &SchemaError{
		Type:    errType,
		Message: message,
		Cause:   cause,
	}
}

// Safe schema generation with error handling
func SafeSchemaFromStruct(v interface{}) (schema map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			schema = nil
			err = fmt.Errorf("panic in schema generation: %v", r)
		}
	}()

	if v == nil {
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}, nil
	}

	t := reflect.TypeOf(v)
	if t == nil {
		return nil, &ErrInvalidType{Type: "nil type"}
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t == nil {
		return nil, &ErrInvalidType{Type: "nil pointer element"}
	}

	if t.Kind() != reflect.Struct {
		return nil, &ErrInvalidType{Type: t.String()}
	}

	schema, err = SchemaFromStruct(v)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema: %w", err)
	}

	return schema, nil
}

// Generate schema from struct using reflection
func SchemaFromStruct(v interface{}) (map[string]interface{}, error) {
	if v == nil {
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}, nil
	}

	t := reflect.TypeOf(v)
	if t == nil {
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}, nil
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t == nil || t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input must be a struct type, got %v", t)
	}

	props := make(map[string]interface{})
	required := make([]string, 0)

	// Safely iterate through fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Handle omitempty and required
		isRequired := true
		if strings.Contains(jsonTag, ",") {
			parts := strings.Split(jsonTag, ",")
			jsonTag = parts[0]
			for _, opt := range parts[1:] {
				if opt == "omitempty" {
					isRequired = false
				}
			}
		}

		// Get validation tag
		validationTag := field.Tag.Get("validate")
		if validationTag != "" {
			validations := strings.Split(validationTag, ",")
			for _, v := range validations {
				if v == "required" {
					isRequired = true
					break
				}
			}
		}

		// Generate schema based on field type
		fieldSchema, err := createSchemaFromGoType(field.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to create schema for field %s: %w", field.Name, err)
		}

		if fieldSchema != nil {
			// Add description from doc tag if available
			if desc := field.Tag.Get("doc"); desc != "" {
				fieldSchema["description"] = desc
			}

			// Add example from example tag if available
			if example := field.Tag.Get("example"); example != "" {
				fieldSchema["example"] = example
			}

			// Add format from format tag if available
			if format := field.Tag.Get("format"); format != "" {
				fieldSchema["format"] = format
			}

			props[jsonTag] = fieldSchema

			if isRequired {
				required = append(required, jsonTag)
			}
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": props,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema, nil
}

// Create schema based on Go type
func createSchemaFromGoType(t reflect.Type) (map[string]interface{}, error) {
	// Handle nil type
	if t == nil {
		return map[string]interface{}{"type": "string"}, nil
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]interface{}{"type": "string"}, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return map[string]interface{}{
			"type":   "integer",
			"format": "int32",
		}, nil

	case reflect.Int64:
		return map[string]interface{}{
			"type":   "integer",
			"format": "int64",
		}, nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return map[string]interface{}{
			"type":    "integer",
			"format":  "int32",
			"minimum": 0,
		}, nil

	case reflect.Uint64:
		return map[string]interface{}{
			"type":    "integer",
			"format":  "int64",
			"minimum": 0,
		}, nil

	case reflect.Float32:
		return map[string]interface{}{
			"type":   "number",
			"format": "float",
		}, nil

	case reflect.Float64:
		return map[string]interface{}{
			"type":   "number",
			"format": "double",
		}, nil

	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}, nil

	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			// Special case for []byte
			return map[string]interface{}{
				"type":   "string",
				"format": "byte",
			}, nil
		}
		fallthrough

	case reflect.Array:
		elemType := t.Elem()
		if elemType == nil {
			return map[string]interface{}{"type": "array"}, nil
		}
		elemSchema, err := createSchemaFromGoType(elemType)
		if err != nil {
			return nil, fmt.Errorf("failed to create schema for array element: %w", err)
		}
		return map[string]interface{}{
			"type":  "array",
			"items": elemSchema,
		}, nil

	case reflect.Struct:
		// Handle special types
		switch t.String() {
		case "time.Time":
			return map[string]interface{}{
				"type":    "string",
				"format":  "date-time",
				"example": time.Now().UTC().Format(time.RFC3339),
			}, nil
		}

		// For regular structs
		v := reflect.New(t).Interface()
		if timeVal, ok := v.(interface{ Time() time.Time }); ok {
			// Handle types that implement Time() time.Time
			return map[string]interface{}{
				"type":    "string",
				"format":  "date-time",
				"example": timeVal.Time().UTC().Format(time.RFC3339),
			}, nil
		}

		schema, err := SchemaFromStruct(v)
		if err != nil {
			return nil, fmt.Errorf("failed to create schema for struct %s: %w", t.String(), err)
		}
		return schema, nil

	case reflect.Ptr:
		elemType := t.Elem()
		if elemType == nil {
			return map[string]interface{}{"type": "string"}, nil
		}
		return createSchemaFromGoType(elemType)

	case reflect.Interface:
		// For empty interfaces, we can't determine the type
		return map[string]interface{}{
			"type":                 "object",
			"additionalProperties": true,
		}, nil

	case reflect.Map:
		// For maps, we create an object with additional properties
		valueType := t.Elem()
		if valueType == nil {
			return map[string]interface{}{
				"type":                 "object",
				"additionalProperties": true,
			}, nil
		}

		valueSchema, err := createSchemaFromGoType(valueType)
		if err != nil {
			return nil, fmt.Errorf("failed to create schema for map value type: %w", err)
		}

		return map[string]interface{}{
			"type":                 "object",
			"additionalProperties": valueSchema,
		}, nil

	default:
		return map[string]interface{}{"type": "string"}, nil
	}
}

// Helper function: create API definition
func NewAPIDefinition(method, path, summary string) *APIDefinition {
	return &APIDefinition{
		Method:   method,
		Path:     path,
		Summary:  summary,
		Tags:     []string{},
		Params:   []Parameter{},
		Examples: make(map[string]Example),
		Security: make([]map[string][]string, 0),
		Servers:  make([]OpenAPIServer, 0),
	}
}

// Chain call: set operation ID
func (api *APIDefinition) WithOperationID(operationID string) *APIDefinition {
	api.OperationID = operationID
	return api
}

// Chain call: set description
func (api *APIDefinition) WithDescription(description string) *APIDefinition {
	api.Description = description
	return api
}

// Chain call: set tags
func (api *APIDefinition) WithTags(tags ...string) *APIDefinition {
	api.Tags = tags
	return api
}

// Chain call: add parameter
func (api *APIDefinition) WithParam(name, in, description string, required bool, validations ...ValidationRule) *APIDefinition {
	param := Parameter{
		Name:        name,
		In:          in,
		Required:    required,
		Description: description,
		Validations: validations,
	}
	api.Params = append(api.Params, param)
	return api
}

// Chain call: add parameter with schema
func (api *APIDefinition) WithParamSchema(name, in, description string, required bool, schema map[string]interface{}, validations ...ValidationRule) *APIDefinition {
	param := Parameter{
		Name:        name,
		In:          in,
		Required:    required,
		Description: description,
		Schema:      schema,
		Validations: validations,
	}
	api.Params = append(api.Params, param)
	return api
}

// WithParams sets multiple parameters at once
func (api *APIDefinition) WithParams(params []Parameter) *APIDefinition {
	api.Params = append(api.Params, params...)
	return api
}

// WithPathParam sets a path parameter
func (api *APIDefinition) WithPathParam(name, description string, required bool, validations ...ValidationRule) *APIDefinition {
	api.Params = append(api.Params, Parameter{
		Name:        name,
		In:          "path",
		Required:    required,
		Description: description,
		Validations: validations,
	})
	return api
}

// WithQueryParam sets a query parameter
func (api *APIDefinition) WithQueryParam(name, description string, required bool, validations ...ValidationRule) *APIDefinition {
	api.Params = append(api.Params, Parameter{
		Name:        name,
		In:          "query",
		Required:    required,
		Description: description,
		Validations: validations,
	})
	return api
}

// WithHeaderParam sets a header parameter
func (api *APIDefinition) WithHeaderParam(name, description string, required bool, validations ...ValidationRule) *APIDefinition {
	api.Params = append(api.Params, Parameter{
		Name:        name,
		In:          "header",
		Required:    required,
		Description: description,
		Validations: validations,
	})
	return api
}

// WithCookieParam sets a cookie parameter
func (api *APIDefinition) WithCookieParam(name, description string, required bool, validations ...ValidationRule) *APIDefinition {
	api.Params = append(api.Params, Parameter{
		Name:        name,
		In:          "cookie",
		Required:    required,
		Description: description,
		Validations: validations,
	})
	return api
}

// Helper function to create a validation rule
func NewValidationRule(ruleType string, value interface{}, message string) ValidationRule {
	return ValidationRule{
		Type:    ruleType,
		Value:   value,
		Message: message,
	}
}

// Chain call: set request structure
func (api *APIDefinition) WithRequest(req interface{}) *APIDefinition {
	api.Request = req
	return api
}

// Chain call: set response structure
func (api *APIDefinition) WithResponse(resp interface{}) *APIDefinition {
	api.Response = resp
	return api
}

// WithHandler sets the standard HTTP handler (used as fallback when no native handler is provided)
// For framework-specific handlers (e.g., gin.HandlerFunc), use WithNativeHandler instead
func (api *APIDefinition) WithHandler(handler http.HandlerFunc) *APIDefinition {
	api.Handler = handler
	return api
}

// WithNativeHandler sets a framework-specific handler (e.g., gin.HandlerFunc)
// This will be used in preference to the standard HTTP handler when available
func (api *APIDefinition) WithNativeHandler(handler interface{}) *APIDefinition {
	api.NativeHandler = handler
	return api
}

// Chain call: mark as deprecated
func (api *APIDefinition) WithDeprecated(deprecated bool) *APIDefinition {
	api.Deprecated = deprecated
	return api
}

// Chain call: add security requirement
func (api *APIDefinition) WithSecurity(scheme string, scopes []string) *APIDefinition {
	requirement := map[string][]string{
		scheme: scopes,
	}
	api.Security = append(api.Security, requirement)
	return api
}

// Chain call: add external documentation
func (api *APIDefinition) WithExternalDocs(description, url string) *APIDefinition {
	api.ExternalDocs = &ExternalDocumentation{
		Description: description,
		URL:         url,
	}
	return api
}

// Chain call: add example
func (api *APIDefinition) WithExample(name string, example Example) *APIDefinition {
	api.Examples[name] = example
	return api
}

// Chain call: add server
func (api *APIDefinition) WithServer(url, description string) *APIDefinition {
	server := OpenAPIServer{
		URL:         url,
		Description: description,
	}
	api.Servers = append(api.Servers, server)
	return api
}
