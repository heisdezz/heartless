package screens

import (
	"context"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/app"
	domainreq "heartless/internal/domain/request"
	domainvar "heartless/internal/domain/variable"
)

// scriptEditorRows is how tall the script editor starts, in lines.
const scriptEditorRows = 14

// scriptPlaceholder hints at what the objects hold while the script is empty.
const scriptPlaceholder = "// Runs after every response.\n" +
	"// req: id, method, url, body, bodyType, params, headers, authType\n" +
	"// res: status, statusText, headers, body, json(), size, contentType, url\n" +
	"// env.get(name), env.set(name, value), env.all()\n" +
	"// test(\"name\", () => { if (res.status !== 200) throw new Error(\"…\") })\n" +
	"console.log(res.status, res.json());"

// scriptTabMark returns the assertion color for the Script tab's dot, shown
// once a script has run: green when every test passed, red when one failed.
func scriptTabMark(rs *requestState, tk *theme.Tokens) (ui.Color, bool) {
	if rs == nil || rs.script == nil || rs.script.result == nil {
		return tk.TextTertiary, false
	}
	r := rs.script.result
	switch {
	case r.Err != "" || (r.Assertion > 0 && r.Passed < r.Assertion):
		return tk.Error, true
	case r.Assertion > 0:
		return tk.Success, true
	default:
		return tk.Info, true
	}
}

// scriptEditor is the Script tab: one inline editor whose text is colored in
// place, and the last run's output beneath it. The script runs by itself when
// a response arrives; the button repeats it against the current response.
func scriptEditor(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services, req domainreq.Request, rs *requestState) {
	st.saveScriptDraft(services, req, rs)
	if st.runner == nil {
		st.runner = newScriptRunner()
	}

	ui.Column(c).AlignSelf(ui.Stretch).Grow(1).MinWidth(0).Gap(10).Children(func() {
		ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, "Post-response").FontSize(textSize(c, 12)).TextColor(tk.TextTertiary).Shrink(0)
			ui.Box(c).Grow(1).Basis(0)
			if action.Button(c, "Run again", action.ButtonProps{Variant: action.Text, Size: theme.Small}).Shrink(0).Clicked() {
				st.runScreenScript(c, services, req, rs)
			}
			if action.Button(c, "Tokens", action.ButtonProps{Variant: action.Text, Size: theme.Small}).Shrink(0).Clicked() {
				rs.Script = insertScriptSnippet(rs.Script, scriptTokenSnippet)
			}
		})

		// One editor, colored in place: keywords, strings, the runtime's
		// objects and {{variables}} read at a glance while typing. Lines gives
		// it a tall, scrollable body; a field stretched by flex is not
		// clickable, so its height comes from Lines rather than Grow.
		ui.Box(c).AlignSelf(ui.Stretch).MinWidth(0).Shrink(0).Children(func() {
			area := field.TextArea(c, &rs.Script, field.TextAreaProps{
				Placeholder: scriptPlaceholder,
				MinRows:     scriptEditorRows,
				MaxRows:     scriptEditorRows + 8,
			}).AlignSelf(ui.Stretch).Font("monospace").FontSize(textSize(c, 12)).TabSize(2).
				CaretColor(tk.Primary).SelectionColor(tk.PrimaryFillSecondary)
			if ranges := scriptStyleRanges(rs.Script, tk, st.knowsVariable); len(ranges) > 0 {
				area.TextRanges(textRanges(ranges)...)
			}
		})

		scriptOutput(c, tk, st, services, rs)
	})
}

// scriptOutput shows the last run: console lines and assertions, the variables
// the script changed, and their save state.
func scriptOutput(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services, rs *requestState) {
	run := rs.script
	if run == nil {
		ui.Text(c, "Output appears here after the response arrives.").
			FontSize(textSize(c, 12)).TextColor(tk.TextTertiary)
		return
	}
	res := run.result
	if res == nil {
		if run.err != "" {
			ui.Text(c, run.err).FontSize(textSize(c, 12)).TextColor(tk.Error)
		}
		return
	}

	ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Gap(8).Wrap().Children(func() {
		if res.Err != "" {
			ui.Text(c, "Error").FontSize(textSize(c, 12)).FontWeight(600).TextColor(tk.Error).Shrink(0)
		} else if res.Assertion > 0 && res.Passed < res.Assertion {
			ui.Text(c, plural(res.Assertion-res.Passed, "failed test", "failed tests")).
				FontSize(textSize(c, 12)).FontWeight(600).TextColor(tk.Error).Shrink(0)
		} else if res.Assertion > 0 {
			ui.Text(c, plural(res.Assertion, "test", "tests")+" passed").
				FontSize(textSize(c, 12)).FontWeight(600).TextColor(tk.Success).Shrink(0)
		}
		ui.Text(c, humanDuration(res.Duration)).FontSize(textSize(c, 11)).TextColor(tk.TextTertiary).Shrink(0)
		if len(res.Updates) > 0 && !run.applied {
			if action.Button(c, "Save variables", action.ButtonProps{Size: theme.Small}).Shrink(0).Clicked() {
				st.applyScriptVariables(services, rs)
			}
		} else if len(res.Updates) > 0 {
			ui.Text(c, "Variables saved").FontSize(textSize(c, 11)).TextColor(tk.Success).Shrink(0)
		}
	})

	if res.Err != "" {
		ui.Text(c, res.Err).Font("monospace").FontSize(textSize(c, 12)).TextColor(tk.Error).MinWidth(0)
	}

	if len(res.Output) > 0 {
		ui.Column(c).AlignSelf(ui.Stretch).MinWidth(0).Gap(2).
			Border(1, tk.BorderSecondary).Radius(tk.BorderRadiusSM).Padding(8).Children(func() {
			for _, line := range res.Output {
				color := tk.TextSecondary
				switch line.Level {
				case "error", "fail":
					color = tk.Error
				case "warn":
					color = tk.Warning
				case "pass":
					color = tk.Success
				}
				ui.Row(c).MinWidth(0).Gap(6).Children(func() {
					ui.Text(c, line.Level).FontSize(textSize(c, 10)).TextColor(color).Width(36).Shrink(0)
					ui.Text(c, line.Text).Font("monospace").FontSize(textSize(c, 12)).TextColor(color).MinWidth(0)
				})
			}
		})
	}

	if len(res.Updates) > 0 {
		ui.Column(c).AlignSelf(ui.Stretch).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, "Variable updates").FontSize(textSize(c, 11)).TextColor(tk.TextTertiary).Shrink(0)
			ui.Column(c).AlignSelf(ui.Stretch).MinWidth(0).Border(1, tk.BorderSecondary).Radius(tk.BorderRadiusSM).
				Padding(8).Gap(4).Children(func() {
				for _, u := range res.Updates {
					ui.Row(c).MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, u.Name).Font("monospace").FontSize(textSize(c, 12)).FontWeight(600).Shrink(0)
						if u.Existed {
							ui.Text(c, u.Previous).Font("monospace").FontSize(textSize(c, 11)).
								TextColor(tk.TextTertiary).SingleLine().Shrink(1)
							ui.Text(c, "→").FontSize(textSize(c, 11)).TextColor(tk.TextTertiary).Shrink(0)
						} else {
							ui.Text(c, "new").FontSize(textSize(c, 10)).TextColor(tk.Info).Shrink(0)
						}
						ui.Text(c, u.Value).Font("monospace").FontSize(textSize(c, 12)).
							TextColor(tk.Primary).SingleLine().MinWidth(0)
					})
				}
			})
		})
	}
}

