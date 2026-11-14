// Package insomnia imports Insomnia export documents: the v4 format
// (root "resources" array with "_type": "export") and the v5 format
// (root "collection" array). JSON documents only.
package insomnia

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"heartless/internal/importer"
)

// Importer implements importer.Importer for Insomnia exports.
type Importer struct{}

// Format implements importer.Importer.
func (Importer) Format() importer.Format { return importer.FormatInsomnia }

// Detect implements importer.Importer.
func (Importer) Detect(data []byte) bool {
	if !json.Valid(data) {
		return false
	}
	var probe struct {
		Type         importer.Text   `json:"type"`            // v5 marker
		UnderType    importer.Text   `json:"_type"`           // v4 marker ("export")
		ExportFormat json.RawMessage `json:"__export_format"` // v4 marker
		Meta         json.RawMessage `json:"meta"`            // v5 marker
		Resources    json.RawMessage `json:"resources"`
		Collection   json.RawMessage `json:"collection"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return false
	}
	if importer.Present(probe.Resources) {
		return probe.UnderType == "export" || importer.Present(probe.ExportFormat)
	}
	if isArray(probe.Collection) {
		return probe.Type == "collection" || importer.Present(probe.Meta)
	}
	return false
}

// Parse implements importer.Importer.
func (Importer) Parse(data []byte) (*importer.Draft, error) {
	var probe struct {
		Resources  json.RawMessage `json:"resources"`
		Collection json.RawMessage `json:"collection"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("insomnia: parse export: %w", err)
	}
	switch {
	case importer.Present(probe.Resources):
		return parseV4(data)
	case importer.Present(probe.Collection):
		return parseV5(data)
	default:
		return nil, errors.New("insomnia: document has neither resources nor collection")
	}
}

func isArray(raw json.RawMessage) bool {
	return importer.Present(raw) && strings.HasPrefix(strings.TrimSpace(string(raw)), "[")
}

// ---------------------------------------------------------------------------
// Shared request/body mapping (both export versions)
// ---------------------------------------------------------------------------

type insNameValue struct {
	Name     importer.Text `json:"name"`
	Value    importer.Text `json:"value"`
	Disabled *bool         `json:"disabled"`
}

type insBodyParam struct {
	Name     importer.Text `json:"name"`
	Value    importer.Text `json:"value"`
	FileName importer.Text `json:"fileName"`
	Disabled *bool         `json:"disabled"`
}

type insBody struct {
	MimeType importer.Text  `json:"mimeType"`
	Text     importer.Text  `json:"text"`
	Params   []insBodyParam `json:"params"`
	FileName importer.Text  `json:"fileName"`
}

type insAuth struct {
	Type importer.Text `json:"type"`
}

// formEntry is the JSON body used for multipart/form-data imports.
type formEntry struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

func convertHeaders(raw []insNameValue) []importer.NameValue {
	if len(raw) == 0 {
		return nil
	}
	out := make([]importer.NameValue, 0, len(raw))
	for _, h := range raw {
		if h.Name == "" && h.Value == "" {
			continue
		}
		out = append(out, importer.NameValue{
			Name:    h.Name.String(),
			Value:   h.Value.String(),
			Enabled: !isDisabled(h.Disabled),
		})
	}
	return out
}

func convertParams(raw []insNameValue) []importer.NameValue {
	return convertHeaders(raw)
}

