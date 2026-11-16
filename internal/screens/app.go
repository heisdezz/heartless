package screens

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/app"
	"heartless/internal/config"
	domaincol "heartless/internal/domain/collection"
	domainenv "heartless/internal/domain/environment"
	domainreq "heartless/internal/domain/request"
	domainres "heartless/internal/domain/response"
	domainvar "heartless/internal/domain/variable"
	domainws "heartless/internal/domain/workspace"
	impsvc "heartless/internal/services/importer"
)

// varDraft holds the editable name and value of one variable. Drafts live in
// shellState so typed edits survive across builds until they are saved.
type varDraft struct {
	Name  string
	Value string
}

// requestState is one open request's editor and response. Each request has
// its own, so a pending execution never affects another request's pane.
type requestState struct {
	Method   string
	URL      string
	Body     string
	BodyType string // "json", "text", "graphql", "form", "multipart", "binary", "none"
	Tab      int    // 0 Params, 1 Headers, 2 Body, 3 Auth, 4 Script

	// Script is the post-response script, kept with the request so it is saved
	// alongside the other fields.
	Script string

	// Form rows back the form-data and multipart body types.
	Form []*kv

	Params  []*kv // editable param rows
	Headers []*kv // editable header rows

	running     bool
	cancel      context.CancelFunc
	response    *domainres.ExecutionResult
	error       string
	ResponseTab int
	sentBody    string
	copyNotice  string

	// historyLoaded is set once a previously persisted response has been
	// looked up for this request, so the lookup happens only on first open.
	historyLoaded bool

	// script holds the last post-response script run's output, and whether its
	// variable updates have been written back to the database.
	script    *scriptRun
	scriptGen uint64 // increments per queued run, so stale results are dropped

	// Inline completion state for the script editor: the caret, the suggestions
	// for the word being typed, and which one is selected.
	scriptCaret        int
	scriptDesiredCaret int
	scriptHasCaret     bool
	scriptSuggest      []string
	scriptSuggestSel   int

	// Auth is the request's authentication; AuthToken, AuthUser and AuthPass
	// back the editor fields for the current auth type.
	Auth        domainreq.Auth
	AuthToken   string
	AuthUser    string
	AuthPass    string
	AuthKeyName string
	AuthKeyVal  string
}

// kv is one editable key/value row.
type kv struct {
	Key   string
	Value string
	On    bool
}

// scriptRun is the outcome of the last post-response script run for a request.
type scriptRun struct {
	result  *ScriptResult
	applied bool // the variable updates were written back
	err     string
}

// shellState holds mutable state for the shell UI.
type shellState struct {
	sidebarWidth    float32
	sidebarExpanded bool
	workspaceID     string
	environmentID   string
	router          *ui.Router
	// window is the window the view draws in, for the title bar's controls.
	window *mygo.Window

	workspaces   []domainws.Workspace
	environments []domainenv.Environment
	variables    []domainvar.Variable

	collections []domaincol.Collection
	folders     []domaincol.Folder
	requests    []domainreq.Request
	expanded    map[string]bool // collection/folder IDs whose children show

	loaded bool
	err    string

	showWorkspaceDialog bool
	showCreateWorkspace bool
	newWorkspaceName    string

	showCreateEnv bool
	newEnvName    string

	newVarName  string
	newVarValue string
	varDrafts   map[string]*varDraft

	showImportCollection bool
	importCollectionURL  string
	importCollectionErr  string
	importCollectionBusy bool

	// selectedRequestID and the request editor states live below.
	selectedRequestID string
	renameRequestID   string
	renameValue       string

	// Settings page state: the editor copy, what was saved, and the theme
	// built from the saved settings.
	settings        config.ApplicationSettings
	settingsBase    config.ApplicationSettings
	settingsLoaded  bool
	settingsDirty   bool
	settingsSaved   bool
	settingsSection settingsSection
	themeVariant    int
	themeConfig     *theme.Config
	// themeSaved is the settings themeConfig was built from, so a saved change
	// is rebuilt once.
	themeSaved  config.ApplicationSettings
	searchQuery string

	// requestStates keeps each open request's editor and response, keyed by
	// request ID. They are isolated from one another.
	requestStates map[string]*requestState

	// runner executes post-response scripts off the frame loop. It is created
	// by the view the first time a script needs it.
	runner *scriptRunner

	// requestsWithResponses tracks which requests have a response persisted in
	// history, so the sidebar can tint their names. It is scanned from history
	// once per workspace and updated after each send.
	requestsWithResponses map[string]bool
	responsesScanned      bool
	responsesWorkspaceID  string
}

