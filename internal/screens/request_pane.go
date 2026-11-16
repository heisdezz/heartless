package screens

import (
	"context"
	"encoding/json"
	"strings"

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
	st.loadResponseFromHistory(services, req.ID, rs)
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

// requestEditor is the top pane: method, URL, send and the request tabs.
func requestEditor(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services, req domainreq.Request, rs *requestState) {
	_, running, _, _, _ := st.responseView(req.ID)

	ui.Column(c).Grow(1).Basis(0).AlignSelf(ui.Stretch).MinWidth(0).Children(func() {
		ui.Row(c).Height(space(c, 52)).AlignSelf(ui.Stretch).PaddingX(16).Gap(8).AlignItems(ui.Center).
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
				ui.Text(c, "→ "+st.previewURL(rs.URL)).Font("monospace").FontSize(textSize(c, 11)).
					TextColor(tk.TextTertiary).SingleLine().MinWidth(0)
			})
		}

		// Tabs, as plain buttons so the pane keeps filling the width.
		ui.Row(c).Height(space(c, 38)).AlignSelf(ui.Stretch).PaddingX(16).Gap(4).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
			for i, name := range []string{"Params", "Headers", "Body", "Auth", "Script"} {
				i, name := i, name
				btn := action.Button(c, name, action.ButtonProps{Variant: action.Text})
				if rs.Tab == i {
					btn.FontWeight(600).TextColor(tk.Text).Underline()
				} else {
					btn.TextColor(tk.TextSecondary)
				}
				// A dot on the tab mirrors the assertion results: green when every
				// test passed, red when one failed.
				if i == 4 {
					if color, ok := scriptTabMark(rs, tk); ok {
						ui.Box(c).Width(6).Height(6).Radius(3).Background(color).Shrink(0)
					}
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
				case 4:
					scriptEditor(c, tk, st, services, req, rs)
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
				ui.Text(c, "This request has no body.").FontSize(textSize(c, 13)).TextColor(tk.TextTertiary)
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
	ui.Text(c, header).FontSize(textSize(c, 11)).TextColor(tk.TextTertiary).Shrink(0)
	ui.Scroll(c).Height(max(160, textSize(c, 12)*12)).Shrink(0).MinWidth(0).Children(func() {
		ui.Box(c).AlignSelf(ui.Stretch).MinWidth(0).Padding(8).Radius(tk.BorderRadiusSM).
			Background(tk.FillQuaternary).Children(func() {
			if spans := jsonHighlightSpans(converted, tk); len(spans) > 0 {
				ui.RichText(c, spans...).Font("monospace").FontSize(textSize(c, 12)).MinWidth(0)
				return
			}
			ui.Text(c, converted).Font("monospace").FontSize(textSize(c, 12)).MinWidth(0)
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
		ui.Row(c).Height(space(c, 28)).AlignSelf(ui.Stretch).PaddingX(8).Gap(6).AlignItems(ui.Center).
			Background(tk.FillQuaternary).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).
			Children(func() {
				ui.Box(c).Width(20).Shrink(0)
				ui.Text(c, "Key").FontSize(textSize(c, 12)).TextColor(tk.TextSecondary).Grow(1).Basis(0).MinWidth(0)
				ui.Text(c, "Value").FontSize(textSize(c, 12)).TextColor(tk.TextSecondary).Grow(1).Basis(0).MinWidth(0)
				ui.Box(c).Width(24).Shrink(0)
			})

		remove := -1
		for i, row := range *rows {
			i, row := i, row
			ui.Row(c).Height(space(c, 36)).AlignSelf(ui.Stretch).PaddingX(8).Gap(6).AlignItems(ui.Center).
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
		ui.Row(c).Height(space(c, 32)).AlignSelf(ui.Stretch).PaddingX(8).AlignItems(ui.Center).Children(func() {
			if action.Button(c, "+ Add", action.ButtonProps{Variant: action.Text, Size: theme.Small}).Shrink(0).Clicked() {
				*rows = append(*rows, &kv{On: true})
			}
		})
	})
}

// variableHighlightText renders text with its {{var}} placeholders highlighted:
// ones the active environment or workspace defines show as a filled chip, ones
// that resolve to nothing show warning-colored and underlined.
func variableHighlightText(c *ui.Context, tk *theme.Tokens, text string, size float32, known func(string) bool) {
	size = textSize(c, size)
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
