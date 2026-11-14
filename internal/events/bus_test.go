package events

import (
	"context"
	"testing"
)

func TestInMemoryBus(t *testing.T) {
	b := NewBus()
	var got []Type
	unsub := b.Subscribe(WorkspaceCreated, func(_ context.Context, e Event) { got = append(got, e.Type) })
	if err := b.Publish(context.Background(), New(WorkspaceCreated, "ws1", "ws1", nil)); err != nil {
		t.Fatal(err)
	}
	unsub()
	_ = b.Publish(context.Background(), New(WorkspaceCreated, "ws1", "ws1", nil))
	if len(got) != 1 || got[0] != WorkspaceCreated {
		t.Fatalf("got %#v", got)
	}
}
