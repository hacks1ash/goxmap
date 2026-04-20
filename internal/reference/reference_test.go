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

func TestParse_FullPath(t *testing.T) {
	got, err := Parse("github.com/org/repo/internal/models.User")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Kind != KindFullPath {
		t.Fatalf("kind = %v", got.Kind)
	}
	if got.PackagePath != "github.com/org/repo/internal/models" {
		t.Fatalf("pkg = %q", got.PackagePath)
	}
	if got.TypeName != "User" {
		t.Fatalf("type = %q", got.TypeName)
	}
}

func TestParse_ModuleRelative(t *testing.T) {
	got, err := Parse("internal/models/request.RequestStruct")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Kind != KindModuleRelative {
		t.Fatalf("kind = %v", got.Kind)
	}
	if got.PackagePath != "internal/models/request" {
		t.Fatalf("pkg = %q", got.PackagePath)
	}
	if got.TypeName != "RequestStruct" {
		t.Fatalf("type = %q", got.TypeName)
	}
}

func TestParse_GopkgIn(t *testing.T) {
	got, err := Parse("gopkg.in/yaml.v3.Node")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Kind != KindFullPath {
		t.Fatalf("kind = %v", got.Kind)
	}
	if got.PackagePath != "gopkg.in/yaml.v3" {
		t.Fatalf("pkg = %q", got.PackagePath)
	}
}

func TestParse_Errors(t *testing.T) {
	cases := []string{
		"models/User",      // missing dot before type
		".User",            // empty package
		"internal/models.", // empty type
	}
	for _, in := range cases {
		if _, err := Parse(in); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}
}

func TestImportPath(t *testing.T) {
	cases := []struct {
		name    string
		ref     Ref
		modPath string
		wantPkg string // empty means "current package" sentinel
	}{
		{"bare", Ref{Kind: KindBare, TypeName: "User"}, "example.com/m", ""},
		{"full", Ref{Kind: KindFullPath, PackagePath: "github.com/x/y", TypeName: "T"}, "example.com/m", "github.com/x/y"},
		{"mod-rel", Ref{Kind: KindModuleRelative, PackagePath: "internal/foo", TypeName: "T"}, "example.com/m", "example.com/m/internal/foo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ref.ImportPath(tc.modPath)
			if got != tc.wantPkg {
				t.Errorf("ImportPath = %q want %q", got, tc.wantPkg)
			}
		})
	}
}
