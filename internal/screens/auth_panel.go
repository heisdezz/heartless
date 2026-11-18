package screens

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	domainreq "heartless/internal/domain/request"
)

// authOptions lists the auth types the menu shows, in the order of the reference
// UI: the concrete schemes, then inherit and none.
var authOptions = []struct {
	typ   domainreq.AuthType
	label string
}{
	{domainreq.AuthAPIKey, "API Key"},
	{domainreq.AuthAWS, "AWS Signature"},
	{domainreq.AuthBasic, "Basic Auth"},
	{domainreq.AuthBearer, "Bearer Token"},
	{domainreq.AuthDigest, "Digest Auth"},
	{domainreq.AuthJWT, "JWT Bearer"},
	{domainreq.AuthOAuth1, "OAuth 1.0"},
	{domainreq.AuthOAuth2, "OAuth 2.0"},
	{domainreq.AuthNTLM, "NTLM Auth"},
	{domainreq.AuthInherit, "Inherit from Parent"},
	{domainreq.AuthNone, "No Auth"},
}

// authLabel returns the menu label for an auth type.
func authLabel(t domainreq.AuthType) string {
	if t == "" {
		return "Inherit"
	}
	for _, o := range authOptions {
		if o.typ == t {
			return o.label
		}
	}
	return string(t)
}

// authShowsToken reports whether the type has a single token field, as shown
// inline in the toolbar the way "Bearer" is in the reference.
func authShowsToken(t domainreq.AuthType) bool {
	switch t {
	case domainreq.AuthBearer, domainreq.AuthJWT:
		return true
	}
	return false
}

// authToolbar renders the auth header: the scheme label beside the type menu,
// with the per-scheme actions at the far end. The auth may be a request's or a
// workspace's default.
func authToolbar(c *ui.Context, tk *theme.Tokens, a *domainreq.Auth) {
	authToolbarWith(c, tk, a, true)
}

// authToolbarWith renders the auth header; inherit is dropped from the menu at
// scopes with no parent, as the workspace's default auth has none.
func authToolbarWith(c *ui.Context, tk *theme.Tokens, a *domainreq.Auth, inherit bool) {
	bar := ui.Row(c)
	open := ui.Local(bar, "authMenu", func() bool { return false })

	bar.Height(40).AlignSelf(ui.Stretch).AlignItems(ui.Center).Gap(8).
		BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).PaddingY(0).Children(func() {
		ui.Text(c, "Auth").FontSize(textSize(c, 13)).FontWeight(600).TextColor(tk.Text).Shrink(0)

		ui.Box(c).Shrink(0).Children(func() {
			chosen := overlay.DropdownMenu(c, open, overlay.DropdownMenuProps{Items: authMenuItemsWith(a.Type, inherit)}, func(t ui.Element) {
				authTrigger(t, c, tk, a)
			})
			if chosen != "" {
				a.Type = domainreq.AuthType(chosen)
			}
		})

		ui.Box(c).Grow(1)

		for _, name := range []string{"Settings", "Info"} {
			action.Button(c, name, action.ButtonProps{Variant: action.Text, Size: theme.Small}).TextColor(tk.TextSecondary)
		}
	})
}

// authTrigger styles the type menu's trigger as a compact select: the scheme,
// a check when credentials are set, and a chevron.
func authTrigger(t ui.Element, c *ui.Context, tk *theme.Tokens, a *domainreq.Auth) {
	t.Height(28).PaddingX(10).Gap(6).Radius(tk.BorderRadius).AlignItems(ui.Center).
		Background(tk.FillQuaternary).TextColor(tk.Text).Cursor(ui.CursorPointer)
	if t.Hovered() {
		t.Background(tk.FillTertiary)
	}
	t.Children(func() {
		ui.Text(c, authShortLabel(a.Type)).FontSize(textSize(c, 13))
		if authApplied(*a) {
			ui.Icon(c, lucide.Asterisk()).FontSize(textSize(c, 11)).TextColor(tk.Success)
		}
		ui.Icon(c, lucide.ChevronDown()).FontSize(textSize(c, 12)).TextColor(tk.TextTertiary)
	})
}

// authShortLabel is the trigger's text: the type's short name.
func authShortLabel(t domainreq.AuthType) string {
	switch t {
	case domainreq.AuthBearer, domainreq.AuthJWT:
		return "Bearer"
	case domainreq.AuthBasic, domainreq.AuthDigest, domainreq.AuthNTLM:
		return "Basic"
	case domainreq.AuthAPIKey:
		return "API Key"
	case domainreq.AuthNone:
		return "No Auth"
	case domainreq.AuthInherit, "":
		return "Inherit"
	default:
		return authLabel(t)
	}
}