func (s *shellState) currentWorkspace() (domainws.Workspace, bool) {
	for _, ws := range s.workspaces {
		if ws.ID == s.workspaceID {
			return ws, true
		}
	}
	return domainws.Workspace{}, false
}

func (s *shellState) workspaceEnvironments() []domainenv.Environment {
	out := make([]domainenv.Environment, len(s.environments))
	copy(out, s.environments)
	return out
}

func (s *shellState) environmentByID(id string) (*domainenv.Environment, bool) {
	for i := range s.environments {
		if s.environments[i].ID == id {
			return &s.environments[i], true
		}
	}
	return nil, false
}

func (s *shellState) currentEnvironment() (domainenv.Environment, bool) {
	for _, env := range s.environments {
		if env.ID == s.environmentID {
			return env, true
		}
	}
	return domainenv.Environment{}, false
}

// variablesForEnvironment returns the variables scoped to envID.
func (s *shellState) variablesForEnvironment(envID string) []domainvar.Variable {
	var out []domainvar.Variable
	for _, v := range s.variables {
		if v.EnvironmentID == envID {
			out = append(out, v)
		}
	}
	return out
}

// varDraft returns the draft for v, seeding it from the stored variable the
// first time it is seen.
func (s *shellState) varDraft(v domainvar.Variable) *varDraft {
	if s.varDrafts == nil {
		s.varDrafts = make(map[string]*varDraft)
	}
	d, ok := s.varDrafts[v.ID]
	if !ok {
		d = &varDraft{Name: v.Name, Value: v.Value}
		s.varDrafts[v.ID] = d
	}
	return d
}

// openEnvironment selects envID and opens its detail page.
func (s *shellState) openEnvironment(envID string, services *app.Services) {
	s.setActiveEnvironment(context.Background(), services, envID)
	s.router.Push("/environments/" + envID)
}

func (s *shellState) setActiveEnvironment(ctx context.Context, services *app.Services, envID string) {
	if s.workspaceID == "" {
		s.environmentID = ""
		return
	}
	s.environmentID = envID
	if services != nil {
		if err := services.Workspaces.SetActiveEnvironmentID(ctx, s.workspaceID, envID); err != nil {
			s.err = err.Error()
		}
	}
}

// selectWorkspace makes id the active workspace and restores its persisted active environment.
func (s *shellState) selectWorkspace(id string, services *app.Services) {
	if id == "" {
		return
	}
	s.workspaceID = id
	s.environmentID = ""
	s.selectedRequestID = ""
	s.renameRequestID = ""
	s.collections = nil
	s.folders = nil
	s.requests = nil
	s.environments = nil
	s.variables = nil
	ctx := context.Background()
	if err := s.reload(ctx, services); err != nil {
		s.err = err.Error()
	}
	s.router.Replace("/")
}

func (s *shellState) load(ctx context.Context, services *app.Services) {
	if s.loaded || services == nil {
		return
	}
	s.loaded = true
	s.err = ""
	if err := s.reload(ctx, services); err != nil {
		s.err = err.Error()
	}
}

func (s *shellState) reload(ctx context.Context, services *app.Services) error {
	if services == nil {
		return nil
	}
	wss, err := services.Workspaces.List(ctx, "")
	if err != nil {
		return err
	}
	s.workspaces = s.workspaces[:0]
	for _, ws := range wss {
		if ws != nil {
			s.workspaces = append(s.workspaces, *ws)
		}
	}
	if len(s.workspaces) > 0 {
		found := false
		for _, ws := range s.workspaces {
			if ws.ID == s.workspaceID {
				found = true
				break
			}
		}
		if !found {
			s.workspaceID = s.workspaces[0].ID
		}
	} else {
		s.workspaceID = ""
		s.collections = nil
		s.folders = nil
		s.requests = nil
		s.environments = nil
		s.variables = nil
		return nil
	}
	if err := s.reloadEnvironments(ctx, services); err != nil {
		return err
	}
	return s.reloadTree(ctx, services)
}

