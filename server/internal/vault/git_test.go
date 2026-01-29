package vault

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupGitTestEnv creates a test environment with a bare "remote" repo and a "local" clone.
// Returns the Git instance (pointed at local), path to local, path to remote, and cleanup function.
func setupGitTestEnv(t *testing.T) (*Git, string, string) {
	t.Helper()

	// Create temp directory for test repos
	testDir := t.TempDir()

	// Create bare "remote" repository
	remoteDir := filepath.Join(testDir, "remote.git")
	if err := os.MkdirAll(remoteDir, 0755); err != nil {
		t.Fatalf("Failed to create remote dir: %v", err)
	}

	// Initialize bare repo
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init bare repo: %v\n%s", err, out)
	}

	// Create local directory
	localDir := filepath.Join(testDir, "local")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("Failed to create local dir: %v", err)
	}

	// Initialize local repo
	runGitCmd(t, localDir, "init")
	runGitCmd(t, localDir, "config", "user.email", "test@example.com")
	runGitCmd(t, localDir, "config", "user.name", "Test User")

	// Add remote
	runGitCmd(t, localDir, "remote", "add", "origin", remoteDir)

	// Create initial commit
	initialFile := filepath.Join(localDir, "README.md")
	if err := os.WriteFile(initialFile, []byte("# Test Repo\n"), 0644); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}
	runGitCmd(t, localDir, "add", ".")
	runGitCmd(t, localDir, "commit", "-m", "Initial commit")

	// Push to remote
	runGitCmd(t, localDir, "push", "-u", "origin", "master")

	return NewGit(localDir), localDir, remoteDir
}

// runGitCmd runs a git command and fails the test if it fails.
func runGitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// runGitCmdIgnoreError runs a git command and returns the output, ignoring errors.
func runGitCmdIgnoreError(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	return strings.TrimSpace(string(out))
}

// cloneRemote creates another clone of the remote, useful for simulating another user.
func cloneRemote(t *testing.T, remoteDir, cloneDir string) {
	t.Helper()
	cmd := exec.Command("git", "clone", remoteDir, cloneDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to clone remote: %v\n%s", err, out)
	}
	runGitCmd(t, cloneDir, "config", "user.email", "other@example.com")
	runGitCmd(t, cloneDir, "config", "user.name", "Other User")
}