// authApplied reports whether the request carries credentials to send.
func authApplied(a domainreq.Auth) bool {
	switch a.Type {
	case domainreq.AuthBearer, domainreq.AuthJWT:
		return a.Token != ""
	case domainreq.AuthBasic, domainreq.AuthDigest, domainreq.AuthNTLM:
		return a.Username != "" || a.Password != ""
	case domainreq.AuthAPIKey:
		return a.Key != ""
	}
	return false
}

// authMenuItems builds the auth menu rows, marking the current type with a check.
func authMenuItems(current domainreq.AuthType) []overlay.MenuItem {
	return authMenuItemsWith(current, true)
}

// authMenuItemsWith builds the menu rows; inherit is omitted at scopes that have
// no parent to inherit from, as the workspace's default auth has none.
func authMenuItemsWith(current domainreq.AuthType, inherit bool) []overlay.MenuItem {
	var items []overlay.MenuItem
	for i, o := range authOptions {
		if o.typ == domainreq.AuthInherit && !inherit {
			continue
		}
		// A divider separates the schemes from Inherit/No Auth.
		if i > 0 && ((o.typ == domainreq.AuthInherit) || (!inherit && o.typ == domainreq.AuthNone)) {
			items = append(items, overlay.MenuItem{Divider: true})
		}
		it := overlay.MenuItem{Key: string(o.typ), Label: o.label}
		if o.typ == current || (current == "" && o.typ == domainreq.AuthInherit) {
			it.Icon = lucide.Check()
		}
		items = append(items, it)
	}
	items = append(items, overlay.MenuItem{Divider: true})
	items = append(items, overlay.MenuItem{Key: "__copy_folder", Label: "Copy from Folder", Icon: lucide.Copy(), Disabled: true})
	return items
}

// authFields renders the credential inputs for the current auth type, one
// labelled field per row with a hint under the group.
func authFields(c *ui.Context, tk *theme.Tokens, a *domainreq.Auth) {
	ui.Column(c).AlignSelf(ui.Stretch).MinWidth(0).Gap(14).Children(func() {
		switch a.Type {
		case domainreq.AuthBearer, domainreq.AuthJWT:
			authTextField(c, tk, "Token", &a.Token, "{{token}} or a token")
			authHint(c, tk, "Sent as Authorization: Bearer <token>.")
		case domainreq.AuthBasic, domainreq.AuthDigest, domainreq.AuthNTLM:
			authTextField(c, tk, "Username", &a.Username, "{{username}}")
			authTextField(c, tk, "Password", &a.Password, "{{password}}")
		case domainreq.AuthAPIKey:
			authTextField(c, tk, "Key", &a.Key, "{{apiKey}}")
			authTextField(c, tk, "Header", &a.Header, "X-API-Key")
		case domainreq.AuthNone:
			authNote(c, tk, "No authentication is applied.")
		case domainreq.AuthInherit, "":
			authNote(c, tk, "Inherited from the collection or workspace.")
		default:
			authNote(c, tk, authLabel(a.Type)+" is not supported yet.")
		}
	})
}

// authTextField renders a label over a full-width input in a capped column.
func authTextField(c *ui.Context, tk *theme.Tokens, name string, value *string, placeholder string) {
	ui.Column(c).AlignSelf(ui.Stretch).MaxWidth(560).Gap(6).Children(func() {
		ui.Text(c, name).FontSize(textSize(c, 12)).FontWeight(500).TextColor(tk.TextSecondary)
		ui.Box(c).AlignSelf(ui.Stretch).MinWidth(0).Clip().Children(func() {
			field.Input(c, value, field.InputProps{Placeholder: placeholder}).Grow(1).Basis(0).Font("monospace")
		})
	})
}

// authHint explains how the credentials are sent.
func authHint(c *ui.Context, tk *theme.Tokens, text string) {
	ui.Text(c, text).FontSize(textSize(c, 11)).TextColor(tk.TextTertiary).MaxWidth(560)
}

// authNote fills the tab when the type takes no credentials.
func authNote(c *ui.Context, tk *theme.Tokens, text string) {
	ui.Text(c, text).FontSize(textSize(c, 13)).TextColor(tk.TextTertiary)
}
