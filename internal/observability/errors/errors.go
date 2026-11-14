package errors

import (
	"errors"
	"fmt"
)

// Category classifies application errors for logging and handling.
type Category string

const (
	CategoryValidation      Category = "validation"
	CategoryNotFound        Category = "not_found"
	CategoryConflict        Category = "conflict"
	CategoryAuth            Category = "auth"
	CategoryVariable        Category = "variable_resolution"
	CategoryNetwork         Category = "network"
	CategoryDNS             Category = "dns"
	CategoryConnect         Category = "connect"
	CategoryTLS             Category = "tls"
	CategoryTimeout         Category = "timeout"
	CategoryCanceled         Category = "canceled"
	CategoryHTTPExecution   Category = "http_execution"
	CategoryProtocol        Category = "protocol"
	CategoryResponse        Category = "response_processing"
	CategoryStorage         Category = "storage"
	CategoryMigration       Category = "migration"
	CategoryScript          Category = "script"
	CategoryImport          Category = "import"
	CategoryExport          Category = "export"
	CategoryConfig          Category = "config"
	CategoryInternal        Category = "internal"
)

// Error is a structured, categorizable error.
type Error struct {
	Category Category
	Message  string
	Err      error
	Fields   map[string]any
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	msg := e.Message
	if msg == "" && e.Err != nil {
		msg = e.Err.Error()
	}
	if msg == "" {
		msg = string(e.Category)
	}
	if e.Category == "" {
		return msg
	}
	if e.Err != nil && e.Message != "" {
		return fmt.Sprintf("%s: %s: %v", e.Category, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Category, msg)
}

func (e *Error) Unwrap() error { return e.Err }

// New creates a categorized error.
func New(category Category, message string) *Error {
	return &Error{Category: category, Message: message}
}

// Wrap wraps an underlying error with a category and message.
func Wrap(category Category, err error, message string) *Error {
	if err == nil {
		return nil
	}
	return &Error{Category: category, Message: message, Err: err}
}

// WithField returns a copy-like error with an attached field (shallow).
func WithField(err error, key string, value any) error {
	var e *Error
	if errors.As(err, &e) {
		out := *e
		if out.Fields == nil {
			out.Fields = map[string]any{}
		} else {
			m := make(map[string]any, len(out.Fields)+1)
			for k, v := range out.Fields {
				m[k] = v
			}
			out.Fields = m
		}
		out.Fields[key] = value
		return &out
	}
	return err
}

// GetCategory extracts a category from an error, defaulting to internal.
func GetCategory(err error) Category {
	var e *Error
	if errors.As(err, &e) && e.Category != "" {
		return e.Category
	}
	return CategoryInternal
}

// Fields extracts structured fields from an error if present.
func Fields(err error) map[string]any {
	var e *Error
	if errors.As(err, &e) && e.Fields != nil {
		return e.Fields
	}
	return nil
}

// IsCategory reports whether err has the given category.
func IsCategory(err error, category Category) bool {
	return GetCategory(err) == category
}
