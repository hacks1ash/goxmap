package reference

import "testing"

func TestParse_Bare(t *testing.T) {
	got, err := Parse("User")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Kind != KindBare || got.PackagePath != "" || got.TypeName != "User" {
		t.Fatalf("got %+v", got)
	}
}

func TestParse_Empty(t *testing.T) {
	if _, err := Parse(""); err == nil {
		t.Fatal("expected error for empty input")
	}
}
