package screens

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/app"
	domaincol "heartless/internal/domain/collection"
	domainreq "heartless/internal/domain/request"
)

// Context-menu keys used by the collection tree.
const (
	menuFolderSettings   = "folder-settings"
	menuSendAll          = "send-all"
	menuRename           = "rename"
	menuDuplicate        = "duplicate"
	menuDelete           = "delete"
	menuHTTP             = "new-http"
	menuGraphQL          = "new-graphql"
	menuGRPC             = "new-grpc"
	menuWebSocket        = "new-websocket"
	menuFolder           = "new-folder"
	menuNewCollection    = "new-collection"
	menuImportCollection = "import-collection"
)

var rowTransition = ui.ElementTransition{Duration: 120 * time.Millisecond, Colors: true}

// treeMenu is the context menu shown on a folder or collection.
var treeMenu = []overlay.MenuItem{
	{Key: menuFolderSettings, Label: "Folder Settings", Icon: lucide.Settings()},
	{Key: menuSendAll, Label: "Send All"},
	{Divider: true},
	{Key: menuRename, Label: "Rename", Icon: lucide.Pencil(), Hotkey: "mod+enter"},
	{Key: menuDuplicate, Label: "Duplicate", Icon: lucide.Copy(), Hotkey: "mod+d"},
	{Key: menuDelete, Label: "Delete", Icon: lucide.Trash2(), Danger: true},
	{Divider: true},
	{Key: menuHTTP, Label: "HTTP", Icon: lucide.Plus()},
	{Key: menuGraphQL, Label: "GraphQL", Icon: lucide.Plus()},
	{Key: menuGRPC, Label: "gRPC", Icon: lucide.Plus()},
	{Key: menuWebSocket, Label: "WebSocket", Icon: lucide.Plus()},
	{Divider: true},
	{Key: menuFolder, Label: "Folder", Icon: lucide.Plus()},
}

// emptyMenu is the context menu shown on the tree's empty area.
var emptyMenu = []overlay.MenuItem{
	{Key: menuNewCollection, Label: "New Collection", Icon: lucide.Plus()},
	{Key: menuImportCollection, Label: "Import Collection", Icon: lucide.Download()},
	{Divider: true},
	{Key: menuHTTP, Label: "HTTP", Icon: lucide.Plus()},
	{Key: menuGraphQL, Label: "GraphQL", Icon: lucide.Plus()},
	{Key: menuGRPC, Label: "gRPC", Icon: lucide.Plus()},
	{Key: menuWebSocket, Label: "WebSocket", Icon: lucide.Plus()},
	{Key: menuFolder, Label: "Folder", Icon: lucide.Plus()},
}

// requestMenu is the context menu shown on a saved request.
var requestMenu = []overlay.MenuItem{
	{Key: menuRename, Label: "Rename", Icon: lucide.Pencil(), Hotkey: "mod+enter"},
	{Key: menuDuplicate, Label: "Duplicate", Icon: lucide.Copy(), Hotkey: "mod+d"},
	{Key: menuDelete, Label: "Delete", Icon: lucide.Trash2(), Danger: true},
}

// sidebar renders the side navigation panel with search and the collection tree.
func sidebar(c *ui.Context, st *shellState, tk *theme.Tokens, services *app.Services) {
	layout.DraggableSideNav(c, &st.sidebarWidth, &st.sidebarExpanded, layout.DraggableSideNavProps{
		MinWidth: 72,
		MaxWidth: 420,
		Header: func(nav layout.SideNavState) {
			if !nav.Expanded {
				return
			}
			ui.Column(c).Padding(8, 8).Gap(6).AlignSelf(ui.Stretch).Opacity(1 - nav.Collapse).Children(func() {
				field.Input(c, &st.searchQuery, field.InputProps{
					Placeholder: "Search",
					Prefix:      lucide.Search(),
					Variant:     field.InputFilled,
				})
				if st.err != "" {
					ui.Text(c, st.err).FontSize(12).TextColor(tk.Error).SingleLine()
				}
			})
		},
		Footer: func(nav layout.SideNavState) {
			if !nav.Expanded {
				return
			}
			ui.Row(c).Padding(12).Gap(4).Opacity(1 - nav.Collapse).Children(func() {
				if action.Button(c, "Environments", action.ButtonProps{Variant: action.Text}).Clicked() {
					st.router.Push("/environments")
				}
				if action.Button(c, "Settings", action.ButtonProps{Variant: action.Text}).Clicked() {
					st.router.Push("/settings")
				}
			})
		},
	}, func(nav layout.SideNavState) {
		if !nav.Expanded {
			return
		}
		treeBody(c, st, tk, services, nav)
	})
}

