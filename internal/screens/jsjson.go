package screens

import (
	"encoding/json"
	"strings"
)

// jsObjectToJSON converts a JavaScript object literal, as people type in a JSON
// body, into strict JSON: unquoted keys get quoted and trailing commas are
// dropped. Input that is already valid JSON is returned unchanged, and anything
// that cannot be converted is returned as it was so the server sees what the
// user typed.
func jsObjectToJSON(s string) string {
	if strings.TrimSpace(s) == "" {
		return s
	}
	if json.Valid([]byte(s)) {
		return s
	}
	out, ok := rewriteJSObject(s)
	if !ok {
		return s
	}
	if !json.Valid([]byte(out)) {
		return s
	}
	return out
}

// rewriteJSObject quotes bare object keys and strips trailing commas. It returns
// false when the text has a construct it does not handle.
func rewriteJSObject(s string) (string, bool) {
	var b strings.Builder
	b.Grow(len(s) + 16)

	i := 0
	// inObject tracks whether the next `,` starts a key (inside an object) or a
	// value (inside an array).
	inObject := false
	var stack []bool
	expectKey := false
	for i < len(s) {
		ch := s[i]
		switch {
		case ch == '"' || ch == '\'':
			j, ok := scanJSString(s, i)
			if !ok {
				return "", false
			}
			literal := s[i:j]
			if ch == '\'' {
				literal = `"` + strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(literal, "'"), "'"), `"`, `\"`) + `"`
			}
			b.WriteString(literal)
			i = j
			expectKey = false
		case ch == '{' || ch == '[':
			stack = append(stack, inObject)
			b.WriteByte(ch)
			i++
			inObject = ch == '{'
			expectKey = ch == '{'
		case ch == '}' || ch == ']':
			trimTrailingComma(&b)
			b.WriteByte(ch)
			i++
			if len(stack) > 0 {
				inObject = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			}
			expectKey = false
		case ch == ',':
			trimTrailingComma(&b)
			b.WriteByte(ch)
			i++
			// A key may follow a comma inside an object, a value inside an array.
			expectKey = inObject
		case ch == ':':
			b.WriteByte(ch)
			i++
			expectKey = false
		case isJSSpace(ch):
			b.WriteByte(ch)
			i++
		default:
			start := i
			for i < len(s) && !isJSDelimiter(s[i]) {
				i++
			}
			tok := s[start:i]
			if expectKey {
				b.WriteByte('"')
				b.WriteString(strings.ReplaceAll(tok, `"`, `\"`))
				b.WriteByte('"')
			} else {
				b.WriteString(tok)
			}
			expectKey = false
		}
	}
	return b.String(), true
}

// trimTrailingComma removes a comma just before a closing bracket, skipping
// whitespace.
func trimTrailingComma(b *strings.Builder) {
	s := b.String()
	end := len(s)
	for end > 0 && isJSSpace(s[end-1]) {
		end--
	}
	if end > 0 && s[end-1] == ',' {
		trimmed := s[:end-1] + s[end:]
		b.Reset()
		b.WriteString(trimmed)
	}
}

func isJSSpace(ch byte) bool {
	return ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t'
}

func isJSDelimiter(ch byte) bool {
	switch ch {
	case '{', '}', '[', ']', ',', ':', '"', '\'':
		return true
	}
	return isJSSpace(ch)
}

// scanJSString returns the index just past the string starting at i.
func scanJSString(s string, i int) (int, bool) {
	quote := s[i]
	j := i + 1
	for j < len(s) {
		switch s[j] {
		case '\\':
			j += 2
			continue
		case quote:
			return j + 1, true
		}
		j++
	}
	return 0, false
}
