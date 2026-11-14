package screens

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/app"
	domainreq "heartless/internal/domain/request"
)

var httpMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

// requestPane renders the selected request's editor and response panes. It binds
// widgets to the request's own persistent state so typing is kept, and renders
// the response from a snapshot so background sends stay isolated.
func requestPane(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services, req domainreq.Request) {
	rs := st.requestStateFor(req)
	resp, running, errText, respTab, sentBody := st.responseView(req.ID)

	ui.Column(c).Grow(1).Basis(0).AlignSelf(ui.Stretch).MinWidth(0).Children(func() {
		// Request on the left, response on the right, split vertically.
		ui.Row(c).Grow(1).Basis(0).MinWidth(0).AlignItems(ui.Stretch).Children(func() {
			ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
				requestEditor(c, tk, st, services, req, rs)
			})
			ui.Box(c).Width(1).AlignSelf(ui.Stretch).Background(tk.BorderSecondary)
			ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
				responsePanel(c, tk, rs, resp, running, errText, respTab, sentBody)
			})
		})
	})
}

// responseView snapshots the response fields under the lock.
func (s *shellState) responseView(reqID string) (resp *responseSnapshot, running bool, errText string, respTab int, sentBody string) {
	mu.Lock()
	defer mu.Unlock()
	rs, ok := s.requestStates[reqID]
	if !ok {
		return nil, false, "", 0, ""
	}
	running, errText, respTab, sentBody = rs.running, rs.error, rs.ResponseTab, rs.sentBody
	if rs.response != nil && rs.response.Response != nil {
		res := rs.response.Response
		var timeline []string
		if !rs.response.StartedAt.IsZero() {
			timeline = append(timeline, "Started: "+rs.response.StartedAt.Format("15:04:05.000"))
		}
		if !rs.response.EndedAt.IsZero() {
			timeline = append(timeline, "Ended: "+rs.response.EndedAt.Format("15:04:05.000"))
		}
		timeline = append(timeline, "Duration: "+humanDuration(rs.response.Duration))
		timeline = append(timeline, "Size: "+humanBytes(res.Size))
		if res.EffectiveURL != "" {
			timeline = append(timeline, "Effective URL: "+res.EffectiveURL)
		}
		if res.ContentType != "" {
			timeline = append(timeline, "Content-Type: "+res.ContentType)
		}
		resp = &responseSnapshot{
			Status:      statusLine(res),
			Code:        res.StatusCode,
			Duration:    humanDuration(rs.response.Duration),
			Size:        humanBytes(res.Size),
			Body:        string(res.Body),
			Headers:     cloneHeaders(res.Headers),
			Cookies:     cookiesFromHeaders(res.Headers),
			Timeline:    timeline,
			ContentType: res.ContentType,
		}
	}
	return resp, running, errText, respTab, sentBody
}

// responseSnapshot is an immutable view of a response for a frame.
type responseSnapshot struct {
	Status      string
	Code        int
	Duration    string
	Size        string
	Body        string
	Headers     map[string]string
	Cookies     []string
	Timeline    []string
	ContentType string
}

