package xdg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookupFindsFirstXDGDataMatch(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	system := filepath.Join(root, "system")
	writeDesktop(t, filepath.Join(home, "applications", "org.example.App.desktop"), "Name=Home\nExec=app\n")
	writeDesktop(t, filepath.Join(system, "applications", "org.example.App.desktop"), "Name=System\nExec=app\n")

	t.Setenv("XDG_DATA_HOME", home)
	t.Setenv("XDG_DATA_DIRS", system)

	result, err := Lookup("org.example.App", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2", len(result.Candidates))
	}
	if got := result.Candidates[0].Entry.Keys["Name"]; got != "Home" {
		t.Fatalf("selected Name = %q, want Home", got)
	}
	if !result.Candidates[0].Selected {
		t.Fatal("first candidate was not selected")
	}
	if got := result.Candidates[1].Problems[0]; got != "shadowed by higher-priority desktop file" {
		t.Fatalf("second candidate first problem = %q", got)
	}
}

func TestLookupMapsSubdirectoryToDesktopID(t *testing.T) {
	root := t.TempDir()
	writeDesktop(t, filepath.Join(root, "applications", "kde", "org.example.App.desktop"), "Name=KDE\nExec=app\n")

	t.Setenv("XDG_DATA_HOME", root)
	t.Setenv("XDG_DATA_DIRS", filepath.Join(root, "empty"))

	result, err := Lookup("kde-org.example.App.desktop", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(result.Candidates))
	}
}

func TestLookupMatchesTrimmedReverseDomainID(t *testing.T) {
	root := t.TempDir()
	writeDesktop(t, filepath.Join(root, "applications", "org.pwmt.zathura.desktop"), "Name=Zathura\nExec=zathura\n")

	t.Setenv("XDG_DATA_HOME", root)
	t.Setenv("XDG_DATA_DIRS", filepath.Join(root, "empty"))

	result, err := Lookup("zathura", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if result.MatchMode != "trimmed" {
		t.Fatalf("match mode = %q, want trimmed", result.MatchMode)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(result.Candidates))
	}
	if got := result.Candidates[0].Entry.ID; got != "org.pwmt.zathura.desktop" {
		t.Fatalf("selected ID = %q, want org.pwmt.zathura.desktop", got)
	}
}

func TestLookupPrefersExactMatchOverTrimmedMatch(t *testing.T) {
	root := t.TempDir()
	writeDesktop(t, filepath.Join(root, "applications", "zathura.desktop"), "Name=Plain\nExec=zathura\n")
	writeDesktop(t, filepath.Join(root, "applications", "org.pwmt.zathura.desktop"), "Name=Prefixed\nExec=zathura\n")

	t.Setenv("XDG_DATA_HOME", root)
	t.Setenv("XDG_DATA_DIRS", filepath.Join(root, "empty"))

	result, err := Lookup("zathura", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if result.MatchMode != "exact" {
		t.Fatalf("match mode = %q, want exact", result.MatchMode)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(result.Candidates))
	}
	if got := result.Candidates[0].Entry.ID; got != "zathura.desktop" {
		t.Fatalf("selected ID = %q, want zathura.desktop", got)
	}
}

func TestLookupReportsVisibilityProblems(t *testing.T) {
	root := t.TempDir()
	writeDesktop(t, filepath.Join(root, "applications", "org.example.Hidden.desktop"), "Name=Hidden\nHidden=true\nOnlyShowIn=GNOME;\nTryExec=/definitely/missing\n")

	t.Setenv("XDG_DATA_HOME", root)
	t.Setenv("XDG_DATA_DIRS", filepath.Join(root, "empty"))

	result, err := Lookup("org.example.Hidden", Options{Desktop: "KDE"})
	if err != nil {
		t.Fatal(err)
	}
	got := result.Candidates[0].Problems
	want := []string{"Hidden=true", "OnlyShowIn does not include current desktop", "TryExec not found or not executable"}
	if len(got) != len(want) {
		t.Fatalf("problems = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("problem %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNormalizeQueryRejectsNonDesktopPath(t *testing.T) {
	if _, err := normalizeQuery("/tmp/example.txt"); err == nil {
		t.Fatal("expected error")
	}
}

func writeDesktop(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "[Desktop Entry]\nType=Application\n" + body
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