// treeBody draws the tree, filling the nav so right-clicking the empty area
// offers to create things.
func treeBody(c *ui.Context, st *shellState, tk *theme.Tokens, services *app.Services, nav layout.SideNavState) {
	body := ui.Column(c).Grow(1).Basis(0).AlignSelf(ui.Stretch).Opacity(1 - nav.Collapse)
	body.Padding(0, 0, 8, 0)
	if st.workspaceID == "" {
		body.Children(func() {
			ui.Text(c, "Create or select a workspace.").FontSize(13).TextColor(tk.TextTertiary).Padding(10, 0)
		})
		return
	}

	query := strings.ToLower(strings.TrimSpace(st.searchQuery))
	cols := st.filteredCollections(query)

	var menuTarget ui.Element
	body.Children(func() {
		scroll := ui.Scroll(c).Grow(1).Basis(0).AlignSelf(ui.Stretch)
		menuTarget = scroll
		scroll.Children(func() {
			ui.Column(c).Grow(1).AlignSelf(ui.Stretch).MinWidth(0).Children(func() {
				if len(cols) == 0 && query == "" {
					ui.Text(c, "No collections yet — right-click here to add or import one.").FontSize(13).
						TextColor(tk.TextTertiary).Padding(10, 0)
				}
				for i := range cols {
					collectionRow(c, tk, st, services, &cols[i], query)
				}
				// Fills remaining space so the context menu is easy to trigger when empty/sparse.
				ui.Box(c).Grow(1).Basis(0).AlignSelf(ui.Stretch).MinHeight(120)
			})
		})
	})

	switch chosen := overlay.ContextMenu(c, menuTarget, overlay.DropdownMenuProps{Items: emptyMenu}); chosen {
	case menuNewCollection:
		st.createCollection(services)
	case menuImportCollection:
		st.openImportCollection()
	case menuHTTP, menuGraphQL, menuGRPC, menuWebSocket, menuFolder:
		st.applyEmptyMenu(services, chosen)
	}
}

// collectionRow renders a collection and, when expanded, its folders and
// requests. Right-clicking it opens the folder menu.
func collectionRow(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services, col *domaincol.Collection, query string) {
	open := st.isExpanded(col.ID)
	row := treeRow(c, tk, "col:"+col.ID, col.Name, 0, open, lucide.Folder())
	if row.Clicked() {
		st.toggleExpanded(col.ID)
	}
	if chosen := overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: treeMenu}); chosen != "" {
		st.applyFolderMenu(services, col.ID, "", chosen)
		return
	}
	if !open {
		return
	}
	for _, f := range st.childFolders(col.ID, "") {
		f := f
		if query == "" || st.folderMatchesQuery(&f, query) {
			folderRowNested(c, tk, st, services, &f, 1, query)
		}
	}
	for _, r := range st.childRequests(col.ID, "") {
		r := r
		if query == "" || requestMatchesQuery(&r, query) {
			requestRowAt(c, tk, st, services, &r, 1)
		}
	}
}