// requestEditor is the top pane: method, URL, send and the request tabs.
func requestEditor(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services, req domainreq.Request, rs *requestState) {
	_, running, _, _, _ := st.responseView(req.ID)

	ui.Column(c).Grow(1).Basis(0).AlignSelf(ui.Stretch).MinWidth(0).Children(func() {
		ui.Row(c).Height(52).AlignSelf(ui.Stretch).PaddingX(16).Gap(8).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
			methodSelect(c, rs)
			// field.Input returns the inner text field, not its styled row, so wrap
			// it in a clipping flex box: the placeholder's intrinsic width would
			// otherwise stop the row from flexing.
			ui.Box(c).Grow(1).Basis(0).MinWidth(0).Clip().Children(func() {
				field.Input(c, &rs.URL, field.InputProps{Placeholder: "https://api.example.com/users  ·  use {{baseUrl}}"}).
					Grow(1).Basis(0).Font("monospace")
			})
			if running {
				if action.Button(c, "Cancel", action.ButtonProps{Variant: action.Dashed}).Clicked() {
					st.cancelSend(req.ID)
				}
			} else if action.Button(c, "Send", action.ButtonProps{Variant: action.Primary}).Clicked() {
				st.persistDraft(services, req.ID, rs)
				st.send(services, req, c.Services().Invalidate)
			}
		})

		// What the URL resolves to with the active environment, and which of its
		// placeholders the environment defines.
		if strings.Contains(rs.URL, "{{") {
			ui.Column(c).AlignSelf(ui.Stretch).MinWidth(0).Padding(4, 16, 8, 16).Gap(2).Children(func() {
				ui.Box(c).AlignSelf(ui.Stretch).MinWidth(0).Clip().Children(func() {
					variableHighlightText(c, tk, rs.URL, 11, st.knowsVariable)
				})
				ui.Text(c, "→ "+st.previewURL(rs.URL)).Font("monospace").FontSize(11).
					TextColor(tk.TextTertiary).SingleLine().MinWidth(0)
			})
		}

		// Tabs, as plain buttons so the pane keeps filling the width.
		ui.Row(c).Height(38).AlignSelf(ui.Stretch).PaddingX(16).Gap(4).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
			for i, name := range []string{"Params", "Headers", "Body", "Auth"} {
				i, name := i, name
				btn := action.Button(c, name, action.ButtonProps{Variant: action.Text})
				if rs.Tab == i {
					btn.FontWeight(600).TextColor(tk.Text).Underline()
				} else {
					btn.TextColor(tk.TextSecondary)
				}
				if btn.Clicked() {
					rs.Tab = i
				}
			}
		})

		ui.Scroll(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
			ui.Column(c).AlignSelf(ui.Stretch).Grow(1).MinWidth(0).Padding(12, 16).Children(func() {
				switch rs.Tab {
				case 0:
					pairsEditor(c, tk, &rs.Params, "Query parameter", "Value")
				case 1:
					pairsEditor(c, tk, &rs.Headers, "Header", "Value")
				case 2:
					bodyEditor(c, tk, rs)
				case 3:
					authEditor(c, tk, rs)
				}
			})
		})
	})
}

// bodyTypes lists the body encodings in the order the menu shows them.
var bodyTypes = []struct {
	value string
	label string
}{
	{string(domainreq.BodyForm), "Url Encoded"},
	{string(domainreq.BodyMultipart), "Multi-Part"},
	{string(domainreq.BodyGraphQL), "GraphQL"},
	{string(domainreq.BodyJSON), "JSON"},
	{string(domainreq.BodyText), "Text"},
	{string(domainreq.BodyBinary), "Binary File"},
	{string(domainreq.BodyNone), "No Body"},
}

// bodyEditor shows the body type picker and the editor for that type.
func bodyEditor(c *ui.Context, tk *theme.Tokens, rs *requestState) {
	if rs.BodyType == "" {
		rs.BodyType = string(domainreq.BodyNone)
	}

	ui.Column(c).AlignSelf(ui.Stretch).MinWidth(0).Gap(8).Children(func() {
		options := make([]field.SelectOption, 0, len(bodyTypes))
		for _, bt := range bodyTypes {
			options = append(options, field.SelectOption{Value: bt.value, Label: bt.label})
		}
		ui.Row(c).AlignSelf(ui.Stretch).Children(func() {
			field.Select(c, &rs.BodyType, field.SelectProps{Options: options, Size: theme.Small}).Width(150).Shrink(0)
		})

		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(8).Children(func() {
			switch rs.BodyType {
			case string(domainreq.BodyNone):
				ui.Text(c, "This request has no body.").FontSize(13).TextColor(tk.TextTertiary)
			case string(domainreq.BodyForm), string(domainreq.BodyMultipart):
				pairsEditor(c, tk, &rs.Form, "Field", "Value")
			case string(domainreq.BodyJSON):
				jsonBodyEditor(c, tk, rs)
			default:
				ui.Box(c).AlignSelf(ui.Stretch).MinWidth(0).Children(func() {
					field.TextArea(c, &rs.Body, field.TextAreaProps{Placeholder: bodyPlaceholder(rs.BodyType), MinRows: 8, MaxRows: 20})
				})
			}
		})
	})
}

func bodyPlaceholder(bodyType string) string {
	switch bodyType {
	case string(domainreq.BodyGraphQL):
		return "query { user(id: 1) { id name } }"
	case string(domainreq.BodyBinary):
		return "Binary payload"
	default:
		return "Request body"
	}
}