func (s *shellState) reloadEnvironments(ctx context.Context, services *app.Services) error {
	if services == nil || s.workspaceID == "" {
		s.environments = nil
		s.environmentID = ""
		s.variables = nil
		return nil
	}
	envs, err := services.Environments.List(ctx, s.workspaceID)
	if err != nil {
		return err
	}
	s.environments = s.environments[:0]
	for _, env := range envs {
		if env != nil {
			s.environments = append(s.environments, *env)
		}
	}

	activeID, err := services.Workspaces.GetActiveEnvironmentID(ctx, s.workspaceID)
	if err != nil {
		return err
	}
	s.environmentID = ""
	if activeID != "" {
		for _, env := range s.environments {
			if env.ID == activeID {
				s.environmentID = activeID
				break
			}
		}
	}
	return s.reloadVariables(ctx, services)
}

func (s *shellState) reloadVariables(ctx context.Context, services *app.Services) error {
	if services == nil || s.workspaceID == "" {
		s.variables = nil
		return nil
	}
	vars, err := services.Variables.List(ctx, s.workspaceID, "")
	if err != nil {
		return err
	}
	s.variables = s.variables[:0]
	for _, v := range vars {
		if v != nil {
			s.variables = append(s.variables, *v)
		}
	}
	return nil
}

// reloadTree reloads the workspace's collections, folders and saved requests.
func (s *shellState) reloadTree(ctx context.Context, services *app.Services) error {
	if services == nil || s.workspaceID == "" {
		s.collections = nil
		s.folders = nil
		s.requests = nil
		return nil
	}
	cols, err := services.Collections.ListCollections(ctx, s.workspaceID)
	if err != nil {
		return err
	}
	s.collections = s.collections[:0]
	for _, col := range cols {
		if col != nil {
			s.collections = append(s.collections, *col)
		}
	}

	s.folders = nil
	s.requests = nil
	for _, col := range s.collections {
		folders, err := services.Collections.ListFolders(ctx, col.ID)
		if err != nil {
			return err
		}
		for _, f := range folders {
			if f != nil {
				s.folders = append(s.folders, *f)
			}
		}
		reqs, err := services.Collections.ListRequests(ctx, s.workspaceID, col.ID, "")
		if err != nil {
			return err
		}
		for _, r := range reqs {
			if r != nil {
				s.requests = append(s.requests, *r)
			}
		}
	}
	if s.expanded == nil {
		s.expanded = make(map[string]bool)
	}
	return nil
}

// isExpanded reports whether a collection or folder's children show. It
// defaults to open so a new tree shows its contents.
func (s *shellState) isExpanded(id string) bool {
	if s.expanded == nil {
		return true
	}
	open, ok := s.expanded[id]
	if !ok {
		return true
	}
	return open
}

// toggleExpanded flips a collection or folder's expanded state.
func (s *shellState) toggleExpanded(id string) {
	if s.expanded == nil {
		s.expanded = make(map[string]bool)
	}
	s.expanded[id] = !s.expanded[id]
}

// setExpanded opens or closes a collection or folder.
func (s *shellState) setExpanded(id string, open bool) {
	if s.expanded == nil {
		s.expanded = make(map[string]bool)
	}
	s.expanded[id] = open
}

// childFolders returns the folders directly under parentID in colID, with an
// empty parentID naming a top-level folder.
func (s *shellState) childFolders(colID, parentID string) []domaincol.Folder {
	var out []domaincol.Folder
	for _, f := range s.folders {
		if f.CollectionID == colID && f.ParentID == parentID {
			out = append(out, f)
		}
	}
	return out
}

// childRequests returns the requests directly in colID's folderID, with an
// empty folderID naming the collection root.
func (s *shellState) childRequests(colID, folderID string) []domainreq.Request {
	var out []domainreq.Request
	for _, r := range s.requests {
		if r.CollectionID == colID && r.FolderID == folderID {
			out = append(out, r)
		}
	}
	return out
}

