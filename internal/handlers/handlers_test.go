package handlers

import (
	"reflect"
	"testing"
)

func TestSplitTags(t *testing.T) {
	got := splitTags("go, cli template\nhttp")
	want := []string{"go", "cli", "template", "http"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitTags() = %#v, want %#v", got, want)
	}
}