// jsonBodyEditor edits a JSON body as text and shows a syntax-highlighted
// preview, which also shows a JavaScript object converted to strict JSON.
func jsonBodyEditor(c *ui.Context, tk *theme.Tokens, rs *requestState) {
	ui.Box(c).AlignSelf(ui.Stretch).MinWidth(0).Shrink(0).Children(func() {
		field.TextArea(c, &rs.Body, field.TextAreaProps{
			Placeholder: "{\n  \"name\": \"Ada\"\n}",
			MinRows:     8,
			MaxRows:     12,
		}).Font("monospace")
	})

	body := strings.TrimSpace(rs.Body)
	if body == "" {
		return
	}
	converted := jsObjectToJSON(rs.Body)
	header := "Preview"
	if !json.Valid([]byte(body)) {
		header = "Preview · sent as JSON"
	}
	ui.Text(c, header).FontSize(11).TextColor(tk.TextTertiary).Shrink(0)
	ui.Scroll(c).Height(160).Shrink(0).MinWidth(0).Children(func() {
		ui.Box(c).AlignSelf(ui.Stretch).MinWidth(0).Padding(8).Radius(tk.BorderRadiusSM).
			Background(tk.FillQuaternary).Children(func() {
			if spans := jsonHighlightSpans(converted, tk); len(spans) > 0 {
				ui.RichText(c, spans...).Font("monospace").FontSize(12).MinWidth(0)
				return
			}
			ui.Text(c, converted).Font("monospace").FontSize(12).MinWidth(0)
		})
	})
}

// authEditor is the Auth tab: the type menu (with Settings and Info beside it)
// and the credential fields for the chosen type.
func authEditor(c *ui.Context, tk *theme.Tokens, rs *requestState) {
	ui.Column(c).AlignSelf(ui.Stretch).MinWidth(0).Gap(16).Children(func() {
		authToolbar(c, tk, rs)
		authFields(c, tk, rs)
	})
}

// methodSelect renders the HTTP method picker.
func methodSelect(c *ui.Context, rs *requestState) {
	if rs.Method == "" {
		rs.Method = "GET"
	}
	options := make([]field.SelectOption, 0, len(httpMethods))
	for _, m := range httpMethods {
		options = append(options, field.SelectOption{Value: m, Label: m})
	}
	field.Select(c, &rs.Method, field.SelectProps{Options: options, Size: theme.Small}).Width(104).Shrink(0)
}

// pairsEditor shows editable key/value rows stored in *rows, with add/remove.
// Widgets bind to the persistent rows, so typing is kept between frames.
func pairsEditor(c *ui.Context, tk *theme.Tokens, rows *[]*kv, keyHint, valueHint string) {
	if len(*rows) == 0 {
		*rows = append(*rows, &kv{On: true})
	}
	ui.Column(c).MinWidth(0).AlignSelf(ui.Stretch).Gap(0).
		Border(1, tk.BorderSecondary).Radius(tk.BorderRadiusSM).Children(func() {
		ui.Row(c).Height(28).AlignSelf(ui.Stretch).PaddingX(8).Gap(6).AlignItems(ui.Center).
			Background(tk.FillQuaternary).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).
			Children(func() {
				ui.Box(c).Width(20).Shrink(0)
				ui.Text(c, "Key").FontSize(12).TextColor(tk.TextSecondary).Grow(1).Basis(0).MinWidth(0)
				ui.Text(c, "Value").FontSize(12).TextColor(tk.TextSecondary).Grow(1).Basis(0).MinWidth(0)
				ui.Box(c).Width(24).Shrink(0)
			})

		remove := -1
		for i, row := range *rows {
			i, row := i, row
			ui.Row(c).Height(36).AlignSelf(ui.Stretch).PaddingX(8).Gap(6).AlignItems(ui.Center).
				BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).
				Children(func() {
					choice.Checkbox(c, &row.On, "", choice.CheckboxProps{})
					ui.Box(c).Grow(1).Basis(0).MinWidth(0).Clip().Children(func() {
						field.Input(c, &row.Key, field.InputProps{Placeholder: keyHint, Size: theme.Small}).Grow(1).Basis(0)
					})
					ui.Box(c).Grow(1).Basis(0).MinWidth(0).Clip().Children(func() {
						valueInput := field.Input(c, &row.Value, field.InputProps{Placeholder: valueHint, Size: theme.Small}).Grow(1).Basis(0)
						if strings.Contains(row.Value, "{{") {
							valueInput.TextColor(tk.Primary).Font("monospace")
						}
					})
					if action.Button(c, "×", action.ButtonProps{Variant: action.Text, Size: theme.Small}).Width(24).Shrink(0).Clicked() {
						remove = i
					}
				})
		}
		if remove >= 0 {
			*rows = append((*rows)[:remove], (*rows)[remove+1:]...)
			if len(*rows) == 0 {
				*rows = append(*rows, &kv{On: true})
			}
		}
		ui.Row(c).Height(32).AlignSelf(ui.Stretch).PaddingX(8).AlignItems(ui.Center).Children(func() {
			if action.Button(c, "+ Add", action.ButtonProps{Variant: action.Text, Size: theme.Small}).Shrink(0).Clicked() {
				*rows = append(*rows, &kv{On: true})
			}
		})
	})
}