// createCollection adds a top-level collection with an unused default name.
func (s *shellState) createCollection(services *app.Services) {
	if s.workspaceID == "" || services == nil {
		return
	}
	col := domaincol.New(s.workspaceID, s.nextCollectionName())
	if err := services.Collections.SaveCollection(context.Background(), col); err != nil {
		s.err = err.Error()
		return
	}
	s.setExpanded(col.ID, true)
	if err := s.reloadTree(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// nextCollectionName returns the first unused "Collection N" name.
func (s *shellState) nextCollectionName() string {
	used := map[string]bool{}
	for _, col := range s.collections {
		used[col.Name] = true
	}
	for n := 1; ; n++ {
		name := "Collection " + strconv.Itoa(n)
		if !used[name] {
			return name
		}
	}
}

// createFolder adds a folder under colID, or under parentID within it.
func (s *shellState) createFolder(services *app.Services, colID, parentID string) {
	col, ok := s.collectionByID(colID)
	if !ok || services == nil {
		return
	}
	name := s.nextFolderName(col.ID)
	folder := domaincol.NewFolder(col.ID, parentID, name)
	if err := services.Collections.SaveFolder(context.Background(), folder); err != nil {
		s.err = err.Error()
		return
	}
	s.setExpanded(col.ID, true)
	if parentID != "" {
		s.setExpanded(parentID, true)
	}
	if err := s.reloadTree(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// createRequest adds a request under colID, or under folderID within it.
func (s *shellState) createRequest(services *app.Services, colID, folderID string, kind domainreq.Kind, name string) {
	col, ok := s.collectionByID(colID)
	if !ok || services == nil {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = s.nextRequestName(col.ID, kind)
	}
	req := domainreq.New(s.workspaceID, name, kind)
	req.CollectionID = col.ID
	req.FolderID = folderID
	if kind == domainreq.KindHTTP {
		req.Method = "GET"
	}
	if err := services.Collections.SaveRequest(context.Background(), req); err != nil {
		s.err = err.Error()
		return
	}
	s.setExpanded(col.ID, true)
	if folderID != "" {
		s.setExpanded(folderID, true)
	}
	if err := s.reloadTree(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// deleteCollection removes a collection and everything under it.
func (s *shellState) deleteCollection(services *app.Services, id string) {
	if services == nil {
		return
	}
	for _, r := range s.childRequests(id, "") {
		_ = services.Collections.DeleteRequest(context.Background(), r.ID)
	}
	for _, f := range s.folders {
		if f.CollectionID == id {
			for _, r := range s.childRequests(id, f.ID) {
				_ = services.Collections.DeleteRequest(context.Background(), r.ID)
			}
			_ = services.Collections.DeleteFolder(context.Background(), f.ID)
		}
	}
	if err := services.Collections.DeleteCollection(context.Background(), id); err != nil {
		s.err = err.Error()
		return
	}
	if err := s.reloadTree(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// deleteFolder removes a folder and the requests directly inside it.
func (s *shellState) deleteFolder(services *app.Services, id string) {
	if services == nil {
		return
	}
	folder, ok := s.folderByID(id)
	if !ok {
		return
	}
	for _, r := range s.childRequests(folder.CollectionID, id) {
		_ = services.Collections.DeleteRequest(context.Background(), r.ID)
	}
	if err := services.Collections.DeleteFolder(context.Background(), id); err != nil {
		s.err = err.Error()
		return
	}
	if err := s.reloadTree(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// deleteRequest removes one saved request.
func (s *shellState) deleteRequest(services *app.Services, id string) {
	if services == nil {
		return
	}
	if err := services.Collections.DeleteRequest(context.Background(), id); err != nil {
		s.err = err.Error()
		return
	}
	if err := s.reloadTree(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// renameCollection saves a new collection name.
func (s *shellState) renameCollection(services *app.Services, id, name string) {
	name = strings.TrimSpace(name)
	col, ok := s.collectionByID(id)
	if !ok || services == nil || name == "" {
		return
	}
	col.Name = name
	if err := services.Collections.SaveCollection(context.Background(), &col); err != nil {
		s.err = err.Error()
		return
	}
	if err := s.reloadTree(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// renameFolder saves a new folder name.
func (s *shellState) renameFolder(services *app.Services, id, name string) {
	name = strings.TrimSpace(name)
	folder, ok := s.folderByID(id)
	if !ok || services == nil || name == "" {
		return
	}
	folder.Name = name
	if err := services.Collections.SaveFolder(context.Background(), &folder); err != nil {
		s.err = err.Error()
		return
	}
	if err := s.reloadTree(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// renameRequest saves a new request name.
func (s *shellState) renameRequest(services *app.Services, id, name string) {
	name = strings.TrimSpace(name)
	req, ok := s.requestByID(id)
	if !ok || services == nil || name == "" {
		return
	}
	req.Name = name
	if err := services.Collections.SaveRequest(context.Background(), &req); err != nil {
		s.err = err.Error()
		return
	}
	if err := s.reloadTree(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// setRequestMethod updates a saved request's HTTP method.
func (s *shellState) setRequestMethod(services *app.Services, id, method string) {
	req, ok := s.requestByID(id)
	if !ok || services == nil {
		return
	}
	req.Method = strings.ToUpper(strings.TrimSpace(method))
	if err := services.Collections.SaveRequest(context.Background(), &req); err != nil {
		s.err = err.Error()
		return
	}
	if err := s.reloadTree(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// duplicateRequest copies a saved request into the same folder.
func (s *shellState) duplicateRequest(services *app.Services, id string) {
	req, ok := s.requestByID(id)
	if !ok || services == nil {
		return
	}
	copy := req
	copy.ID = ""
	copy.Name = req.Name + " (copy)"
	copy.CreatedAt = time.Time{}
	if err := services.Collections.SaveRequest(context.Background(), &copy); err != nil {
		s.err = err.Error()
		return
	}
	if err := s.reloadTree(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// duplicateFolder copies a folder and the requests directly inside it.
func (s *shellState) duplicateFolder(services *app.Services, id string) {
	folder, ok := s.folderByID(id)
	if !ok || services == nil {
		return
	}
	copy := folder
	copy.ID = ""
	copy.Name = folder.Name + " (copy)"
	copy.CreatedAt = time.Time{}
	if err := services.Collections.SaveFolder(context.Background(), &copy); err != nil {
		s.err = err.Error()
		return
	}
	for _, r := range s.childRequests(folder.CollectionID, id) {
		rc := r
		rc.ID = ""
		rc.FolderID = copy.ID
		rc.CreatedAt = time.Time{}
		_ = services.Collections.SaveRequest(context.Background(), &rc)
	}
	if err := s.reloadTree(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// duplicateCollection copies a collection with its folders and requests.
func (s *shellState) duplicateCollection(services *app.Services, id string) {
	col, ok := s.collectionByID(id)
	if !ok || services == nil {
		return
	}
	copy := col
	copy.ID = ""
	copy.Name = col.Name + " (copy)"
	copy.CreatedAt = time.Time{}
	if err := services.Collections.SaveCollection(context.Background(), &copy); err != nil {
		s.err = err.Error()
		return
	}
	for _, f := range s.folders {
		if f.CollectionID == id {
			fc := f
			fc.ID = ""
			fc.CollectionID = copy.ID
			fc.CreatedAt = time.Time{}
			_ = services.Collections.SaveFolder(context.Background(), &fc)
		}
	}
	for _, r := range s.requests {
		if r.CollectionID == id {
			rc := r
			rc.ID = ""
			rc.CollectionID = copy.ID
			rc.CreatedAt = time.Time{}
			_ = services.Collections.SaveRequest(context.Background(), &rc)
		}
	}
	if err := s.reloadTree(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// openCollectionSettings opens the settings page for a collection.
func (s *shellState) openCollectionSettings(colID string) {
	s.selectedRequestID = ""
	s.router.Push("/collections/" + colID + "/settings")
}

// beginRename shows an inline name field for a request.
func (s *shellState) beginRename(req domainreq.Request) {
	s.selectedRequestID = req.ID
	s.renameRequestID = req.ID
	s.renameValue = req.Name
}

func (s *shellState) collectionByID(id string) (domaincol.Collection, bool) {
	for _, col := range s.collections {
		if col.ID == id {
			return col, true
		}
	}
	return domaincol.Collection{}, false
}

func (s *shellState) folderByID(id string) (domaincol.Folder, bool) {
	for _, f := range s.folders {
		if f.ID == id {
			return f, true
		}
	}
	return domaincol.Folder{}, false
}

func (s *shellState) requestByID(id string) (domainreq.Request, bool) {
	for _, r := range s.requests {
		if r.ID == id {
			return r, true
		}
	}
	return domainreq.Request{}, false
}

// nextFolderName returns the first unused "Folder N" name in a collection.
func (s *shellState) nextFolderName(colID string) string {
	used := map[string]bool{}
	for _, f := range s.folders {
		if f.CollectionID == colID {
			used[f.Name] = true
		}
	}
	for n := 1; ; n++ {
		name := "Folder " + strconv.Itoa(n)
		if !used[name] {
			return name
		}
	}
}

// nextRequestName returns the first unused "<Kind> N" name in a collection.
func (s *shellState) nextRequestName(colID string, kind domainreq.Kind) string {
	used := map[string]bool{}
	for _, r := range s.requests {
		if r.CollectionID == colID {
			used[r.Name] = true
		}
	}
	base := "New request"
	switch kind {
	case domainreq.KindHTTP:
		base = "HTTP"
	case domainreq.KindGraphQL:
		base = "GraphQL"
	case domainreq.KindGRPC:
		base = "gRPC"
	case domainreq.KindWebSocket:
		base = "WebSocket"
	}
	for n := 1; ; n++ {
		name := base + " " + strconv.Itoa(n)
		if !used[name] {
			return name
		}
	}
}

// App renders the main application view.
func App(c *ui.Context, services *app.Services) {
	root := ui.Column(c).Fill()

	st := ui.Local(root, "shell", func() shellState {
		return shellState{
			sidebarWidth:    260,
			sidebarExpanded: true,
			router:          ui.NewRouter("/"),
		}
	})
	st.load(context.Background(), services)

	// Theme the frame from the saved settings: loading them here, rather than
	// only on the settings page, is what lets the app start with the user's
	// theme and font size. A saved change rebuilds the config, so it applies
	// at once.
	st.ensureSettings(services)
	if st.themeConfig == nil || st.themeSaved.Theme != st.settings.Theme ||
		st.themeSaved.Interface.FontSize != st.settings.Interface.FontSize {
		st.themeConfig = ThemeConfig(st.settings)
		st.themeSaved = st.settings
	}
	// The chosen variant is applied rather than the system's, so the Variant
	// control means what it says.
	applyTheme(c, st.themeConfig, st.settings)
	// Density reaches the helpers deep in the tree through this flag.
	setCompact(st.settings.Interface.CompactMode)
	tk := theme.TokensOf(c)
	root.Background(tk.BgLayout)

	root.Children(func() {
		titleBar(c, tk, st, services)
		ui.Row(c).Grow(1).Basis(0).MinWidth(0).AlignItems(ui.Stretch).Children(func() {
			sidebar(c, st, tk, services)
			mainContent(c, tk, st, services)
		})

		workspaceDialog(c, st, services)
		createWorkspaceModal(c, st, services)
		createEnvironmentModal(c, st, services)
		importCollectionModal(c, st, services)
	})
}

func mainContent(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services) {
	ui.Column(c).Grow(1).Basis(0).MinWidth(0).Background(tk.BgContainer).Children(func() {
		st.router.View(c, func(r *ui.Route) {
			switch {
			case r.Match("/"):
				r.Title("Home")
				workspaceView(c, tk, st, services)
			case r.Match("/settings"):
				r.Title("Settings")
				settingsView(c, tk, st, services)
			case r.Match("/environments"):
				r.Title("Environments")
				environmentsView(c, tk, st, services)
			case r.Match("/environments/{id}"):
				r.Title("Environment")
				environmentDetailView(c, tk, st, services, r.Param("id"))
			case r.Match("/variables"):
				r.Title("Variables")
				variablesView(c, tk, st, services)
			default:
				r.Title("Not found")
				notFoundView(c, tk, r.Path())
			}
		})
	})
}

// workspaceView shows the selected request's panes, or a hint when none is
// selected.
func workspaceView(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services) {
	ui.Column(c).Grow(1).Basis(0).AlignSelf(ui.Stretch).Fill().MinWidth(0).Children(func() {
		if req, ok := st.selectedRequest(); ok {
			contextHeader(c, tk, req.Name)
			requestPane(c, tk, st, services, req)
			return
		}

		title := "No request selected"
		if ws, ok := st.currentWorkspace(); ok {
			title = ws.Name
			if env, ok := st.currentEnvironment(); ok {
				title += " • " + env.Name
			}
		}
		contextHeader(c, tk, title)

		ui.Row(c).Grow(1).Basis(0).AlignItems(ui.Stretch).Children(func() {
			hint := "Select or create a request to start."
			if st.workspaceID == "" {
				hint = "Create or select a workspace to start."
			}
			pane(c, tk, "Request", hint)
			ui.Box(c).Width(1).AlignSelf(ui.Stretch).Background(tk.BorderSecondary)
			pane(c, tk, "Response", "Send a request to see the response.")
		})
	})
}

// selectedRequest returns the request the pane is showing, if any.
func (s *shellState) selectedRequest() (domainreq.Request, bool) {
	if s.selectedRequestID == "" {
		return domainreq.Request{}, false
	}
	return s.requestByID(s.selectedRequestID)
}

// variablesView lists workspace-scoped variables and creates new ones.
func variablesView(c *ui.Context, tk *theme.Tokens, st *shellState, services *app.Services) {
	ui.Column(c).Fill().Children(func() {
		pageHeader(c, tk, "Global variables", "Available to every environment in this workspace.")
		ui.Scroll(c).Grow(1).Basis(0).Children(func() {
			ui.Column(c).Padding(24, 24).Gap(8).Children(func() {
				if st.workspaceID == "" {
					emptyState(c, tk, lucide.Pencil(), "No workspace selected", "Select a workspace first.")
					return
				}
				if len(st.variables) == 0 {
					ui.Text(c, "No workspace variables yet.").TextColor(tk.TextSecondary)
				}
				for _, v := range st.variables {
					if v.EnvironmentID != "" {
						continue
					}
					label := v.Name
					if v.Value != "" {
						label += " = " + v.Value
					}
					ui.Text(c, label)
				}
				ui.Row(c).Gap(8).AlignItems(ui.Center).Margin(12, 0).Children(func() {
					field.Input(c, &st.newVarName, field.InputProps{Placeholder: "Name"}).Grow(3).Basis(0)
					field.Input(c, &st.newVarValue, field.InputProps{Placeholder: "Value"}).Grow(5).Basis(0)
					if action.Button(c, "Create", action.ButtonProps{Variant: action.Primary}).Clicked() {
						st.createVariable(services)
					}
				})
			})
		})
	})
}

func notFoundView(c *ui.Context, tk *theme.Tokens, path string) {
	ui.Column(c).Fill().Children(func() {
		contextHeader(c, tk, "Not found")
		ui.Column(c).Grow(1).Basis(0).Center().Gap(8).Padding(24).Children(func() {
			ui.Text(c, "Page not found").Bold()
			ui.Text(c, path).TextColor(tk.TextSecondary)
		})
	})
}

func contextHeader(c *ui.Context, tk *theme.Tokens, title string) {
	ui.Row(c).
		Height(40).
		Padding(0, 16).
		AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).
		BorderColor(tk.BorderSecondary).
		Background(tk.BgContainer).
		Children(func() {
			ui.Text(c, title).TextColor(tk.TextSecondary)
		})
}

func pane(c *ui.Context, tk *theme.Tokens, title, hint string) {
	ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
		ui.Row(c).
			Height(max(36, textSize(c, 13)+23)).
			Padding(0, 12).
			AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).
			BorderColor(tk.BorderSecondary).
			Children(func() {
				ui.Text(c, title).FontSize(textSize(c, 13)).Bold()
			})

		ui.Column(c).Grow(1).Basis(0).Center().Gap(6).Padding(24).Children(func() {
			ui.Text(c, hint).TextColor(tk.TextSecondary)
		})
	})
}

func (s *shellState) createEnvironment(services *app.Services) {
	name := strings.TrimSpace(s.newEnvName)
	if s.workspaceID == "" || name == "" || services == nil {
		return
	}
	env := domainenv.NewEnvironment(name)

	if err := services.Environments.Save(context.Background(), env, s.workspaceID); err != nil {
		s.err = err.Error()
		return
	}
	s.newEnvName = ""
	s.err = ""
	if err := s.reloadEnvironments(context.Background(), services); err != nil {
		s.err = err.Error()
		return
	}
	s.setActiveEnvironment(context.Background(), services, env.ID)
}

// createEnvironmentVariable adds a variable scoped to envID from the new
// variable inputs.
func (s *shellState) createEnvironmentVariable(services *app.Services, envID string) {
	name := strings.TrimSpace(s.newVarName)
	if s.workspaceID == "" || name == "" || services == nil {
		return
	}
	v := domainvar.NewVariable(name, s.newVarValue, "text", false)

	v.Scope = "environment"
	v.WorkspaceID = s.workspaceID
	v.EnvironmentID = envID
	if err := services.Variables.Save(context.Background(), v.ID, s.workspaceID, envID, v); err != nil {
		s.err = err.Error()
		return
	}
	s.newVarName = ""
	s.newVarValue = ""
	s.err = ""
	if err := s.reloadVariables(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

// saveVariable persists an edited environment variable.
func (s *shellState) saveVariable(services *app.Services, envID, varID, name, value string) {
	if services == nil {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	v := domainvar.NewVariable(name, value, "text", false)
	v.ID = varID
	v.Scope = "environment"
	v.WorkspaceID = s.workspaceID
	v.EnvironmentID = envID
	if err := services.Variables.Save(context.Background(), varID, s.workspaceID, envID, v); err != nil {
		s.err = err.Error()
		return
	}
	s.err = ""
	if err := s.reloadVariables(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

func (s *shellState) createVariable(services *app.Services) {
	name := strings.TrimSpace(s.newVarName)
	if s.workspaceID == "" || name == "" || services == nil {
		return
	}
	v := domainvar.NewVariable(name, s.newVarValue, "text", false)

	v.Scope = "workspace"
	v.WorkspaceID = s.workspaceID
	if err := services.Variables.Save(context.Background(), v.ID, s.workspaceID, "", v); err != nil {
		s.err = err.Error()
		return
	}
	s.newVarName = ""
	s.newVarValue = ""
	s.err = ""
	if err := s.reloadVariables(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}

func (s *shellState) openImportCollection() {
	s.importCollectionURL = ""
	s.importCollectionErr = ""
	s.importCollectionBusy = false
	s.showImportCollection = true
}

func (s *shellState) importCollectionFromURL(services *app.Services, rawURL string) {
	if services == nil || s.workspaceID == "" || rawURL == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		s.importCollectionErr = err.Error()
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.importCollectionErr = err.Error()
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.importCollectionErr = "download failed: " + resp.Status
		return
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		s.importCollectionErr = err.Error()
		return
	}
	s.importCollectionData(services, data)
}

func (s *shellState) importCollectionFromFile(services *app.Services, f field.File) {
	data, err := f.ReadAll()
	if err != nil {
		s.importCollectionErr = err.Error()
		return
	}
	s.importCollectionData(services, data)
}

func (s *shellState) importCollectionData(services *app.Services, data []byte) {
	if services == nil || s.workspaceID == "" {
		return
	}
	s.importCollectionBusy = true
	s.importCollectionErr = ""
	svc := impsvc.New(impsvc.Deps{Workspaces: services.Workspaces, Collections: services.Collections})
	_, err := svc.Import(context.Background(), s.workspaceID, data, impsvc.Options{})
	if err != nil {
		s.importCollectionErr = err.Error()
		s.importCollectionBusy = false
		return
	}
	s.importCollectionBusy = false
	s.showImportCollection = false
	s.importCollectionURL = ""
	if err := s.reloadTree(context.Background(), services); err != nil {
		s.err = err.Error()
	}
}