// convertBody maps an Insomnia body onto body text plus a content type.
func convertBody(b *insBody, path string, draft *importer.Draft) (string, string) {
	if b == nil {
		return "", ""
	}
	mime := strings.ToLower(strings.TrimSpace(b.MimeType.String()))
	if i := strings.Index(mime, ";"); i >= 0 {
		mime = strings.TrimSpace(mime[:i])
	}

	switch {
	case mime == "" && b.Text == "" && len(b.Params) == 0:
		return "", ""
	case mime == "application/octet-stream":
		draft.Issues = append(draft.Issues, importer.Issue{
			Severity: importer.SeverityUnsupported,
			Path:     path + ".body",
			Message:  fmt.Sprintf("binary file body (%q) not imported", b.FileName),
		})
		return "", ""
	case mime == "application/x-www-form-urlencoded":
		pairs := make([]string, 0, len(b.Params))
		for _, p := range b.Params {
			if isDisabled(p.Disabled) {
				continue
			}
			pairs = append(pairs, url.QueryEscape(p.Name.String())+"="+url.QueryEscape(p.Value.String()))
		}
		return strings.Join(pairs, "&"), mime
	case mime == "multipart/form-data":
		entries := make([]formEntry, 0, len(b.Params))
		for _, p := range b.Params {
			if p.FileName != "" {
				draft.Issues = append(draft.Issues, importer.Issue{
					Severity: importer.SeverityUnsupported,
					Path:     path + ".body",
					Message:  fmt.Sprintf("file form field %q not imported", p.Name),
				})
				continue
			}
			entries = append(entries, formEntry{
				Name:    p.Name.String(),
				Value:   p.Value.String(),
				Enabled: !isDisabled(p.Disabled),
			})
		}
		encoded, err := json.Marshal(entries)
		if err != nil {
			return "", mime
		}
		return string(encoded), mime
	case mime == "application/graphql":
		return convertGraphQL(b.Text.String())
	case mime == "application/json" || strings.HasSuffix(mime, "+json"):
		return b.Text.String(), "application/json"
	case mime == "text/xml" || mime == "application/xml" || strings.HasSuffix(mime, "+xml"):
		return b.Text.String(), "text/xml"
	case b.Text != "":
		return b.Text.String(), mime
	default:
		return "", ""
	}
}

// convertGraphQL normalizes an Insomnia GraphQL body: either a JSON envelope
// ({query, variables, operationName}) or a raw GraphQL document.
func convertGraphQL(text string) (string, string) {
	var envelope struct {
		Query         importer.Text   `json:"query"`
		Variables     json.RawMessage `json:"variables"`
		OperationName importer.Text   `json:"operationName"`
	}
	if err := json.Unmarshal([]byte(text), &envelope); err == nil && envelope.Query != "" {
		vars := json.RawMessage("{}")
		if importer.Present(envelope.Variables) {
			vars = envelope.Variables
		}
		out, err := json.Marshal(struct {
			Query         string          `json:"query"`
			Variables     json.RawMessage `json:"variables"`
			OperationName string          `json:"operationName,omitempty"`
		}{
			Query:         envelope.Query.String(),
			Variables:     vars,
			OperationName: envelope.OperationName.String(),
		})
		if err == nil {
			return string(out), "graphql"
		}
	}
	out, err := json.Marshal(struct {
		Query     string          `json:"query"`
		Variables json.RawMessage `json:"variables"`
	}{Query: text, Variables: json.RawMessage("{}")})
	if err != nil {
		return text, "graphql"
	}
	return string(out), "graphql"
}

// issueAuth reports authentication that the domain model cannot represent yet.
// path is the full source path of the authentication field.
func issueAuth(draft *importer.Draft, auth *insAuth, path string) {
	if auth == nil || auth.Type == "" || auth.Type == "noauth" {
		return
	}
	draft.Issues = append(draft.Issues, importer.Issue{
		Severity: importer.SeverityUnsupported,
		Path:     path,
		Message:  fmt.Sprintf("authentication type %q not imported; auth support is pending", auth.Type),
	})
}

func isDisabled(v *bool) bool { return v != nil && *v }

// tally accumulates skipped-resource counts so each type is reported once.
type tally struct {
	counts map[string]int
}

func newTally() *tally { return &tally{counts: map[string]int{}} }

func (t *tally) add(typ string) { t.counts[typ]++ }

// emit appends one issue per skipped type, in stable type order.
func (t *tally) emit(draft *importer.Draft, severity, hint string) {
	if len(t.counts) == 0 {
		return
	}
	types := make([]string, 0, len(t.counts))
	for typ := range t.counts {
		types = append(types, typ)
	}
	sort.Strings(types)
	for _, typ := range types {
		msg := fmt.Sprintf("%d %q resource(s) skipped", t.counts[typ], typ)
		if hint != "" {
			msg = fmt.Sprintf("%d %q resource(s) not imported; %s", t.counts[typ], typ, hint)
		}
		draft.Issues = append(draft.Issues, importer.Issue{
			Severity: severity,
			Path:     typ,
			Message:  msg,
		})
	}
}