// responsePanel is the bottom pane: status, timing and tabbed response details.
func responsePanel(c *ui.Context, tk *theme.Tokens, rs *requestState, resp *responseSnapshot, running bool, errText string, respTab int, sentBody string) {
	ui.Column(c).Fill().MinWidth(0).Children(func() {
		ui.Row(c).Height(36).PaddingX(16).Gap(12).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
			ui.Text(c, "Response").FontSize(13).Bold().Shrink(0)
			switch {
			case running:
				ui.Text(c, "Sending…").FontSize(12).TextColor(tk.TextTertiary)
			case resp != nil:
				ui.Text(c, resp.Status).FontSize(12).FontWeight(600).TextColor(statusColor(tk, resp.Code)).Shrink(0)
				ui.Text(c, resp.Duration).FontSize(12).TextColor(tk.TextTertiary).Shrink(0)
				ui.Text(c, resp.Size).FontSize(12).TextColor(tk.TextTertiary).Shrink(0)
			case errText != "":
				ui.Text(c, errText).FontSize(12).TextColor(tk.Error).SingleLine().MinWidth(0)
			}
		})

		tabs := []string{"Body", "Headers", "Cookies", "Timeline"}
		if strings.TrimSpace(sentBody) != "" {
			tabs = append(tabs, "Request body")
		}
		ui.Row(c).Height(34).PaddingX(16).Gap(4).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
			for i, name := range tabs {
				i, name := i, name
				btn := action.Button(c, name, action.ButtonProps{Variant: action.Text})
				if respTab == i {
					btn.FontWeight(600).TextColor(tk.Text).Underline()
				} else {
					btn.TextColor(tk.TextSecondary)
				}
				if btn.Clicked() {
					rs.ResponseTab = i
				}
			}
		})

		ui.Scroll(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Padding(12, 16).Gap(6).Children(func() {
				switch {
				case running:
					ui.Text(c, "Waiting for response…").TextColor(tk.TextSecondary)
				case resp == nil && errText != "":
					ui.Text(c, errText).TextColor(tk.Error)
				case resp == nil:
					ui.Text(c, "Send a request to see the response.").TextColor(tk.TextSecondary)
				default:
					switch {
					case respTab == 0:
						responseBodyView(c, tk, rs, resp.Body, resp.ContentType)
					case respTab == 1:
						if len(resp.Headers) == 0 {
							ui.Text(c, "No headers").TextColor(tk.TextSecondary)
						} else {
							for _, line := range headerLines(resp.Headers) {
								ui.Text(c, line).Font("monospace").FontSize(12).TextColor(tk.Text)
							}
						}
					case respTab == 2:
						if len(resp.Cookies) == 0 {
							ui.Text(c, "No cookies").TextColor(tk.TextSecondary)
						} else {
							for _, ck := range resp.Cookies {
								ui.Text(c, ck).Font("monospace").FontSize(12).TextColor(tk.Text)
							}
						}
					case respTab == 3:
						if len(resp.Timeline) == 0 {
							ui.Text(c, "No timeline data").TextColor(tk.TextSecondary)
						} else {
							for _, line := range resp.Timeline {
								ui.Text(c, line).Font("monospace").FontSize(12).TextColor(tk.Text)
							}
						}
					case respTab == 4:
						responseTextView(c, tk, rs, sentBody, "(no request body sent)", false)
					default:
						responseBodyView(c, tk, rs, resp.Body, resp.ContentType)
					}
				}
			})
		})
	})
}

