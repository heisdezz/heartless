package screens

import (
	"sort"
	"strings"
)

// scriptCompletionMax is how many suggestions the editor shows at once.
const scriptCompletionMax = 8

// Top-level tokens offered by the script editor's inline completion.
var scriptTopCompletions = []string{
	"res", "req", "env", "console", "test",
	"res.body", "res.text", "res.status", "res.json()", "res.headers", "res.contentType",
	"env.get(", "env.set(", "env.all()", "env.has(",
	"console.log(", `test("name", () => {})`,
}

// Members offered after an object and a dot.
var scriptMemberCompletions = map[string][]string{
	"res":     {"body", "text", "status", "statusText", "headers", "json()", "size", "contentType", "url", "id", "error"},
	"req":     {"id", "method", "url", "body", "bodyType", "params", "headers", "authType", "workspaceId"},
	"env":     {"get(", "set(", "all()", "has("},
	"console": {"log(", "info(", "warn(", "error(", "debug("},
}

// isScriptWordRune reports whether r continues a word for completion. Dots are
// included so `res.bo` completes as one token.
func isScriptWordRune(r rune) bool {
	return r == '_' || r == '.' ||
		r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}

// scriptWordPrefix returns the word ending at caret (in runes) and its start.
func scriptWordPrefix(text string, caret int) (string, int) {
	runes := []rune(text)
	if caret < 0 {
		caret = 0
	}
	if caret > len(runes) {
		caret = len(runes)
	}
	start := caret
	for start > 0 && isScriptWordRune(runes[start-1]) {
		start--
	}
	return string(runes[start:caret]), start
}

// scriptCompletions lists up to scriptCompletionMax completions for prefix,
// drawn from the runtime's objects and the active environment's variables.
func scriptCompletions(prefix string, env ScriptEnvironment) []string {
	lower := strings.ToLower(prefix)
	seen := map[string]bool{}
	out := make([]string, 0, scriptCompletionMax)
	add := func(s string) {
		if s == prefix || seen[s] {
			return
		}
		if !strings.HasPrefix(strings.ToLower(s), lower) {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	if dot := strings.LastIndex(prefix, "."); dot >= 0 {
		object := prefix[:dot]
		for _, member := range scriptMemberCompletions[object] {
			add(object + "." + member)
		}
	} else {
		for _, token := range scriptTopCompletions {
			add(token)
		}
		for _, v := range env.Environment {
			add(v.Name)
		}
		for _, v := range env.Workspace {
			add(v.Name)
		}
	}
	sort.Strings(out)
	if len(out) > scriptCompletionMax {
		out = out[:scriptCompletionMax]
	}
	return out
}

// scriptInsertCompletion replaces the word before caret with completion and
// returns the new text and caret, both as rune offsets into the text.
func scriptInsertCompletion(text string, caret int, completion string) (string, int) {
	runes := []rune(text)
	if caret < 0 {
		caret = 0
	}
	if caret > len(runes) {
		caret = len(runes)
	}
	_, start := scriptWordPrefix(text, caret)
	insert := []rune(completion)
	out := make([]rune, 0, len(runes)+len(insert))
	out = append(out, runes[:start]...)
	out = append(out, insert...)
	out = append(out, runes[caret:]...)
	return string(out), start + len(insert)
}
