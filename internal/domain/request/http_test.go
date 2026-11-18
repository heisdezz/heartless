package request

import "testing"

// TestApplyPathParams checks ":name" parameters fill the matching token in the
// URL, everything else stays a query parameter, and an unmatched ":name" is
// dropped rather than percent-encoded into the query.
func TestApplyPathParams(t *testing.T) {
	cases := []struct {
		name      string
		url       string
		params    map[string]string
		wantURL   string
		wantQuery map[string]string
	}{
		{
			name:      "path param replaces token",
			url:       "https://api.test/v1/contacts/:id",
			params:    map[string]string{":id": "42"},
			wantURL:   "https://api.test/v1/contacts/42",
			wantQuery: map[string]string{},
		},
		{
			name:      "plain params stay query params",
			url:       "https://api.test/contacts",
			params:    map[string]string{"page": "2"},
			wantURL:   "https://api.test/contacts",
			wantQuery: map[string]string{"page": "2"},
		},
		{
			name:      "mixed path and query params",
			url:       "https://api.test/v1/contacts/:id",
			params:    map[string]string{":id": "42", "page": "2"},
			wantURL:   "https://api.test/v1/contacts/42",
			wantQuery: map[string]string{"page": "2"},
		},
		{
			name:      "values are escaped for a path segment",
			url:       "https://api.test/v1/contacts/:id",
			params:    map[string]string{":id": "a/b c"},
			wantURL:   "https://api.test/v1/contacts/a%2Fb%20c",
			wantQuery: map[string]string{},
		},
		{
			name:      "unmatched path param is dropped",
			url:       "https://api.test/contacts",
			params:    map[string]string{":id": "42"},
			wantURL:   "https://api.test/contacts",
			wantQuery: map[string]string{},
		},
		{
			name:      "every occurrence of the token is replaced",
			url:       "https://api.test/:org/items/:org",
			params:    map[string]string{":org": "acme"},
			wantURL:   "https://api.test/acme/items/acme",
			wantQuery: map[string]string{},
		},
		{
			name:      "no params leaves the URL alone",
			url:       "https://api.test/contacts/:id",
			params:    nil,
			wantURL:   "https://api.test/contacts/:id",
			wantQuery: map[string]string{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotURL, gotQuery := ApplyPathParams(tc.url, tc.params)
			if gotURL != tc.wantURL {
				t.Errorf("url = %q, want %q", gotURL, tc.wantURL)
			}
			if len(gotQuery) != len(tc.wantQuery) {
				t.Fatalf("query = %v, want %v", gotQuery, tc.wantQuery)
			}
			for k, v := range tc.wantQuery {
				if gotQuery[k] != v {
					t.Errorf("query[%q] = %q, want %q", k, gotQuery[k], v)
				}
			}
		})
	}
}