// responseBodyView renders response body, auto-formatting and highlighting JSON.
func responseBodyView(c *ui.Context, tk *theme.Tokens, rs *requestState, body, contentType string) {
	jsonLike := isJSONContentType(contentType) || looksLikeJSON(body)
	formatted := body
	if jsonLike {
		if f, ok := formatJSONBody(body); ok {
			formatted = f
		}
	}
	responseTextView(c, tk, rs, formatted, "(empty body)", jsonLike && looksLikeJSON(formatted))
}

// responseTextView renders body text with optional JSON highlighting and copy action.
func responseTextView(c *ui.Context, tk *theme.Tokens, rs *requestState, body, emptyText string, highlightJSON bool) {
	trimmed := strings.TrimSpace(body)
	display := body
	if trimmed == "" {
		display = emptyText
	}
	ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Start).Gap(8).Children(func() {
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
			if highlightJSON && trimmed != "" {
				spans := jsonHighlightSpans(display, tk)
				if len(spans) > 0 {
					ui.RichText(c, spans...).Font("monospace").FontSize(12).Selectable()
					return
				}
			}
			ui.Text(c, display).Font("monospace").FontSize(12).TextColor(tk.Text).Selectable()
		})
		btn := action.Button(c, "Copy", action.ButtonProps{Variant: action.Text, Size: theme.Small}).Shrink(0)
		if btn.Clicked() && trimmed != "" {
			c.Services().WriteClipboard(body)
			rs.copyNotice = "Copied"
		}
	})
	if rs.copyNotice != "" {
		ui.Text(c, rs.copyNotice).FontSize(11).TextColor(tk.TextTertiary)
	}
}

// jsonHighlightSpans tokenizes JSON and returns colored spans.
func jsonHighlightSpans(body string, tk *theme.Tokens) []ui.Span {
	if strings.TrimSpace(body) == "" {
		return nil
	}
	keyColor := tk.Info
	stringColor := tk.Success
	numberColor := tk.Warning
	boolNullColor := tk.Primary
	punctColor := tk.TextTertiary
	textColor := tk.Text

	spans := make([]ui.Span, 0, 64)
	var seg strings.Builder
	segColor := textColor
	flush := func() {
		if seg.Len() == 0 {
			return
		}
		spans = append(spans, ui.Span{Text: seg.String(), Color: segColor})
		seg.Reset()
	}
	appendSeg := func(s string, color ui.Color) {
		if s == "" {
			return
		}
		if seg.Len() > 0 && color != segColor {
			flush()
		}
		segColor = color
		seg.WriteString(s)
	}

	for i := 0; i < len(body); {
		b := body[i]
		switch b {
		case '{', '}', '[', ']', ',', ':':
			appendSeg(string(b), punctColor)
			i++
		case '"':
			j := i + 1
			esc := false
			for j < len(body) {
				cj := body[j]
				if esc {
					esc = false
				} else if cj == '\\' {
					esc = true
				} else if cj == '"' {
					j++
					break
				}
				j++
			}
			token := body[i:j]
			color := stringColor
			if isJSONKeyToken(body, j) {
				color = keyColor
			}
			appendSeg(token, color)
			i = j
		case ' ', '\n', '\r', '\t':
			j := i + 1
			for j < len(body) {
				switch body[j] {
				case ' ', '\n', '\r', '\t':
					j++
				default:
					goto wsDone
				}
			}
		wsDone:
			appendSeg(body[i:j], textColor)
			i = j
		default:
			if b == '-' || (b >= '0' && b <= '9') {
				j := i + 1
				for j < len(body) {
					cj := body[j]
					if (cj >= '0' && cj <= '9') || cj == '.' || cj == 'e' || cj == 'E' || cj == '+' || cj == '-' {
						j++
					} else {
						break
					}
				}
				appendSeg(body[i:j], numberColor)
				i = j
				continue
			}
			if strings.HasPrefix(body[i:], "true") {
				appendSeg("true", boolNullColor)
				i += 4
				continue
			}
			if strings.HasPrefix(body[i:], "false") {
				appendSeg("false", boolNullColor)
				i += 5
				continue
			}
			if strings.HasPrefix(body[i:], "null") {
				appendSeg("null", boolNullColor)
				i += 4
				continue
			}
			r, size := utf8.DecodeRuneInString(body[i:])
			appendSeg(string(r), textColor)
			i += size
		}
	}
	flush()
	return spans
}

