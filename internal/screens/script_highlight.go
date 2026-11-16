package screens

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// jsKeywords are highlighted in a post-response script.
var jsKeywords = map[string]bool{
	"const": true, "let": true, "var": true, "function": true, "return": true,
	"if": true, "else": true, "for": true, "while": true, "of": true, "in": true,
	"true": true, "false": true, "null": true, "undefined": true, "new": true,
	"try": true, "catch": true, "finally": true, "throw": true, "typeof": true,
	"delete": true, "break": true, "continue": true, "this": true, "switch": true,
	"case": true, "default": true, "async": true, "await": true,
}

// scriptGlobals are the objects the runtime provides; highlighting them makes
// the available API visible while editing.
var scriptGlobals = map[string]bool{
	"req": true, "res": true, "env": true, "console": true, "test": true, "JSON": true,
}

// styleRange is a run of a script's text in one color, in rune offsets so it
// can style the editor's own text.
type styleRange struct {
	Start, End int
	Color      ui.Color
	Bold       bool
}

// scriptStyleRanges tokenizes JavaScript for the editor: keywords, strings,
// numbers, comments, the runtime's globals and {{variables}}. A placeholder
// the active environment defines is primary-colored; an undefined one is a
// warning.
func scriptStyleRanges(src string, tk *theme.Tokens, known func(string) bool) []styleRange {
	if strings.TrimSpace(src) == "" {
		return nil
	}
	// Variables read as keywords: a defined one is what the runtime will
	// substitute, an undefined one is a warning.
	variableColor := tk.Primary
	if variableColor == tk.Text {
		// In themes whose primary matches body text, the info color keeps a
		// placeholder visible.
		variableColor = tk.Info
	}

	var out []styleRange
	// add colors src[from:to], converting its byte offsets to rune offsets.
	add := func(from, to int, color ui.Color, bold bool) {
		if to <= from || from < 0 || to > len(src) {
			return
		}
		start := utf8.RuneCountInString(src[:from])
		out = append(out, styleRange{
			Start: start,
			End:   start + utf8.RuneCountInString(src[from:to]),
			Color: color,
			Bold:  bold,
		})
	}
	variable := func(from, to int) {
		name := strings.TrimSpace(src[from+2 : to-2])
		if known != nil && known(name) {
			add(from, to, variableColor, true)
			return
		}
		add(from, to, tk.Warning, true)
	}

	i := 0
	for i < len(src) {
		ch := src[i]
		switch {
		case ch == '/' && i+1 < len(src) && src[i+1] == '/':
			end := strings.IndexByte(src[i:], '\n')
			if end < 0 {
				end = len(src) - i
			}
			add(i, i+end, tk.TextTertiary, false)
			i += end
		case ch == '/' && i+1 < len(src) && src[i+1] == '*':
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				end = len(src) - i
			} else {
				end += 4
			}
			add(i, i+end, tk.TextTertiary, false)
			i += end
		case ch == '"' || ch == '\'' || ch == '`':
			end, ok := scanJSString(src, i)
			if !ok {
				add(i, len(src), tk.Success, false)
				i = len(src)
				continue
			}
			literal := src[i:end]
			add(i, end, tk.Success, false)
			// A placeholder inside a string is colored over it, so a scripted
			// URL reads as it does in the URL field.
			for off := 0; ; {
				start := strings.Index(literal[off:], "{{")
				if start < 0 {
					break
				}
				from := i + off + start
				stop := strings.Index(src[from:], "}}")
				if stop < 0 || from+stop+2 > end {
					break
				}
				variable(from, from+stop+2)
				off += start + stop + 2
			}
			i = end
		case ch == '{' && strings.HasPrefix(src[i:], "{{"):
			end := strings.Index(src[i:], "}}")
			if end < 0 {
				add(i, len(src), tk.Warning, true)
				i = len(src)
				continue
			}
			end += i + 2
			variable(i, end)
			i = end
		case isDigit(ch):
			start := i
			for i < len(src) && (isDigit(src[i]) || src[i] == '.') {
				i++
			}
			add(start, i, tk.Warning, false)
		case isIdentStart(ch):
			start := i
			for i < len(src) && isIdentPart(src[i]) {
				i++
			}
			word := src[start:i]
			switch {
			case jsKeywords[word]:
				add(start, i, tk.Primary, true)
			case scriptGlobals[word]:
				add(start, i, tk.Info, true)
			}
		default:
			i++
		}
	}
	return out
}

// textRanges turns style ranges into the editor's text ranges.
func textRanges(ranges []styleRange) []ui.TextRange {
	out := make([]ui.TextRange, 0, len(ranges))
	for _, r := range ranges {
		tr := ui.TextRange{Start: r.Start, End: r.End, Color: r.Color}
		if r.Bold {
			tr.Weight = 600
		}
		out = append(out, tr)
	}
	return out
}

// scriptHighlightSpans renders a script as standalone spans, for showing it
// outside an editor.
func scriptHighlightSpans(src string, tk *theme.Tokens, known func(string) bool) []ui.Span {
	ranges := scriptStyleRanges(src, tk, known)
	if len(ranges) == 0 {
		return nil
	}
	runes := []rune(src)
	spans := make([]ui.Span, 0, len(ranges)+1)
	at := 0
	for _, r := range ranges {
		if r.Start < at || r.End > len(runes) {
			continue
		}
		if r.Start > at {
			spans = append(spans, ui.Span{Text: string(runes[at:r.Start]), Color: tk.Text})
		}
		s := ui.Span{Text: string(runes[r.Start:r.End]), Color: r.Color}
		if r.Bold {
			s.Weight = 600
		}
		spans = append(spans, s)
		at = r.End
	}
	if at < len(runes) {
		spans = append(spans, ui.Span{Text: string(runes[at:]), Color: tk.Text})
	}
	return spans
}

func isDigit(ch byte) bool { return ch >= '0' && ch <= '9' }

func isIdentStart(ch byte) bool {
	return ch == '_' || ch == '$' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isIdentPart(ch byte) bool { return isIdentStart(ch) || isDigit(ch) }
