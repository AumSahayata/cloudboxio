package internal

import (
	"runtime"
	"testing"

	"github.com/AumSahayata/cloudboxio/tests"
)

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "report.pdf", "report.pdf"},
		{"unix traversal", "../../etc/passwd", "passwd"},
		{"windows traversal", `..\..\Windows\system32\cfg`, "cfg"},
		{"leading dots", "...hidden", "hidden"},
		{"absolute unix", "/var/log/syslog", "syslog"},
		{"control chars", "bad\x00\x1fname.txt", "badname.txt"},
		{"only dots", "..", "file"},
		{"empty", "", "file"},
		{"spaces trimmed", "  spaced.txt  ", "spaced.txt"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SanitizeFilename(c.in); got != c.want {
				t.Fatalf("SanitizeFilename(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestSanitizeFilename_NoPathSeparators(t *testing.T) {
	got := SanitizeFilename("a/b/c/d.txt")
	if got != "d.txt" {
		t.Fatalf("expected basename only, got %q", got)
	}
}

func TestCleanParam(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"plain id", "12", "12", false},
		{"encoded space", "my%20file.txt", "my file.txt", false},
		{"traversal", "../../etc/passwd", "", true},
		{"encoded traversal", "..%2F..%2Fpasswd", "", true},
	}

	// What counts as an absolute path differs per OS.
	absolute := "/etc/passwd"
	if runtime.GOOS == "windows" {
		absolute = `C:\Windows\system32`
	}
	cases = append(cases, struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{"absolute path", absolute, "", true})

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := CleanParam(c.in)
			if c.wantErr {
				if err == nil {
					t.Fatalf("CleanParam(%q) expected an error, got %q", c.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("CleanParam(%q) returned error: %v", c.in, err)
			}
			if got != c.want {
				t.Fatalf("CleanParam(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestResolveFileNameConflict(t *testing.T) {
	db := tests.SetupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO metadata (user_id, filename, path, is_public) VALUES (?, ?, ?, ?)`,
		"user-1", "report.pdf", "uploads/user-1/report.pdf", false)
	if err != nil {
		t.Fatalf("failed to seed metadata: %v", err)
	}

	// Taken name gets a counter appended.
	name, err := ResolveFileNameConflict("user-1", "report.pdf", false, db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "report(1).pdf" {
		t.Fatalf("got %q, want report(1).pdf", name)
	}

	// Free name is returned unchanged.
	name, err = ResolveFileNameConflict("user-1", "notes.txt", false, db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "notes.txt" {
		t.Fatalf("got %q, want notes.txt", name)
	}

	// The same name is free for a different user.
	name, err = ResolveFileNameConflict("user-2", "report.pdf", false, db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "report.pdf" {
		t.Fatalf("got %q, want report.pdf", name)
	}
}

func TestIsAdminSetupAndChangeSetting(t *testing.T) {
	db := tests.SetupTestDB(t)
	defer db.Close()

	tests.SetAdminSetupFlag(db, false)
	if IsAdminSetup(db) {
		t.Fatal("expected admin setup to be incomplete")
	}

	if err := ChangeSetting("admin_setup_done", "true", db); err != nil {
		t.Fatalf("ChangeSetting failed: %v", err)
	}
	if !IsAdminSetup(db) {
		t.Fatal("expected admin setup to be complete after the update")
	}
}

func TestGetUsernameByID(t *testing.T) {
	db := tests.SetupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO users (id, username, password, is_admin) VALUES (?, ?, ?, ?)`,
		"user-1", "alice", "hash", false)
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	username, err := GetUsernameByID("user-1", db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "alice" {
		t.Fatalf("got %q, want alice", username)
	}

	if _, err := GetUsernameByID("missing", db); err == nil {
		t.Fatal("expected an error for a missing user")
	}
}
