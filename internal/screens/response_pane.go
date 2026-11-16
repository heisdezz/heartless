package screens

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

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

// responsePanel is the bottom pane: status, timing and tabbed response details.
func responsePanel(c *ui.Context, tk *theme.Tokens, rs *requestState, resp *responseSnapshot, running bool, errText string, respTab int, sentBody string) {
	ui.Column(c).Fill().MinWidth(0).Children(func() {
		ui.Row(c).Height(space(c, 36)).PaddingX(16).Gap(12).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
			ui.Text(c, "Response").FontSize(textSize(c, 13)).Bold().Shrink(0)
			switch {
			case running:
				ui.Text(c, "Sending…").FontSize(textSize(c, 12)).TextColor(tk.TextTertiary)
			case resp != nil:
				ui.Text(c, resp.Status).FontSize(textSize(c, 12)).FontWeight(600).TextColor(statusColor(tk, resp.Code)).Shrink(0)
				ui.Text(c, resp.Duration).FontSize(textSize(c, 12)).TextColor(tk.TextTertiary).Shrink(0)
				ui.Text(c, resp.Size).FontSize(textSize(c, 12)).TextColor(tk.TextTertiary).Shrink(0)
			case errText != "":
				ui.Text(c, errText).FontSize(textSize(c, 12)).TextColor(tk.Error).SingleLine().MinWidth(0)
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
								ui.Text(c, line).Font("monospace").FontSize(textSize(c, 12)).TextColor(tk.Text)
							}
						}
					case respTab == 2:
						if len(resp.Cookies) == 0 {
							ui.Text(c, "No cookies").TextColor(tk.TextSecondary)
						} else {
							for _, ck := range resp.Cookies {
								ui.Text(c, ck).Font("monospace").FontSize(textSize(c, 12)).TextColor(tk.Text)
							}
						}
					case respTab == 3:
						if len(resp.Timeline) == 0 {
							ui.Text(c, "No timeline data").TextColor(tk.TextSecondary)
						} else {
							for _, line := range resp.Timeline {
								ui.Text(c, line).Font("monospace").FontSize(textSize(c, 12)).TextColor(tk.Text)
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
					ui.RichText(c, spans...).Font("monospace").FontSize(textSize(c, 12)).Selectable()
					return
				}
			}
			ui.Text(c, display).Font("monospace").FontSize(textSize(c, 12)).TextColor(tk.Text).Selectable()
		})
		btn := action.Button(c, "Copy", action.ButtonProps{Variant: action.Text, Size: theme.Small}).Shrink(0)
		if btn.Clicked() && trimmed != "" {
			c.Services().WriteClipboard(body)
			rs.copyNotice = "Copied"
		}
	})
	if rs.copyNotice != "" {
		ui.Text(c, rs.copyNotice).FontSize(textSize(c, 11)).TextColor(tk.TextTertiary)
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
