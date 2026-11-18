package request

import (
	"net/url"
	"strings"
	"time"
)

// HTTPMethod represents a supported HTTP method.
type HTTPMethod string

const (
	MethodGET     HTTPMethod = "GET"
	MethodPOST    HTTPMethod = "POST"
	MethodPUT     HTTPMethod = "PUT"
	MethodPATCH   HTTPMethod = "PATCH"
	MethodDELETE  HTTPMethod = "DELETE"
	MethodHEAD    HTTPMethod = "HEAD"
	MethodOPTIONS HTTPMethod = "OPTIONS"
)

// BodyType identifies the request body encoding.
type BodyType string

const (
	BodyNone      BodyType = ""
	BodyJSON      BodyType = "json"
	BodyText      BodyType = "text"
	BodyForm      BodyType = "form"
	BodyMultipart BodyType = "multipart"
	BodyBinary    BodyType = "binary"
	BodyGraphQL   BodyType = "graphql"
)

// HTTPSettings contains transport and execution settings for HTTP requests.
type HTTPSettings struct {
	Timeout         time.Duration `json:"timeout,omitempty"`
	FollowRedirects bool          `json:"follow_redirects"`
	MaxRedirects    int           `json:"max_redirects,omitempty"`
	VerifyTLS       bool          `json:"verify_tls"`
	ProxyURL        string        `json:"proxy_url,omitempty"`
	HTTPVersion     string        `json:"http_version,omitempty"`
}

// DefaultHTTPSettings returns baseline HTTP execution settings.
func DefaultHTTPSettings() HTTPSettings {
	return HTTPSettings{
		Timeout:         30 * time.Second,
		FollowRedirects: true,
		MaxRedirects:    10,
		VerifyTLS:       true,
		HTTPVersion:     "1.1",
	}
}

// HTTPRequest captures HTTP-specific request data.
type HTTPRequest struct {
	Method   string            `json:"method"`
	URL      string            `json:"url"`
	Headers  map[string]string `json:"headers,omitempty"`
	Params   map[string]string `json:"params,omitempty"`
	Body     string            `json:"body,omitempty"`
	BodyType BodyType          `json:"body_type,omitempty"`
	Settings HTTPSettings      `json:"settings,omitempty"`
}

// ApplyPathParams substitutes parameter names written as ":name" into the URL,
// replacing every occurrence of the token, with each value escaped so it stays
// a single path segment. It returns the substituted URL and the parameters
// left over, which are sent as the query string. A ":name" parameter whose
// token appears nowhere in the URL is dropped rather than appended to the
// query, where the colon would be percent-encoded.
func ApplyPathParams(rawURL string, params map[string]string) (string, map[string]string) {
	query := make(map[string]string, len(params))
	sub, replaced := rawURL, false
	for k, v := range params {
		if !strings.HasPrefix(k, ":") {
			query[k] = v
			continue
		}
		if strings.Contains(sub, k) {
			sub = strings.ReplaceAll(sub, k, url.PathEscape(v))
			replaced = true
		}
	}
	if !replaced {
		return rawURL, query
	}
	return sub, query
}

// NewHTTP creates a saved HTTP request definition.
func NewHTTP(workspaceID, name string) *Request {
	r := New(workspaceID, name, KindHTTP)
	r.Method = string(MethodGET)
	r.BodyType = string(BodyNone)
	return r
}

// NormalizeHTTP applies defaults for HTTP request fields.
func NormalizeHTTP(r *Request) {
	if r == nil {
		return
	}
	if r.Kind == "" {
		r.Kind = KindHTTP
	}
	if r.Method == "" {
		r.Method = string(MethodGET)
	}
	if r.Headers == nil {
		r.Headers = map[string]string{}
	}
	if r.Params == nil {
		r.Params = map[string]string{}
	}
}
