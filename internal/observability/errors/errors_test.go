package errors

import (
	stderrors "errors"
	"testing"
)

func TestStructuredError(t *testing.T) {
	base := stderrors.New("boom")
	err := WithField(Wrap(CategoryStorage, base, "save failed"), "entity", "workspace")
	if !IsCategory(err, CategoryStorage) {
		t.Fatalf("expected storage category, got %s", GetCategory(err))
	}
	if Fields(err)["entity"] != "workspace" {
		t.Fatalf("expected entity field, got %#v", Fields(err))
	}
	if !stderrors.Is(err, base) {
		t.Fatal("expected unwrap to base")
	}
	if err.Error() == "" {
		t.Fatal("expected error string")
	}
}
