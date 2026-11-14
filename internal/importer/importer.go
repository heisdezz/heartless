// Package importer defines the format-neutral import model and the contract
// that source-format adapters implement.
//
// The pipeline is: Detect -> Parse (into a Draft) -> validate/normalize ->
// conflict resolution -> transactional persistence (see
// heartless/internal/services/importer). Adapters never write to storage;
// they only map external documents onto the normalized Draft.
package importer

import "errors"

// Format identifies a supported external source format.
type Format string

const (
	FormatPostman  Format = "postman"
	FormatInsomnia Format = "insomnia"
)

// Issue severities reported by the pipeline.
const (
	// SeverityUnsupported marks source features with no Heartless equivalent yet.
	SeverityUnsupported = "unsupported"
	// SeverityWarning marks data that was adjusted, merged, or dropped to proceed.
	SeverityWarning = "warning"
	// SeverityError marks source problems that prevented mapping part of the document.
	SeverityError = "error"
)

// ErrUnsupportedFormat is returned when no registered importer recognizes the source.
var ErrUnsupportedFormat = errors.New("unsupported import format")

// Issue describes an unsupported feature, conflict, or mapping problem found
// in a source document. Path locates the problem inside the source (for
// example "item[2].request.auth").
type Issue struct {
	Severity string `json:"severity"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

// Draft is the normalized result of parsing a source document, before
// validation, conflict resolution, and persistence.
type Draft struct {
	Format      Format       `json:"format"`
	Name        string       `json:"name,omitempty"`
	Description string       `json:"description,omitempty"`
	Collections []Collection `json:"collections"`
	Issues      []Issue      `json:"issues,omitempty"`
}

// Collection is a group of folders and requests to create inside the target
// workspace. Names are as they appear in the source; the service layer
// normalizes empty names and resolves conflicts.
type Collection struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Items       []Item `json:"items,omitempty"`
}

// Item is a node in an imported tree. Exactly one of Folder or Request is
// set; Children is only populated for folder nodes and is ordered as in the
// source document.
type Item struct {
	Folder   *Folder  `json:"folder,omitempty"`
	Request  *Request `json:"request,omitempty"`
	Children []Item   `json:"children,omitempty"`
}

// Folder is an imported folder node.
type Folder struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Request is an imported saved-request definition, independent of execution.
type Request struct {
	// Kind is a request.Kind value ("http", "graphql", "grpc", "websocket").
	// Empty is normalized to "http" by the service layer.
	Kind        string      `json:"kind,omitempty"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Method      string      `json:"method,omitempty"`
	URL         string      `json:"url,omitempty"`
	Headers     []NameValue `json:"headers,omitempty"`
	Params      []NameValue `json:"params,omitempty"`
	Body        string      `json:"body,omitempty"`
	BodyType    string      `json:"body_type,omitempty"`
}

// NameValue is a header or URL parameter entry. Enabled=false entries are
// kept by parsers so the service layer can report and drop them deliberately.
type NameValue struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

// Importer detects and parses one external source format.
type Importer interface {
	// Format returns the identifier of the format this importer handles.
	Format() Format
	// Detect cheaply reports whether data looks like this importer's format.
	// It must not return true for documents it cannot parse.
	Detect(data []byte) bool
	// Parse maps a source document onto a normalized Draft. It returns an
	// error only when the document is recognized but malformed; unrecognized
	// documents should fail Detect instead.
	Parse(data []byte) (*Draft, error)
}

// Select returns the first importer that recognizes data, or
// ErrUnsupportedFormat when none does.
func Select(list []Importer, data []byte) (Importer, error) {
	for _, im := range list {
		if im != nil && im.Detect(data) {
			return im, nil
		}
	}
	return nil, ErrUnsupportedFormat
}
