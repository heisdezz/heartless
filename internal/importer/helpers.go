package importer

import (
	"encoding/json"
	"errors"
	"strings"
)

// Text is a JSON string field that also accepts other scalar values
// (numbers, booleans), which show up in hand-edited or tool-generated
// exports. Objects and arrays are rejected so structural mistakes surface
// as parse errors instead of being flattened into garbage.
type Text string

// UnmarshalJSON decodes a JSON string, number, boolean, or null into Text.
func (t *Text) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*t = ""
		return nil
	}
	switch data[0] {
	case '"':
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*t = Text(s)
		return nil
	case '{', '[':
		return errors.New("importer: cannot decode composite value as text")
	default:
		// Number or boolean literal: keep it verbatim.
		*t = Text(data)
		return nil
	}
}

// String returns the text value.
func (t Text) String() string { return string(t) }

// FirstNonEmpty returns the first non-empty value.
func FirstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// Present reports whether a raw JSON value is present and not null.
func Present(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	s := strings.TrimSpace(string(raw))
	return s != "" && s != "null"
}

// Description extracts text from description fields, which tools write as
// either a plain string or an object with a "content" member.
func Description(raw json.RawMessage) string {
	if !Present(raw) {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var obj struct {
		Content Text `json:"content"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.Content.String()
	}
	return ""
}
