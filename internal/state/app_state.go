package state

// AppState holds persistent application-level selections shared across the UI.
// It must not perform persistence or domain logic.
type AppState struct {
	ActiveWorkspaceID   string
	ActiveEnvironmentID string
	ActiveRequestID     string
	ActiveCollectionID  string
	ActiveFolderID      string
}

// NewAppState returns an initialized application state.
func NewAppState() AppState {
	return AppState{}
}