// TestGit_Pull_Success tests a successful pull with no conflicts.
func TestGit_Pull_Success(t *testing.T) {
	git, localDir, remoteDir := setupGitTestEnv(t)

	// Create another clone to simulate remote changes
	testDir := filepath.Dir(localDir)
	otherDir := filepath.Join(testDir, "other")
	cloneRemote(t, remoteDir, otherDir)

	// Make changes in "other" clone and push
	otherFile := filepath.Join(otherDir, "other.md")
	if err := os.WriteFile(otherFile, []byte("# From Other\n"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	runGitCmd(t, otherDir, "add", ".")
	runGitCmd(t, otherDir, "commit", "-m", "Add other file")
	runGitCmd(t, otherDir, "push")

	// Pull in local - should succeed
	err := git.Pull()
	if err != nil {
		t.Fatalf("Pull() error = %v", err)
	}

	// Verify the file exists in local
	localOtherFile := filepath.Join(localDir, "other.md")
	if _, err := os.Stat(localOtherFile); os.IsNotExist(err) {
		t.Error("Pull() did not bring in remote changes")
	}
}

// TestGit_Pull_NoRemoteChanges tests pull when remote has no new changes.
func TestGit_Pull_NoRemoteChanges(t *testing.T) {
	git, _, _ := setupGitTestEnv(t)

	// Pull when already up to date
	err := git.Pull()
	if err != nil {
		t.Fatalf("Pull() with no changes error = %v", err)
	}
}

// TestGit_Pull_ConflictResolution tests pull with conflicts (remote wins).
func TestGit_Pull_ConflictResolution(t *testing.T) {
	git, localDir, remoteDir := setupGitTestEnv(t)

	// Create another clone to simulate conflicting changes
	testDir := filepath.Dir(localDir)
	otherDir := filepath.Join(testDir, "other")
	cloneRemote(t, remoteDir, otherDir)

	// Both edit the same file
	testFile := "conflict.md"

	// Remote (other) makes a change first and pushes
	otherFile := filepath.Join(otherDir, testFile)
	if err := os.WriteFile(otherFile, []byte("Remote content\n"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	runGitCmd(t, otherDir, "add", ".")
	runGitCmd(t, otherDir, "commit", "-m", "Remote change")
	runGitCmd(t, otherDir, "push")

	// Local makes a conflicting change (uncommitted)
	localFile := filepath.Join(localDir, testFile)
	if err := os.WriteFile(localFile, []byte("Local content\n"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	runGitCmd(t, localDir, "add", ".")
	runGitCmd(t, localDir, "commit", "-m", "Local change")

	// Pull should resolve conflict with remote winning
	err := git.Pull()
	if err != nil {
		t.Fatalf("Pull() with conflict error = %v", err)
	}

	// Verify remote content won (theirs strategy)
	content, err := os.ReadFile(localFile)
	if err != nil {
		t.Fatalf("Failed to read file after pull: %v", err)
	}
	if string(content) != "Remote content\n" {
		t.Errorf("After Pull() with conflict, content = %q, want %q", string(content), "Remote content\n")
	}
}

// TestGit_Pull_FallbackToFetchReset tests that pull falls back to fetch+reset on failure.
func TestGit_Pull_FallbackToFetchReset(t *testing.T) {
	git, localDir, remoteDir := setupGitTestEnv(t)

	// Create another clone and push changes
	testDir := filepath.Dir(localDir)
	otherDir := filepath.Join(testDir, "other")
	cloneRemote(t, remoteDir, otherDir)

	// Push changes from other
	otherFile := filepath.Join(otherDir, "other.md")
	if err := os.WriteFile(otherFile, []byte("Remote content\n"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	runGitCmd(t, otherDir, "add", ".")
	runGitCmd(t, otherDir, "commit", "-m", "Other commit")
	runGitCmd(t, otherDir, "push")

	// Create local uncommitted changes that would cause merge issues
	localFile := filepath.Join(localDir, "local-only.md")
	if err := os.WriteFile(localFile, []byte("Local uncommitted\n"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	// Don't commit - leave dirty state

	// Also rewrite history to force a fetch+reset scenario
	// This simulates a diverged branch that can't be merged normally
	divergedFile := filepath.Join(localDir, "diverged.md")
	if err := os.WriteFile(divergedFile, []byte("Diverged content\n"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	runGitCmd(t, localDir, "add", ".")
	runGitCmd(t, localDir, "commit", "-m", "Diverged commit")

	// Force amend to rewrite history (creates divergence)
	amendFile := filepath.Join(localDir, "amended.md")
	if err := os.WriteFile(amendFile, []byte("Amended content\n"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	runGitCmd(t, localDir, "add", ".")
	runGitCmd(t, localDir, "commit", "--amend", "-m", "Amended diverged commit")

	// Pull should eventually succeed via fallback
	err := git.Pull()
	if err != nil {
		t.Fatalf("Pull() with fallback error = %v", err)
	}

	// Verify remote content is present
	remoteFile := filepath.Join(localDir, "other.md")
	if _, err := os.Stat(remoteFile); os.IsNotExist(err) {
		t.Error("After fallback Pull(), remote changes not present")
	}
}

// TestGit_CommitAndPush_Success tests successful commit and push.
func TestGit_CommitAndPush_Success(t *testing.T) {
	git, localDir, remoteDir := setupGitTestEnv(t)

	// Create a new file
	newFile := filepath.Join(localDir, "new.md")
	if err := os.WriteFile(newFile, []byte("# New File\n"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Commit and push
	err := git.CommitAndPush("Add new file")
	if err != nil {
		t.Fatalf("CommitAndPush() error = %v", err)
	}

	// Verify changes were pushed by cloning fresh and checking
	testDir := filepath.Dir(localDir)
	verifyDir := filepath.Join(testDir, "verify")
	cloneRemote(t, remoteDir, verifyDir)

	verifyFile := filepath.Join(verifyDir, "new.md")
	if _, err := os.Stat(verifyFile); os.IsNotExist(err) {
		t.Error("CommitAndPush() did not push changes to remote")
	}
}

// TestGit_CommitAndPush_NoChanges tests commit when there are no changes.
func TestGit_CommitAndPush_NoChanges(t *testing.T) {
	git, _, _ := setupGitTestEnv(t)

	// Commit with no changes - should succeed without error
	err := git.CommitAndPush("Empty commit attempt")
	if err != nil {
		t.Fatalf("CommitAndPush() with no changes error = %v", err)
	}
}

// TestGit_CommitAndPush_PushRetry tests push retry after initial failure.
func TestGit_CommitAndPush_PushRetry(t *testing.T) {
	git, localDir, remoteDir := setupGitTestEnv(t)

	// Create another clone and push changes (will cause local push to fail initially)
	testDir := filepath.Dir(localDir)
	otherDir := filepath.Join(testDir, "other")
	cloneRemote(t, remoteDir, otherDir)

	// Other pushes first
	otherFile := filepath.Join(otherDir, "other.md")
	if err := os.WriteFile(otherFile, []byte("# Other\n"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	runGitCmd(t, otherDir, "add", ".")
	runGitCmd(t, otherDir, "commit", "-m", "Other commit")
	runGitCmd(t, otherDir, "push")

	// Now local tries to commit and push - first push will fail, should retry
	localFile := filepath.Join(localDir, "local.md")
	if err := os.WriteFile(localFile, []byte("# Local\n"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	err := git.CommitAndPush("Local commit")
	if err != nil {
		t.Fatalf("CommitAndPush() with retry error = %v", err)
	}

	// Verify both files are in remote
	verifyDir := filepath.Join(testDir, "verify")
	cloneRemote(t, remoteDir, verifyDir)

	for _, file := range []string{"other.md", "local.md"} {
		verifyFile := filepath.Join(verifyDir, file)
		if _, err := os.Stat(verifyFile); os.IsNotExist(err) {
			t.Errorf("After push retry, %s not found in remote", file)
		}
	}
}

// TestGit_CommitAndPush_PushFailureGraceful tests graceful degradation when push fails.
func TestGit_CommitAndPush_PushFailureGraceful(t *testing.T) {
	// Create a setup where push will always fail (no remote configured properly)
	testDir := t.TempDir()
	localDir := filepath.Join(testDir, "local")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("Failed to create local dir: %v", err)
	}

	// Initialize local repo without valid remote
	runGitCmd(t, localDir, "init")
	runGitCmd(t, localDir, "config", "user.email", "test@example.com")
	runGitCmd(t, localDir, "config", "user.name", "Test User")
	runGitCmd(t, localDir, "remote", "add", "origin", "/nonexistent/path")

	// Create initial commit
	initialFile := filepath.Join(localDir, "README.md")
	if err := os.WriteFile(initialFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	runGitCmd(t, localDir, "add", ".")
	runGitCmd(t, localDir, "commit", "-m", "Initial")

	git := NewGit(localDir)

	// Create a new file
	newFile := filepath.Join(localDir, "new.md")
	if err := os.WriteFile(newFile, []byte("# New\n"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// CommitAndPush should NOT return error even when push fails (graceful degradation)
	err := git.CommitAndPush("Add new file")
	if err != nil {
		t.Fatalf("CommitAndPush() should not error on push failure (graceful degradation), got: %v", err)
	}

	// Verify commit was made locally
	log := runGitCmd(t, localDir, "log", "--oneline", "-1")
	if !strings.Contains(log, "Add new file") {
		t.Error("Commit was not made locally despite push failure")
	}
}

// TestGit_Status tests the Status function.
func TestGit_Status(t *testing.T) {
	git, localDir, _ := setupGitTestEnv(t)

	// Clean state
	status, err := git.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status != "" {
		t.Errorf("Status() on clean repo = %q, want empty", status)
	}

	// Add untracked file
	newFile := filepath.Join(localDir, "untracked.md")
	if err := os.WriteFile(newFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	status, err = git.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !strings.Contains(status, "untracked.md") {
		t.Errorf("Status() should show untracked file, got: %q", status)
	}
}

// TestGit_getCurrentBranch tests getting the current branch name.
func TestGit_getCurrentBranch(t *testing.T) {
	git, _, _ := setupGitTestEnv(t)

	branch, err := git.getCurrentBranch()
	if err != nil {
		t.Fatalf("getCurrentBranch() error = %v", err)
	}
	if branch != "master" {
		t.Errorf("getCurrentBranch() = %q, want %q", branch, "master")
	}
}

// TestGit_AbortOngoingOperations tests aborting rebase/merge operations.
func TestGit_AbortOngoingOperations(t *testing.T) {
	git, _, _ := setupGitTestEnv(t)

	// Should not error even when no rebase/merge is in progress
	err := git.abortOngoingOperations()
	if err != nil {
		t.Errorf("abortOngoingOperations() error = %v", err)
	}
}

// TestGit_Pull_AbortsOngoingMerge tests that Pull aborts an ongoing merge before pulling.
func TestGit_Pull_AbortsOngoingMerge(t *testing.T) {
	git, localDir, remoteDir := setupGitTestEnv(t)

	// Create another clone
	testDir := filepath.Dir(localDir)
	otherDir := filepath.Join(testDir, "other")
	cloneRemote(t, remoteDir, otherDir)

	// Create divergent commits that will conflict
	// In other: modify README.md
	otherReadme := filepath.Join(otherDir, "README.md")
	if err := os.WriteFile(otherReadme, []byte("# Other Version\n"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	runGitCmd(t, otherDir, "add", ".")
	runGitCmd(t, otherDir, "commit", "-m", "Other version")
	runGitCmd(t, otherDir, "push")

	// In local: modify same file differently
	localReadme := filepath.Join(localDir, "README.md")
	if err := os.WriteFile(localReadme, []byte("# Local Version\n"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	runGitCmd(t, localDir, "add", ".")
	runGitCmd(t, localDir, "commit", "-m", "Local version")

	// Try to start a merge (will conflict) - ignore the error since we expect conflict
	runGitCmdIgnoreError(localDir, "pull", "--no-rebase")

	// Now Pull should abort the conflicted merge and succeed
	err := git.Pull()
	if err != nil {
		t.Fatalf("Pull() after conflicted merge error = %v", err)
	}

	// Verify we're in a clean state
	status := runGitCmd(t, localDir, "status", "--porcelain")
	if strings.Contains(status, "UU") || strings.Contains(status, "AA") {
		t.Error("After Pull(), repo still has unmerged files")
	}
}

// TestGit_CommitAndPush_MultipleFiles tests committing multiple files at once.
func TestGit_CommitAndPush_MultipleFiles(t *testing.T) {
	git, localDir, remoteDir := setupGitTestEnv(t)

	// Create multiple files
	files := []string{"file1.md", "file2.md", "subdir/file3.md"}
	for _, f := range files {
		fullPath := filepath.Join(localDir, f)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("# "+f+"\n"), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	err := git.CommitAndPush("Add multiple files")
	if err != nil {
		t.Fatalf("CommitAndPush() error = %v", err)
	}

	// Verify all files pushed
	testDir := filepath.Dir(localDir)
	verifyDir := filepath.Join(testDir, "verify")
	cloneRemote(t, remoteDir, verifyDir)

	for _, f := range files {
		verifyFile := filepath.Join(verifyDir, f)
		if _, err := os.Stat(verifyFile); os.IsNotExist(err) {
			t.Errorf("File %s not found in remote after push", f)
		}
	}
}

// TestGit_CommitAndPush_ModifyFile tests committing file modifications.
func TestGit_CommitAndPush_ModifyFile(t *testing.T) {
	git, localDir, remoteDir := setupGitTestEnv(t)

	// Modify existing file
	readmeFile := filepath.Join(localDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# Modified README\n\nUpdated content.\n"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	err := git.CommitAndPush("Update README")
	if err != nil {
		t.Fatalf("CommitAndPush() error = %v", err)
	}

	// Verify modification pushed
	testDir := filepath.Dir(localDir)
	verifyDir := filepath.Join(testDir, "verify")
	cloneRemote(t, remoteDir, verifyDir)

	content, err := os.ReadFile(filepath.Join(verifyDir, "README.md"))
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if !strings.Contains(string(content), "Modified README") {
		t.Errorf("Modification not pushed to remote, content = %q", string(content))
	}
}

// TestGit_CommitAndPush_DeleteFile tests committing file deletions.
func TestGit_CommitAndPush_DeleteFile(t *testing.T) {
	git, localDir, remoteDir := setupGitTestEnv(t)

	// Create and push a file first
	testFile := filepath.Join(localDir, "to-delete.md")
	if err := os.WriteFile(testFile, []byte("# To Delete\n"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	err := git.CommitAndPush("Add file to delete")
	if err != nil {
		t.Fatalf("CommitAndPush() error = %v", err)
	}

	// Delete the file
	if err := os.Remove(testFile); err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	err = git.CommitAndPush("Delete file")
	if err != nil {
		t.Fatalf("CommitAndPush() delete error = %v", err)
	}

	// Verify deletion pushed
	testDir := filepath.Dir(localDir)
	verifyDir := filepath.Join(testDir, "verify")
	cloneRemote(t, remoteDir, verifyDir)

	verifyFile := filepath.Join(verifyDir, "to-delete.md")
	if _, err := os.Stat(verifyFile); !os.IsNotExist(err) {
		t.Error("Deleted file still exists in remote")
	}
}
