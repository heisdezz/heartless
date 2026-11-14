// Package postman imports Postman Collection v2.0/v2.1 JSON documents.
package postman

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"heartless/internal/importer"
)

// Schema URLs accepted by Detect/Parse.
const (
	schemaV200 = "https://schema.getpostman.com/json/collection/v2.0.0/collection.json"
	schemaV210 = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
)

// Importer implements importer.Importer for Postman collections.
type Importer struct{}

// Format implements importer.Importer.
func (Importer) Format() importer.Format { return importer.FormatPostman }

// Detect implements importer.Importer.
func (Importer) Detect(data []byte) bool {
	if !json.Valid(data) {
		return false
	}
	var probe struct {
		Info struct {
			Schema importer.Text `json:"schema"`
		} `json:"info"`
		Item json.RawMessage `json:"item"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return false
	}
	if !validSchema(probe.Info.Schema) || !importer.Present(probe.Item) {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(string(probe.Item)), "[")
}

// Parse implements importer.Importer.
func (Importer) Parse(data []byte) (*importer.Draft, error) {
	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("postman: parse collection: %w", err)
	}
	if !validSchema(doc.Info.Schema) || doc.Item == nil {
		return nil, errors.New("postman: document is not a v2.0/v2.1 collection")
	}

	draft := &importer.Draft{
		Format:      importer.FormatPostman,
		Name:        doc.Info.Name.String(),
		Description: importer.Description(doc.Info.Description),
	}
	col := importer.Collection{
		Name:        doc.Info.Name.String(),
		Description: draft.Description,
	}

	if len(doc.Event) > 0 {
		draft.Issues = append(draft.Issues, importer.Issue{
			Severity: importer.SeverityUnsupported,
			Path:     "event",
			Message:  "collection-level scripts (prerequest/test) are not imported",
		})
	}
	if len(doc.Variable) > 0 {
		draft.Issues = append(draft.Issues, importer.Issue{
			Severity: importer.SeverityUnsupported,
			Path:     "variable",
			Message: fmt.Sprintf("%d collection variable(s) not imported; environment/variable import is pending",
				len(doc.Variable)),
		})
	}
	issueAuth(&draft.Issues, doc.Auth, "auth")

	for i := range doc.Item {
		col.Items = append(col.Items, mapItem(&doc.Item[i], fmt.Sprintf("item[%d]", i), draft))
	}
	draft.Collections = append(draft.Collections, col)
	return draft, nil
}

func validSchema(s importer.Text) bool {
	return s == schemaV200 || s == schemaV210
}

// document is the root of a Postman collection document.
type document struct {
	Info struct {
		Schema      importer.Text   `json:"schema"`
		Name        importer.Text   `json:"name"`
		Description json.RawMessage `json:"description"`
	} `json:"info"`
	Item     []item          `json:"item"`
	Variable []struct{}      `json:"variable"` // only counted for reporting
	Auth     json.RawMessage `json:"auth"`
	Event    []struct{}      `json:"event"`
}

// item is either a folder (has "item") or a request (has "request").
type item struct {
	Name        importer.Text   `json:"name"`
	Description json.RawMessage `json:"description"`
	Item        json.RawMessage `json:"item"`
	Request     json.RawMessage `json:"request"`
	Auth        json.RawMessage `json:"auth"`
	Event       []struct{}      `json:"event"`
}

func mapItem(it *item, path string, draft *importer.Draft) importer.Item {
	switch {
	case importer.Present(it.Item):
		var children []item
		folder := &importer.Folder{
			Name:        it.Name.String(),
			Description: importer.Description(it.Description),
		}
		if err := json.Unmarshal(it.Item, &children); err != nil {
			draft.Issues = append(draft.Issues, importer.Issue{
				Severity: importer.SeverityError,
				Path:     path + ".item",
				Message:  fmt.Sprintf("folder %q children unreadable: %v", folder.Name, err),
			})
			return importer.Item{Folder: folder}
		}
		out := importer.Item{Folder: folder}
		for i := range children {
			out.Children = append(out.Children, mapItem(&children[i], fmt.Sprintf("%s.item[%d]", path, i), draft))
		}
		return out
	case importer.Present(it.Request):
		req := mapRequest(it, path, draft)
		return importer.Item{Request: req}
	default:
		draft.Issues = append(draft.Issues, importer.Issue{
			Severity: importer.SeverityWarning,
			Path:     path,
			Message:  fmt.Sprintf("item %q has neither item nor request and was skipped", it.Name),
		})
		return importer.Item{}
	}
}

type postmanRequest struct {
	Method      importer.Text   `json:"method"`
	Description json.RawMessage `json:"description"`
	URL         json.RawMessage `json:"url"`
	Header      []header        `json:"header"`
	Body        json.RawMessage `json:"body"`
	Auth        json.RawMessage `json:"auth"`
}

type header struct {
	Key      importer.Text `json:"key"`
	Value    importer.Text `json:"value"`
	Disabled *bool         `json:"disabled"`
}

func mapRequest(it *item, path string, draft *importer.Draft) *importer.Request {
	reqPath := path + ".request"
	var r postmanRequest
	if err := json.Unmarshal(it.Request, &r); err != nil {
		draft.Issues = append(draft.Issues, importer.Issue{
			Severity: importer.SeverityError,
			Path:     reqPath,
			Message:  fmt.Sprintf("request %q unreadable: %v", it.Name, err),
		})
		return &importer.Request{Name: it.Name.String(), Description: importer.Description(it.Description)}
	}

	rawURL, params := convertURL(r.URL, reqPath, draft)
	body, bodyType := convertBody(r.Body, reqPath, draft)

	out := &importer.Request{
		Kind: "http",
		Name: it.Name.String(),
		// Postman puts request descriptions on the request object; some
		// exporters put them on the item instead.
		Description: importer.FirstNonEmpty(importer.Description(r.Description), importer.Description(it.Description)),
		Method:      r.Method.String(),
		URL:         rawURL,
		Headers:     convertHeaders(r.Header),
		Params:      params,
		Body:        body,
		BodyType:    bodyType,
	}
	issueAuth(&draft.Issues, r.Auth, reqPath+".auth")
	if len(it.Event) > 0 {
		draft.Issues = append(draft.Issues, importer.Issue{
			Severity: importer.SeverityUnsupported,
			Path:     reqPath + ".event",
			Message:  fmt.Sprintf("scripts on request %q are not imported", out.Name),
		})
	}
	return out
}

func convertHeaders(raw []header) []importer.NameValue {
	if len(raw) == 0 {
		return nil
	}
	out := make([]importer.NameValue, 0, len(raw))
	for _, h := range raw {
		if h.Key == "" && h.Value == "" {
			continue
		}
		out = append(out, importer.NameValue{
			Name:    h.Key.String(),
			Value:   h.Value.String(),
			Enabled: !disabled(h.Disabled),
		})
	}
	return out
}

// convertURL renders a Postman URL (string or object form) and collects its
// query parameters plus path variables (exposed as ":name" parameters).
func convertURL(raw json.RawMessage, path string, draft *importer.Draft) (string, []importer.NameValue) {
	if !importer.Present(raw) {
		return "", nil
	}
	var s importer.Text
	if err := json.Unmarshal(raw, &s); err == nil {
		return s.String(), nil
	}

	var o struct {
		Protocol importer.Text   `json:"protocol"`
		Host     json.RawMessage `json:"host"`
		Port     importer.Text   `json:"port"`
		Path     json.RawMessage `json:"path"`
		Query    []kv            `json:"query"`
		Variable []kv            `json:"variable"`
		Hash     importer.Text   `json:"hash"`
	}
	if err := json.Unmarshal(raw, &o); err != nil {
		draft.Issues = append(draft.Issues, importer.Issue{
			Severity: importer.SeverityError,
			Path:     path + ".url",
			Message:  fmt.Sprintf("url unreadable: %v", err),
		})
		return "", nil
	}

	var sb strings.Builder
	if o.Protocol != "" {
		sb.WriteString(o.Protocol.String())
		sb.WriteString("://")
	}
	sb.WriteString(joinDotted(o.Host))
	if o.Port != "" {
		sb.WriteByte(':')
		sb.WriteString(o.Port.String())
	}
	if p := joinSlashed(o.Path); p != "" {
		sb.WriteByte('/')
		sb.WriteString(p)
	}
	if o.Hash != "" {
		sb.WriteByte('#')
		sb.WriteString(o.Hash.String())
	}

	var params []importer.NameValue
	for _, q := range o.Query {
		params = append(params, importer.NameValue{
			Name:    q.Key.String(),
			Value:   q.Value.String(),
			Enabled: !disabled(q.Disabled),
		})
	}
	for _, v := range o.Variable {
		params = append(params, importer.NameValue{
			Name:    ":" + v.Key.String(),
			Value:   v.Value.String(),
			Enabled: !disabled(v.Disabled),
		})
	}
	return sb.String(), params
}

type kv struct {
	Key      importer.Text `json:"key"`
	Value    importer.Text `json:"value"`
	Disabled *bool         `json:"disabled"`
}

// joinDotted renders host, which Postman writes as a string or a label array.
func joinDotted(raw json.RawMessage) string {
	if !importer.Present(raw) {
		return ""
	}
	var s importer.Text
	if err := json.Unmarshal(raw, &s); err == nil {
		return s.String()
	}
	var labels []importer.Text
	if err := json.Unmarshal(raw, &labels); err != nil {
		return ""
	}
	parts := make([]string, 0, len(labels))
	for _, l := range labels {
		parts = append(parts, l.String())
	}
	return strings.Join(parts, ".")
}

// joinSlashed renders path, which Postman writes as a string or segment array.
func joinSlashed(raw json.RawMessage) string {
	if !importer.Present(raw) {
		return ""
	}
	var s importer.Text
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimPrefix(s.String(), "/")
	}
	var segs []importer.Text
	if err := json.Unmarshal(raw, &segs); err != nil {
		return ""
	}
	parts := make([]string, 0, len(segs))
	for _, seg := range segs {
		if seg == "" {
			continue
		}
		parts = append(parts, strings.TrimPrefix(seg.String(), "/"))
	}
	return strings.Join(parts, "/")
}

type postmanBody struct {
	Mode       importer.Text `json:"mode"`
	Raw        importer.Text `json:"raw"`
	URLencoded []kv          `json:"urlencoded"`
	Formdata   []formField   `json:"formdata"`
	File       *struct {
		Src importer.Text `json:"src"`
	} `json:"file"`
	GraphQL *struct {
		Query     importer.Text   `json:"query"`
		Variables json.RawMessage `json:"variables"`
	} `json:"graphql"`
	Options *struct {
		Raw *struct {
			Language importer.Text `json:"language"`
		} `json:"raw"`
	} `json:"options"`
}

type formField struct {
	Key         importer.Text `json:"key"`
	Value       importer.Text `json:"value"`
	Disabled    *bool         `json:"disabled"`
	Src         importer.Text `json:"src"`
	ContentType importer.Text `json:"contentType"`
}

// formEntry is the JSON body used for multipart/form-data imports.
type formEntry struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

// convertBody maps a Postman body onto body text plus a content type.
// Unsupported modes (file bodies) are reported as issues.
func convertBody(raw json.RawMessage, path string, draft *importer.Draft) (string, string) {
	if !importer.Present(raw) {
		return "", ""
	}
	var b postmanBody
	if err := json.Unmarshal(raw, &b); err != nil {
		draft.Issues = append(draft.Issues, importer.Issue{
			Severity: importer.SeverityError,
			Path:     path + ".body",
			Message:  fmt.Sprintf("body unreadable: %v", err),
		})
		return "", ""
	}

	switch b.Mode {
	case "raw":
		bodyType := "text/plain"
		if b.Options != nil && b.Options.Raw != nil &&
			strings.EqualFold(b.Options.Raw.Language.String(), "json") {
			bodyType = "application/json"
		}
		return b.Raw.String(), bodyType
	case "urlencoded":
		pairs := make([]string, 0, len(b.URLencoded))
		for _, f := range b.URLencoded {
			if disabled(f.Disabled) {
				continue
			}
			pairs = append(pairs, url.QueryEscape(f.Key.String())+"="+url.QueryEscape(f.Value.String()))
		}
		return strings.Join(pairs, "&"), "application/x-www-form-urlencoded"
	case "formdata":
		entries := make([]formEntry, 0, len(b.Formdata))
		for _, f := range b.Formdata {
			if f.Src != "" {
				draft.Issues = append(draft.Issues, importer.Issue{
					Severity: importer.SeverityUnsupported,
					Path:     path + ".body",
					Message:  fmt.Sprintf("file form field %q not imported", f.Key),
				})
				continue
			}
			entries = append(entries, formEntry{
				Name:    f.Key.String(),
				Value:   f.Value.String(),
				Enabled: !disabled(f.Disabled),
			})
		}
		encoded, err := json.Marshal(entries)
		if err != nil {
			return "", "multipart/form-data"
		}
		return string(encoded), "multipart/form-data"
	case "graphql":
		out, err := json.Marshal(struct {
			Query     string          `json:"query"`
			Variables json.RawMessage `json:"variables"`
		}{Query: b.GraphQLQueryString(), Variables: graphqlVariables(&b)})
		if err != nil {
			return "", "graphql"
		}
		return string(out), "graphql"
	case "file":
		src := ""
		if b.File != nil {
			src = b.File.Src.String()
		}
		draft.Issues = append(draft.Issues, importer.Issue{
			Severity: importer.SeverityUnsupported,
			Path:     path + ".body",
			Message:  fmt.Sprintf("binary file body (%q) not imported", src),
		})
		return "", ""
	case "":
		return "", ""
	default:
		draft.Issues = append(draft.Issues, importer.Issue{
			Severity: importer.SeverityWarning,
			Path:     path + ".body",
			Message:  fmt.Sprintf("unknown body mode %q ignored", b.Mode),
		})
		return "", ""
	}
}

// GraphQLQueryString returns the query text, tolerating a nil graphql block.
func (b *postmanBody) GraphQLQueryString() string {
	if b.GraphQL == nil {
		return ""
	}
	return b.GraphQL.Query.String()
}

// graphqlVariables normalizes Postman's graphql variables, which is a string
// containing JSON. Objects are kept; anything else becomes an empty object.
func graphqlVariables(b *postmanBody) json.RawMessage {
	if b.GraphQL == nil || !importer.Present(b.GraphQL.Variables) {
		return json.RawMessage("{}")
	}
	raw := b.GraphQL.Variables
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		raw = json.RawMessage(strings.TrimSpace(s))
	}
	if len(raw) > 0 && raw[0] == '{' && json.Valid(raw) {
		return raw
	}
	return json.RawMessage("{}")
}

// issueAuth reports authentication that the domain model cannot represent yet.
func issueAuth(issues *[]importer.Issue, raw json.RawMessage, path string) {
	if !importer.Present(raw) {
		return
	}
	var a struct {
		Type importer.Text `json:"type"`
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		return
	}
	t := a.Type.String()
	if t == "" || t == "noauth" {
		return
	}
	*issues = append(*issues, importer.Issue{
		Severity: importer.SeverityUnsupported,
		Path:     path,
		Message:  fmt.Sprintf("authentication type %q not imported; auth support is pending", t),
	})
}

func disabled(v *bool) bool { return v != nil && *v }
