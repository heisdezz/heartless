package screens

import (
	"strconv"

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

// authToolbar renders the Headers | Bearer ▾ | Settings | Info bar with the auth
// type menu, and reports a chosen action ("settings" or "info"). The bar element
// is the anchor for the menu's open state.
func authToolbar(c *ui.Context, tk *theme.Tokens, rs *requestState) string {
	chosenAction := ""
	bar := ui.Row(c)
	open := ui.Local(bar, "authMenu", func() bool { return false })

	bar.Height(34).AlignSelf(ui.Stretch).PaddingX(16).Gap(6).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
		// The Headers tab label the bar sits beside.
		ui.Text(c, "Headers").FontSize(13).FontWeight(600).TextColor(tk.Text).Shrink(0)
		if n := len(rs.Headers); n > 0 {
			ui.Text(c, strconv.Itoa(n)).FontSize(11).TextColor(tk.TextTertiary).Shrink(0)
		}

		chosen := overlay.DropdownMenu(c, open, overlay.DropdownMenuProps{Items: authMenuItems(rs.Auth.Type)}, func(t ui.Element) {
			authTrigger(t, c, tk, rs)
		})
		if chosen != "" {
			rs.Auth.Type = domainreq.AuthType(chosen)
		}

		for _, act := range []struct{ key, label string }{{"settings", "Settings"}, {"info", "Info"}} {
			act := act
			btn := action.Button(c, act.label, action.ButtonProps{Variant: action.Text})
			if btn.TextColor(tk.TextSecondary).Clicked() {
				chosenAction = act.key
			}
		}
	})
	return chosenAction
}

// authTrigger shows the current auth type, with a check mark inline when it is
// applied, and a chevron for the menu.
func authTrigger(t ui.Element, c *ui.Context, tk *theme.Tokens, rs *requestState) {
	t.Height(24).PaddingX(6).Gap(6).Radius(tk.BorderRadiusSM).AlignItems(ui.Center).
		TextColor(tk.Text).Cursor(ui.CursorPointer)
	if t.Hovered() {
		t.Background(tk.FillTertiary)
	}
	t.Children(func() {
		ui.Text(c, authShortLabel(rs.Auth.Type)).FontSize(13)
		if authApplied(rs.Auth) {
			ui.Icon(c, lucide.Asterisk()).FontSize(11).TextColor(tk.Success)
		}
		ui.Icon(c, lucide.ChevronDown()).FontSize(12).TextColor(tk.TextTertiary)
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
	var items []overlay.MenuItem
	for i, o := range authOptions {
		// A divider separates the schemes from Inherit/No Auth.
		if o.typ == domainreq.AuthInherit && i > 0 {
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

// authFields renders the credential inputs for the current auth type.
func authFields(c *ui.Context, tk *theme.Tokens, rs *requestState) {
	ui.Column(c).AlignSelf(ui.Stretch).Gap(8).Children(func() {
		switch rs.Auth.Type {
		case domainreq.AuthBearer, domainreq.AuthJWT:
			label(c, tk, "Token")
			ui.Box(c).AlignSelf(ui.Stretch).MinWidth(0).Children(func() {
				field.Input(c, &rs.Auth.Token, field.InputProps{Placeholder: "{{token}} or a token"}).Grow(1)
			})
			ui.Text(c, "Sent as Authorization: Bearer <token>.").FontSize(11).TextColor(tk.TextTertiary)
		case domainreq.AuthBasic, domainreq.AuthDigest, domainreq.AuthNTLM:
			label(c, tk, "Username")
			ui.Box(c).AlignSelf(ui.Stretch).MinWidth(0).Children(func() {
				field.Input(c, &rs.Auth.Username, field.InputProps{Placeholder: "{{username}}"}).Grow(1)
			})
			label(c, tk, "Password")
			ui.Box(c).AlignSelf(ui.Stretch).MinWidth(0).Children(func() {
				field.Input(c, &rs.Auth.Password, field.InputProps{Placeholder: "{{password}}"}).Grow(1)
			})
		case domainreq.AuthAPIKey:
			label(c, tk, "Key")
			ui.Box(c).AlignSelf(ui.Stretch).MinWidth(0).Children(func() {
				field.Input(c, &rs.Auth.Key, field.InputProps{Placeholder: "{{apiKey}}"}).Grow(1)
			})
			label(c, tk, "Header")
			ui.Box(c).AlignSelf(ui.Stretch).MinWidth(0).Children(func() {
				field.Input(c, &rs.Auth.Header, field.InputProps{Placeholder: "X-API-Key"}).Grow(1)
			})
		case domainreq.AuthNone:
			ui.Text(c, "No authentication is applied.").FontSize(13).TextColor(tk.TextTertiary)
		case domainreq.AuthInherit, "":
			ui.Text(c, "Inherited from the collection or workspace.").FontSize(13).TextColor(tk.TextTertiary)
		default:
			ui.Text(c, authLabel(rs.Auth.Type)+" is not supported yet.").FontSize(13).TextColor(tk.TextTertiary)
		}
	})
}

func label(c *ui.Context, tk *theme.Tokens, s string) {
	ui.Text(c, s).FontSize(12).TextColor(tk.TextSecondary)
}