// folderRowNested renders a folder at a fixed depth and, when expanded, its
// nested children.
func folderRowNested(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services, folder *domaincol.Folder, depth int, query string) {
	open := st.isExpanded(folder.ID)
	row := treeRow(c, tk, "fld:"+folder.ID, folder.Name, depth, open, lucide.Folder())
	if row.Clicked() {
		st.toggleExpanded(folder.ID)
	}
	if chosen := overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: treeMenu}); chosen != "" {
		st.applyFolderMenu(services, folder.CollectionID, folder.ID, chosen)
		return
	}
	if !open {
		return
	}
	for _, f := range st.childFolders(folder.CollectionID, folder.ID) {
		f := f
		if query == "" || st.folderMatchesQuery(&f, query) {
			folderRowNested(c, tk, st, services, &f, depth+1, query)
		}
	}
	for _, r := range st.childRequests(folder.CollectionID, folder.ID) {
		r := r
		if query == "" || requestMatchesQuery(&r, query) {
			requestRowAt(c, tk, st, services, &r, depth+1)
		}
	}
}

// requestRowAt renders a saved request row: method badge, name and menu.
func requestRowAt(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services, req *domainreq.Request, depth int) {
	selected := st.selectedRequestID == req.ID
	row := ui.ButtonBase(c).Key("req:"+req.ID).Label(req.Name).Row().
		Height(28).AlignItems(ui.Center).Justify(ui.Start).Gap(8).
		Radius(tk.BorderRadiusSM).AlignSelf(ui.Stretch).
		Transition(rowTransition).Cursor(ui.CursorPointer).
		Padding(0, 6, 0, float32(depth)*14+22)
	switch {
	case selected:
		row.Background(tk.FillSecondary)
	case row.Hovered():
		row.Background(tk.FillTertiary)
	}
	row.Children(func() {
		methodBadge(c, tk, req)
		if st.renameRequestID == req.ID {
			field.Input(c, &st.renameValue, field.InputProps{Size: theme.Small, Variant: field.InputFilled}).Grow(1).Basis(0)
			return
		}
		ui.Text(c, req.Name).FontSize(13).SingleLine().TextColor(textColor(tk, selected))
	})
	if st.renameRequestID == req.ID {
		if action.Button(c, "Save", action.ButtonProps{Variant: action.Text, Size: theme.Small}).Clicked() {
			st.renameRequest(services, req.ID, st.renameValue)
			st.renameRequestID = ""
		}
		return
	}
	if row.Clicked() {
		st.selectedRequestID = req.ID
		// Clear the collection settings page and show the request on home.
		if st.router.Path() != "/" {
			st.router.Push("/")
		}
	}
	if chosen := overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: requestMenu}); chosen != "" {
		switch chosen {
		case menuRename:
			st.beginRename(*req)
		case menuDuplicate:
			st.duplicateRequest(services, req.ID)
		case menuDelete:
			st.deleteRequest(services, req.ID)
		}
	}
}

// methodBadge shows the request method (or protocol) in small colored type.
func methodBadge(c *ui.Context, tk *theme.Tokens, req *domainreq.Request) {
	label := kindLabel(req.Kind)
	color := tk.TextTertiary
	if req.Kind == domainreq.KindHTTP {
		label = req.Method
		if label == "" {
			label = "GET"
		}
		color = methodColor(tk, label)
	}
	ui.Text(c, label).Font("monospace").FontSize(10).FontWeight(700).TextColor(color).Width(40).SingleLine()
}

// treeRow draws a tree row with an expand chevron, icon and label, left-aligned.
func treeRow(c *ui.Context, tk *theme.Tokens, key, label string, depth int, open bool, icon *ui.SVG) ui.Element {
	row := ui.ButtonBase(c).Key(key).Label(label).Row().
		Height(28).AlignItems(ui.Center).Justify(ui.Start).Gap(6).
		Radius(tk.BorderRadiusSM).AlignSelf(ui.Stretch).
		Transition(rowTransition).Cursor(ui.CursorPointer).
		Padding(0, 6, 0, float32(depth)*14+6)
	if row.Hovered() {
		row.Background(tk.FillTertiary)
	}
	row.Children(func() {
		chev := lucide.ChevronRight()
		if open {
			chev = lucide.ChevronDown()
		}
		ui.Icon(c, chev).FontSize(12).TextColor(tk.TextTertiary)
		if icon != nil {
			ui.Icon(c, icon).FontSize(14).TextColor(tk.TextSecondary)
		}
		ui.Text(c, label).FontSize(13).SingleLine().TextColor(tk.Text)
	})
	return row
}