// ---------------------------------------------------------------------------
// v4: root "resources" array
// ---------------------------------------------------------------------------

type v4Doc struct {
	Resources []v4Resource `json:"resources"`
}

type v4Resource struct {
	Type           importer.Text   `json:"_type"`
	ID             importer.Text   `json:"_id"`
	ParentID       importer.Text   `json:"parentId"`
	Name           importer.Text   `json:"name"`
	Description    json.RawMessage `json:"description"`
	Method         importer.Text   `json:"method"`
	URL            importer.Text   `json:"url"`
	Headers        []insNameValue  `json:"headers"`
	Parameters     []insNameValue  `json:"parameters"`
	Body           *insBody        `json:"body"`
	Authentication *insAuth        `json:"authentication"`
}

func parseV4(data []byte) (*importer.Draft, error) {
	var doc v4Doc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("insomnia: parse v4 resources: %w", err)
	}

	var workspaces []*v4Resource
	byParent := map[string][]*v4Resource{}
	for i := range doc.Resources {
		r := &doc.Resources[i]
		if r.Type == "workspace" {
			workspaces = append(workspaces, r)
			continue
		}
		byParent[r.ParentID.String()] = append(byParent[r.ParentID.String()], r)
	}
	if len(workspaces) == 0 {
		return nil, errors.New("insomnia: export contains no workspace")
	}

	draft := &importer.Draft{
		Format: importer.FormatInsomnia,
		Name:   workspaces[0].Name.String(),
	}
	unsupported := newTally()
	unknown := newTally()
	visited := map[string]bool{}

	var walk func(parentID, path string) []importer.Item
	walk = func(parentID, path string) []importer.Item {
		var items []importer.Item
		for i, r := range byParent[parentID] {
			p := fmt.Sprintf("%s[%d]", path, i)
			visited[r.ID.String()] = true
			switch r.Type {
			case "request_group":
				node := importer.Item{Folder: &importer.Folder{
					Name:        r.Name.String(),
					Description: importer.Description(r.Description),
				}}
				node.Children = walk(r.ID.String(), p)
				items = append(items, node)
			case "request":
				items = append(items, importer.Item{Request: mapV4Request(r, p, draft)})
			case "grpc_request":
				unsupported.add(r.Type.String())
			case "websocket_request":
				unsupported.add(r.Type.String())
			case "environment":
				unsupported.add(r.Type.String())
			case "api_spec":
				unsupported.add(r.Type.String())
			default:
				unknown.add(r.Type.String())
			}
		}
		return items
	}

	for _, w := range workspaces {
		col := importer.Collection{
			Name:        w.Name.String(),
			Description: importer.Description(w.Description),
			Items:       walk(w.ID.String(), "resources"),
		}
		draft.Collections = append(draft.Collections, col)
	}

	// Resources whose parent is not a workspace or reachable group.
	orphaned := 0
	for i := range doc.Resources {
		r := &doc.Resources[i]
		if r.Type == "workspace" || visited[r.ID.String()] {
			continue
		}
		if r.Type == "request" || r.Type == "request_group" {
			orphaned++
			continue
		}
		unsupported.add(r.Type.String())
	}
	if orphaned > 0 {
		draft.Issues = append(draft.Issues, importer.Issue{
			Severity: importer.SeverityWarning,
			Path:     "resources",
			Message:  fmt.Sprintf("%d request/folder resource(s) skipped (missing parent)", orphaned),
		})
	}

	unsupported.emit(draft, importer.SeverityUnsupported, "import support is pending")
	unknown.emit(draft, importer.SeverityWarning, "")
	return draft, nil
}

func mapV4Request(r *v4Resource, path string, draft *importer.Draft) *importer.Request {
	body, bodyType := convertBody(r.Body, path, draft)
	out := &importer.Request{
		Kind:        "http",
		Name:        r.Name.String(),
		Description: importer.Description(r.Description),
		Method:      r.Method.String(),
		URL:         r.URL.String(),
		Headers:     convertHeaders(r.Headers),
		Params:      convertParams(r.Parameters),
		Body:        body,
		BodyType:    bodyType,
	}
	issueAuth(draft, r.Authentication, path+".authentication")
	return out
}

