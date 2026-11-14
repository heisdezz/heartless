package request

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidationError represents a structured request validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors collects multiple validation errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	parts := make([]string, 0, len(e))
	for _, err := range e {
		parts = append(parts, err.Error())
	}
	return strings.Join(parts, "; ")
}

// Has reports whether any validation errors are present.
func (e ValidationErrors) Has() bool { return len(e) > 0 }

var allowedMethods = map[string]struct{}{
	"GET": {}, "POST": {}, "PUT": {}, "PATCH": {}, "DELETE": {}, "HEAD": {}, "OPTIONS": {},
}

// ValidateHTTP validates an HTTP request definition and returns structured errors.
func ValidateHTTP(r *Request, settings HTTPSettings) ValidationErrors {
	var errs ValidationErrors
	if r == nil {
		errs = append(errs, ValidationError{Field: "request", Code: "required", Message: "request is required"})
		return errs
	}
	if r.WorkspaceID == "" {
		errs = append(errs, ValidationError{Field: "workspace_id", Code: "required", Message: "workspace_id is required"})
	}
	if strings.TrimSpace(r.Name) == "" {
		errs = append(errs, ValidationError{Field: "name", Code: "required", Message: "name is required"})
	}
	if r.Kind != "" && r.Kind != KindHTTP && r.Kind != KindGraphQL {
		errs = append(errs, ValidationError{Field: "kind", Code: "invalid", Message: "kind must be http or graphql"})
	}

	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method == "" {
		errs = append(errs, ValidationError{Field: "method", Code: "required", Message: "method is required"})
	} else if _, ok := allowedMethods[method]; !ok {
		errs = append(errs, ValidationError{Field: "method", Code: "invalid", Message: fmt.Sprintf("unsupported method: %s", method)})
	}

	rawURL := strings.TrimSpace(r.URL)
	if rawURL == "" {
		errs = append(errs, ValidationError{Field: "url", Code: "required", Message: "url is required"})
	} else {
		u, err := url.Parse(rawURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			errs = append(errs, ValidationError{Field: "url", Code: "invalid", Message: "url must be a valid absolute URL"})
		} else if u.Scheme != "http" && u.Scheme != "https" {
			errs = append(errs, ValidationError{Field: "url", Code: "invalid_scheme", Message: "url scheme must be http or https"})
		}
	}

	if settings.Timeout < 0 {
		errs = append(errs, ValidationError{Field: "settings.timeout", Code: "invalid", Message: "timeout must be >= 0"})
	}
	if settings.MaxRedirects < 0 {
		errs = append(errs, ValidationError{Field: "settings.max_redirects", Code: "invalid", Message: "max_redirects must be >= 0"})
	}
	return errs
}