// scriptEnv captures the variables a script may read and update.
func (s *shellState) scriptEnv() ScriptEnvironment {
	env := ScriptEnvironment{EnvironmentID: s.environmentID}
	for i := range s.variables {
		v := &s.variables[i]
		if v.EnvironmentID != "" && v.EnvironmentID == s.environmentID {
			env.Environment = append(env.Environment, *v)
		} else if v.EnvironmentID == "" {
			env.Workspace = append(env.Workspace, *v)
		}
	}
	return env
}

// runScreenScript runs the script now, for the Run again button. The run is
// queued like an automatic one, so a slow script never blocks a frame.
func (s *shellState) runScreenScript(c *ui.Context, services *app.Services, req domainreq.Request, rs *requestState) {
	if rs == nil || rs.Script == "" {
		return
	}
	if s.runner == nil {
		s.runner = newScriptRunner()
	}
	s.runner.enqueue(s.newScriptTask(c, req, rs))
}

// saveScriptDraft writes the editor's script back to the stored request so it
// survives reopening. It only saves when the text actually changed.
func (s *shellState) saveScriptDraft(services *app.Services, req domainreq.Request, rs *requestState) {
	if services == nil || req.ID == "" || rs == nil {
		return
	}
	base, ok := s.requestByID(req.ID)
	if !ok {
		base = req
	}
	if base.Script == rs.Script {
		return
	}
	base.Script = rs.Script
	if err := services.Collections.SaveRequest(context.Background(), &base); err != nil {
		s.err = err.Error()
		return
	}
	for i := range s.requests {
		if s.requests[i].ID == base.ID {
			s.requests[i] = base
			return
		}
	}
}

// applyScriptVariables persists the variables the script's last run changed,
// into the environment they came from (or the active environment for new ones).
func (s *shellState) applyScriptVariables(services *app.Services, rs *requestState) {
	if services == nil || rs == nil || rs.script == nil || rs.script.result == nil {
		return
	}
	run := rs.script
	for _, u := range run.result.Updates {
		envID := s.environmentID
		if u.Workspace {
			envID = ""
		}
		var existing *domainvar.Variable
		for i := range s.variables {
			v := &s.variables[i]
			if v.Name == u.Name && v.EnvironmentID == envID {
				existing = v
				break
			}
		}
		if existing != nil {
			v := *existing
			v.Value = u.Value
			if err := services.Variables.Save(context.Background(), v.ID, s.workspaceID, envID, &v); err != nil {
				run.err = err.Error()
				return
			}
			continue
		}
		v := domainvar.NewVariable(u.Name, u.Value, "text", false)
		v.Scope = "environment"
		if envID == "" {
			v.Scope = "workspace"
		}
		v.WorkspaceID = s.workspaceID
		v.EnvironmentID = envID
		if err := services.Variables.Save(context.Background(), v.ID, s.workspaceID, envID, v); err != nil {
			run.err = err.Error()
			return
		}
	}
	_ = s.reloadVariables(context.Background(), services)
	run.applied = true
}

// scriptTokenSnippet is what the Tokens button inserts: the objects and
// helpers a script can use.
const scriptTokenSnippet = "\n// req: id method url body bodyType params headers authType\n" +
	"// res: status statusText headers body json() size contentType url\n" +
	"// env.get(name)  env.set(name, value)  env.all()\n" +
	"// test(\"name\", () => {})\n"

// insertScriptSnippet appends snippet to src when it is not already there.
func insertScriptSnippet(src, snippet string) string {
	if src == "" {
		return snippet
	}
	return src + snippet
}
