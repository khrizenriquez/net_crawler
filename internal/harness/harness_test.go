package harness

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestProtectedOperationalPathsAreBlocked(t *testing.T) {
	for _, path := range []string{".env", "data/staging/a.pcap", "backups/db.sql", "exports/report.csv", "volumes/postgres"} {
		if err := ValidateAgentPath(path); err == nil {
			t.Fatalf("expected %q to be blocked", path)
		}
	}
	if err := ValidateAgentPath("internal/analyzer/redact.go"); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrencyLimitIsThree(t *testing.T) {
	if MaxConcurrentAgents != 3 {
		t.Fatalf("got %d", MaxConcurrentAgents)
	}
}

func TestLoadTaskAndPrompt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task.json")
	data := `{"id":"dashboard-test","objective":"Cover dashboard helpers","allowedPaths":["dashboard/src"],"acceptanceCriteria":["tests pass"],"verifyCommands":["npm test"]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	task, err := LoadTask(path)
	if err != nil {
		t.Fatal(err)
	}
	prompt := Prompt(task, "verify")
	for _, wanted := range []string{"verify", "dashboard-test", "Cover dashboard helpers", "dashboard/src", "tests pass", ".env"} {
		if !strings.Contains(prompt, wanted) {
			t.Fatalf("prompt missing %q: %s", wanted, prompt)
		}
	}
}

func TestLoadTaskRejectsMalformedMissingAndProtectedTasks(t *testing.T) {
	for name, data := range map[string]string{
		"malformed": `{`,
		"missing":   `{"id":"empty"}`,
		"protected": `{"id":"unsafe","objective":"read secrets","allowedPaths":[".env"]}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "task.json")
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadTask(path); err == nil {
				t.Fatal("expected task validation error")
			}
		})
	}
}

func TestCreateWorktreeCreatesDedicatedBranch(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "tests@example.test")
	runGit(t, repo, "config", "user.name", "Duku Tests")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "fixture")
	root := filepath.Join(t.TempDir(), "worktrees")
	worktree, err := CreateWorktree(repo, root, Task{ID: "coverage"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(worktree, "README.md")); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "-C", worktree, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(output)) != "agent/coverage" {
		t.Fatalf("branch=%q err=%v", output, err)
	}
}

func runGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