func isJSONKeyToken(body string, end int) bool {
	for k := end; k < len(body); k++ {
		switch body[k] {
		case ' ', '\n', '\r', '\t':
			continue
		case ':':
			return true
		default:
			return false
		}
	}
	return false
}

func cloneHeaders(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cookiesFromHeaders(headers map[string]string) []string {
	if len(headers) == 0 {
		return nil
	}
	for k, v := range headers {
		if strings.EqualFold(k, "Set-Cookie") && strings.TrimSpace(v) != "" {
			return []string{v}
		}
	}
	return nil
}

func headerLines(headers map[string]string) []string {
	if len(headers) == 0 {
		return nil
	}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+": "+headers[k])
	}
	return out
}

// statusColor colors a status line by its class.
func statusColor(tk *theme.Tokens, code int) ui.Color {
	switch {
	case code >= 200 && code < 300:
		return tk.Success
	case code >= 300 && code < 400:
		return tk.Info
	case code >= 400 && code < 500:
		return tk.Warning
	case code >= 500:
		return tk.Error
	default:
		return tk.TextTertiary
	}
}

// variableHighlightText renders text with its {{var}} placeholders highlighted:
// ones the active environment or workspace defines show as a filled chip, ones
// that resolve to nothing show warning-colored and underlined.
func variableHighlightText(c *ui.Context, tk *theme.Tokens, text string, size float32, known func(string) bool) {
	spans := variableSpans(text, tk, known)
	if len(spans) == 0 {
		ui.Text(c, text).Font("monospace").FontSize(size).TextColor(tk.TextSecondary).SingleLine().MinWidth(0)
		return
	}
	ui.RichText(c, spans...).Font("monospace").FontSize(size).SingleLine().MinWidth(0).Shrink(1)
}

// variableSpans splits text into spans, marking each {{...}} token by whether
// `known` defines its name.
func variableSpans(text string, tk *theme.Tokens, known func(string) bool) []ui.Span {
	if !strings.Contains(text, "{{") {
		return nil
	}
	plain := ui.Span{Text: "", Color: tk.TextSecondary}
	spans := make([]ui.Span, 0, 8)
	add := func(s ui.Span) {
		s.Font, s.Size = "monospace", 0
		if s.Text != "" {
			spans = append(spans, s)
		}
	}
	for len(text) > 0 {
		start := strings.Index(text, "{{")
		if start < 0 {
			plain.Text = text
			add(plain)
			break
		}
		if start > 0 {
			plain.Text = text[:start]
			add(plain)
			text = text[start:]
		}
		end := strings.Index(text, "}}")
		if end < 0 {
			// An unclosed placeholder: mark the rest as malformed.
			add(ui.Span{Text: text, Color: tk.Warning, WavyUnderline: true, Weight: 600})
			break
		}
		end += 2
		token := text[:end]
		name := strings.TrimSpace(text[2 : end-2])
		if known != nil && known(name) {
			// Defined: a subtle chip so it reads as a resolved value.
			add(ui.Span{Text: token, Color: tk.Primary, Weight: 600, Background: tk.PrimaryFillSecondary})
		} else {
			// Undefined: warning-colored and wavy-underlined.
			add(ui.Span{Text: token, Color: tk.Warning, Weight: 600, WavyUnderline: true})
		}
		text = text[end:]
	}
	return spans
}

// persistDraft saves the editor's URL/body/params/headers back to the request.
func (s *shellState) persistDraft(services *app.Services, reqID string, rs *requestState) {
	req, ok := s.requestByID(reqID)
	if !ok || services == nil {
		return
	}
	req.Method = rs.Method
	req.URL = rs.URL
	req.Body = rs.Body
	req.BodyType = rs.BodyType
	req.Auth = rs.Auth
	req.Params = rowsToParams(rs.Params)
	req.Headers = rowsToParams(rs.Headers)
	if err := services.Collections.SaveRequest(context.Background(), &req); err != nil {
		s.err = err.Error()
		return
	}
	// Keep the loaded copy in step so the pane and tree see the saved fields.
	for i := range s.requests {
		if s.requests[i].ID == reqID {
			s.requests[i] = req
			break
		}
	}
}
