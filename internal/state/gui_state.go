package state

// GUIState holds view/navigation and presentation state for the GUI.
// It is transient UI state and must not contain persistence logic.
type GUIState struct {
	SidebarExpanded bool
	SidebarWidth    float32

	CurrentRoute string
	History      []string

	SearchQuery string
}

// NewGUIState returns initialized GUI state with defaults.
func NewGUIState() GUIState {
	return GUIState{
		SidebarExpanded: true,
		SidebarWidth:    260,
		CurrentRoute:    "/",
	}
}

// PushRoute records navigation to path.
func (s *GUIState) PushRoute(path string) {
	if path == "" {
		return
	}
	if s.CurrentRoute != "" {
		s.History = append(s.History, s.CurrentRoute)
	}
	s.CurrentRoute = path
}

// CanGoBack reports whether back navigation is available.
func (s *GUIState) CanGoBack() bool { return len(s.History) > 0 }

// Back navigates to previous route and returns it, or empty if unavailable.
func (s *GUIState) Back() string {
	if len(s.History) == 0 {
		return ""
	}
	last := s.History[len(s.History)-1]
	s.History = s.History[:len(s.History)-1]
	s.CurrentRoute = last
	return last
}
