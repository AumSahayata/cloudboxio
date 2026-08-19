package handlers

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/AumSahayata/cloudboxio/internal"
	"github.com/AumSahayata/cloudboxio/tests"
	"github.com/gofiber/fiber/v2"
)

// seedPrivateFile inserts one private file owned by "test-id" and writes it to
// disk, returning its metadata id and on-disk path.
// Every test gets its own directory, Fiber keeps served files open for a while
// and Windows refuses to remove a file that still has an open handle.
func seedPrivateFile(t *testing.T, ctx *TestContext) (int64, string) {
	t.Helper()
	dir, err := os.MkdirTemp("", "cloudboxio-test")
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Cleanup(func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Errorf("failed to remove temp dir: %v", err)
		}
	})

	path := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(path, []byte("top secret"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	res, err := ctx.DB.Exec(
		`INSERT INTO metadata (user_id, filename, size, path, is_public) VALUES (?, ?, ?, ?, ?)`,
		"test-id", "secret.txt", 10, path, false)
	if err != nil {
		t.Fatalf("insert metadata: %v", err)
	}
	id, _ := res.LastInsertId()
	return id, path
}

func fileRoutes(ctx *TestContext) {
	handler := NewFileHandler(ctx.DB)
	ctx.App.Use(internal.JWTProtected())
	ctx.App.Get("/file/:fileid", handler.DownloadFile)
	ctx.App.Delete("/file/:fileid", handler.DeleteFile)
}

func doReq(t *testing.T, ctx *TestContext, method, url, token string) int {
	t.Helper()
	req := httptest.NewRequest(method, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := ctx.App.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// A non-owner must not be able to download another user's private file (IDOR).
func TestDownloadPrivateFile_DeniedForNonOwner(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)
	id, _ := seedPrivateFile(t, ctx)
	fileRoutes(ctx)

	attacker, err := internal.GenerateToken("attacker-id", false, 1)
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	if got := doReq(t, ctx, "GET", "/file/"+strconv.FormatInt(id, 10), attacker); got != fiber.StatusNotFound {
		t.Fatalf("expected 404 for non-owner download, got %d", got)
	}
}

// The owner can still download their own private file.
func TestDownloadPrivateFile_AllowedForOwner(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)
	id, _ := seedPrivateFile(t, ctx)
	fileRoutes(ctx)

	if got := doReq(t, ctx, "GET", "/file/"+strconv.FormatInt(id, 10), ctx.Token); got != fiber.StatusOK {
		t.Fatalf("expected 200 for owner download, got %d", got)
	}
}

// A non-owner must not be able to delete another user's private file, and the
// file must remain on disk (regression: os.Remove used to run before authz).
func TestDeletePrivateFile_DeniedAndFileSurvives(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)
	id, path := seedPrivateFile(t, ctx)
	fileRoutes(ctx)

	attacker, err := internal.GenerateToken("attacker-id", false, 1)
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	if got := doReq(t, ctx, "DELETE", "/file/"+strconv.FormatInt(id, 10), attacker); got != fiber.StatusForbidden {
		t.Fatalf("expected 403 for non-owner delete, got %d", got)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("file was deleted from disk by a non-owner")
	}
}

// The owner can delete their own file and it is removed from disk.
func TestDeletePrivateFile_AllowedForOwner(t *testing.T) {
	ctx := SetupTestContext(t)
	tests.SetAdminSetupFlag(ctx.DB, true)
	id, path := seedPrivateFile(t, ctx)
	fileRoutes(ctx)

	if got := doReq(t, ctx, "DELETE", "/file/"+strconv.FormatInt(id, 10), ctx.Token); got != fiber.StatusNoContent {
		t.Fatalf("expected 204 for owner delete, got %d", got)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file should have been removed from disk")
	}
}