func textColor(tk *theme.Tokens, selected bool) ui.Color {
	if selected {
		return tk.Text
	}
	return tk.TextSecondary
}

// methodColor colors an HTTP method label.
func methodColor(tk *theme.Tokens, method string) ui.Color {
	switch method {
	case "GET":
		return tk.Success
	case "POST":
		return tk.Warning
	case "PUT", "PATCH":
		return tk.Primary
	case "DELETE":
		return tk.Error
	default:
		return tk.TextTertiary
	}
}

// filteredCollections returns collections matching query by collection, folder, or request fields.
func (s *shellState) filteredCollections(query string) []domaincol.Collection {
	if query == "" {
		return s.collections
	}
	var out []domaincol.Collection
	for _, col := range s.collections {
		if matchesQuery(col.Name, query) || s.collectionHasQueryMatch(col.ID, query) {
			out = append(out, col)
		}
	}
	return out
}

func (s *shellState) collectionHasQueryMatch(colID, query string) bool {
	for _, f := range s.folders {
		if f.CollectionID == colID && s.folderMatchesQuery(&f, query) {
			return true
		}
	}
	for _, r := range s.requests {
		if r.CollectionID == colID && requestMatchesQuery(&r, query) {
			return true
		}
	}
	return false
}

func (s *shellState) folderMatchesQuery(folder *domaincol.Folder, query string) bool {
	if folder == nil {
		return false
	}
	if matchesQuery(folder.Name, query) {
		return true
	}
	for _, f := range s.childFolders(folder.CollectionID, folder.ID) {
		f := f
		if s.folderMatchesQuery(&f, query) {
			return true
		}
	}
	for _, r := range s.childRequests(folder.CollectionID, folder.ID) {
		r := r
		if requestMatchesQuery(&r, query) {
			return true
		}
	}
	return false
}

func requestMatchesQuery(req *domainreq.Request, query string) bool {
	if req == nil {
		return false
	}
	return matchesQuery(req.Name, query) || matchesQuery(req.Method, query) || matchesQuery(req.URL, query)
}

func matchesQuery(value, query string) bool {
	return strings.Contains(strings.ToLower(value), query)
}

// applyEmptyMenu runs a choice from the empty-area menu, creating the item
// inside the first collection when one exists.
func (s *shellState) applyEmptyMenu(services *app.Services, chosen string) {
	if s.workspaceID == "" {
		return
	}
	if chosen == menuNewCollection || len(s.collections) == 0 {
		s.createCollection(services)
		return
	}
	s.applyFolderMenu(services, s.collections[0].ID, "", chosen)
}

// applyFolderMenu runs the action chosen from a folder's context menu.
func (s *shellState) applyFolderMenu(services *app.Services, colID, folderID, chosen string) {
	switch chosen {
	case menuHTTP:
		s.createRequest(services, colID, folderID, domainreq.KindHTTP, "")
	case menuGraphQL:
		s.createRequest(services, colID, folderID, domainreq.KindGraphQL, "")
	case menuGRPC:
		s.createRequest(services, colID, folderID, domainreq.KindGRPC, "")
	case menuWebSocket:
		s.createRequest(services, colID, folderID, domainreq.KindWebSocket, "")
	case menuFolder:
		s.createFolder(services, colID, folderID)
	case menuDelete:
		if folderID == "" {
			s.deleteCollection(services, colID)
		} else {
			s.deleteFolder(services, folderID)
		}
	case menuDuplicate:
		if folderID == "" {
			s.duplicateCollection(services, colID)
		} else {
			s.duplicateFolder(services, folderID)
		}
	case menuFolderSettings:
		s.openCollectionSettings(colID)
	}
}

// kindLabel maps a Kind to its display label.
func kindLabel(k domainreq.Kind) string {
	switch k {
	case domainreq.KindGraphQL:
		return "GraphQL"
	case domainreq.KindGRPC:
		return "gRPC"
	case domainreq.KindWebSocket:
		return "WebSocket"
	default:
		return "HTTP"
	}
}