// ---------------------------------------------------------------------------
// v5: root "collection" array
// ---------------------------------------------------------------------------

type v5Doc struct {
	Type           importer.Text   `json:"type"`
	Name           importer.Text   `json:"name"`
	Meta           v5Meta          `json:"meta"`
	Collection     []v5Item        `json:"collection"`
	Environments   json.RawMessage `json:"environments"`
	Headers        []insNameValue  `json:"headers"`
	Authentication *insAuth        `json:"authentication"`
}

type v5Meta struct {
	ID          importer.Text   `json:"id"`
	Description json.RawMessage `json:"description"`
}

type v5Item struct {
	Meta           v5Meta          `json:"meta"`
	Name           importer.Text   `json:"name"`
	Description    json.RawMessage `json:"description"`
	Children       *[]v5Item       `json:"children"`
	Method         importer.Text   `json:"method"`
	URL            importer.Text   `json:"url"`
	ProtoFileID    importer.Text   `json:"protoFileId"`
	Headers        []insNameValue  `json:"headers"`
	Parameters     []insNameValue  `json:"parameters"`
	Body           *insBody        `json:"body"`
	Authentication *insAuth        `json:"authentication"`
}

func parseV5(data []byte) (*importer.Draft, error) {
	var doc v5Doc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("insomnia: parse v5 collection: %w", err)
	}

	draft := &importer.Draft{
		Format:      importer.FormatInsomnia,
		Name:        doc.Name.String(),
		Description: importer.Description(doc.Meta.Description),
	}
	if importer.Present(doc.Environments) {
		draft.Issues = append(draft.Issues, importer.Issue{
			Severity: importer.SeverityUnsupported,
			Path:     "environments",
			Message:  "environments not imported; environment/variable import is pending",
		})
	}
	if len(doc.Headers) > 0 {
		draft.Issues = append(draft.Issues, importer.Issue{
			Severity: importer.SeverityUnsupported,
			Path:     "headers",
			Message:  "workspace-level headers not imported",
		})
	}
	issueAuth(draft, doc.Authentication, "authentication")

	unsupported := newTally()
	unknown := newTally()

	var walk func(children []v5Item, path string) []importer.Item
	walk = func(children []v5Item, path string) []importer.Item {
		items := make([]importer.Item, 0, len(children))
		for i := range children {
			c := &children[i]
			p := fmt.Sprintf("%s[%d]", path, i)
			switch {
			case c.Children != nil:
				node := importer.Item{Folder: &importer.Folder{
					Name: c.Name.String(),
					Description: importer.FirstNonEmpty(
						importer.Description(c.Meta.Description),
						importer.Description(c.Description),
					),
				}}
				node.Children = walk(*c.Children, p)
				items = append(items, node)
			case c.Method != "":
				items = append(items, importer.Item{Request: mapV5Request(c, p, draft)})
			case c.ProtoFileID != "":
				unsupported.add("grpc_request")
			case c.URL != "":
				unsupported.add("websocket_request")
			default:
				unknown.add("item")
			}
		}
		return items
	}

	col := importer.Collection{
		Name:        doc.Name.String(),
		Description: draft.Description,
		Items:       walk(doc.Collection, "collection"),
	}
	draft.Collections = append(draft.Collections, col)

	unsupported.emit(draft, importer.SeverityUnsupported, "import support is pending")
	unknown.emit(draft, importer.SeverityWarning, "")
	return draft, nil
}

func mapV5Request(c *v5Item, path string, draft *importer.Draft) *importer.Request {
	body, bodyType := convertBody(c.Body, path, draft)
	out := &importer.Request{
		Kind: "http",
		Name: c.Name.String(),
		Description: importer.FirstNonEmpty(
			importer.Description(c.Meta.Description),
			importer.Description(c.Description),
		),
		Method:   c.Method.String(),
		URL:      c.URL.String(),
		Headers:  convertHeaders(c.Headers),
		Params:   convertParams(c.Parameters),
		Body:     body,
		BodyType: bodyType,
	}
	issueAuth(draft, c.Authentication, path+".authentication")
	return out
}
