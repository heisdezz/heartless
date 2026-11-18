package http

import "testing"

// TestBuildURL checks query params merge with any existing query, and that
// params named ":name" fill the matching token in the path instead of becoming
// a query string entry.
func TestBuildURL(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		params map[string]string
		want   string
	}{
		{
			name:   "no params",
			raw:    "https://api.test/users",
			params: nil,
			want:   "https://api.test/users",
		},
		{
			name:   "query params",
			raw:    "https://api.test/users",
			params: map[string]string{"page": "2", "q": "hello world"},
			want:   "https://api.test/users?page=2&q=hello+world",
		},
		{
			name:   "query params merge with the URL's own query",
			raw:    "https://api.test/users?limit=10",
			params: map[string]string{"page": "2"},
			want:   "https://api.test/users?limit=10&page=2",
		},
		{
			name:   "path params replace their token",
			raw:    "https://api.test/v1/contacts/:id",
			params: map[string]string{":id": "42"},
			want:   "https://api.test/v1/contacts/42",
		},
		{
			name:   "path params alongside query params",
			raw:    "https://api.test/v1/contacts/:id?full=true",
			params: map[string]string{":id": "42", "page": "2"},
			want:   "https://api.test/v1/contacts/42?full=true&page=2",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildURL(tc.raw, tc.params)
			if err != nil {
				t.Fatalf("buildURL: %v", err)
			}
			if got != tc.want {
				t.Errorf("buildURL = %q, want %q", got, tc.want)
			}
		})
	}
}
