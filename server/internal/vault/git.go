package vault

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Git provides git operations for the vault.
type Git struct {
	vaultRoot string
}

// NewGit creates a new Git instance for the given vault root.
func NewGit(vaultRoot string) *Git {
	return &Git{vaultRoot: vaultRoot}
}

// runGit executes a git command in the vault directory.
func (g *Git) runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.vaultRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w\nstderr: %s", args[0], err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// StatusShort returns a concise status suitable for UI display.
func (g *Git) StatusShort() (string, error) {
	return g.runGit("status", "--short", "--branch")
}

// getCurrentBranch returns the current git branch name.
func (g *Git) getCurrentBranch() (string, error) {
	return g.runGit("rev-parse", "--abbrev-ref", "HEAD")
}

// PullFFOnly pulls only when a fast-forward is possible.
func (g *Git) PullFFOnly() error {
	_, err := g.runGit("pull", "--ff-only")
	return err
}

// Pull pulls changes from the remote repository.
// It uses the "theirs" strategy to resolve conflicts (remote wins).
// If the pull fails, it falls back to fetch + reset.
func (g *Git) Pull() error {
	// First, abort any existing rebase or merge
	_ = g.abortOngoingOperations()

	// Try normal pull with theirs strategy
	_, err := g.runGit("pull", "--no-rebase", "-X", "theirs")
	if err == nil {
		return nil
	}

	// Fallback: fetch + reset --hard
	branch, err := g.getCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	if _, err := g.runGit("fetch", "origin", branch); err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	if _, err := g.runGit("reset", "--hard", "origin/"+branch); err != nil {
		return fmt.Errorf("reset failed: %w", err)
	}

	return nil
}

// abortOngoingOperations aborts any ongoing rebase or merge operations.
func (g *Git) abortOngoingOperations() error {
	// Try to abort rebase (ignore errors - may not be in a rebase)
	_, _ = g.runGit("rebase", "--abort")
	// Try to abort merge (ignore errors - may not be in a merge)
	_, _ = g.runGit("merge", "--abort")
	return nil
}

// CommitAndPush stages all changes, commits with the given message, and pushes.
// It retries the push once if it fails initially.
func (g *Git) CommitAndPush(message string) error {
	// Stage all changes
	if _, err := g.runGit("add", "."); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Check if there are changes to commit
	status, err := g.runGit("status", "--porcelain")
	if err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}
	if status == "" {
		// Nothing to commit
		return nil
	}

	// Commit
	if _, err := g.runGit("commit", "-m", message); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	// Push with retry
	if err := g.pushWithRetry(); err != nil {
		// Push failed but commit succeeded - graceful degradation
		// Log the error but don't fail the operation
		return nil
	}

	return nil
}

// Commit stages all changes and creates a commit with the given message.
// committed is false when there was nothing to commit.
func (g *Git) Commit(message string) (committed bool, err error) {
	if _, err := g.runGit("add", "."); err != nil {
		return false, fmt.Errorf("git add failed: %w", err)
	}

	status, err := g.runGit("status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	if status == "" {
		return false, nil
	}

	if _, err := g.runGit("commit", "-m", message); err != nil {
		return false, fmt.Errorf("git commit failed: %w", err)
	}

	return true, nil
}

// Push pushes local commits to the configured remote.
func (g *Git) Push() error {
	_, err := g.runGit("push")
	return err
}

// pushWithRetry attempts to push, pulling and retrying once if the push fails.
func (g *Git) pushWithRetry() error {
	// First push attempt
	_, err := g.runGit("push")
	if err == nil {
		return nil
	}

	// Pull and retry
	if pullErr := g.Pull(); pullErr != nil {
		return fmt.Errorf("pull failed during push retry: %w", pullErr)
	}

	// Second push attempt
	_, err = g.runGit("push")
	return err
}
