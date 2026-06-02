package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const MaxConcurrentAgents = 3

var protectedParts = []string{".env", "pcap", "exports", "backups", "volumes", ".duku-data", "data/"}

type Task struct {
	ID                 string   `json:"id"`
	Objective          string   `json:"objective"`
	AllowedPaths       []string `json:"allowedPaths"`
	AcceptanceCriteria []string `json:"acceptanceCriteria"`
	VerifyCommands     []string `json:"verifyCommands"`
}

func LoadTask(path string) (Task, error) {
	var task Task
	data, err := os.ReadFile(path)
	if err != nil {
		return task, err
	}
	if err := json.Unmarshal(data, &task); err != nil {
		return task, err
	}
	if task.ID == "" || task.Objective == "" {
		return task, errors.New("task id and objective are required")
	}
	for _, path := range task.AllowedPaths {
		if err := ValidateAgentPath(path); err != nil {
			return task, err
		}
	}
	return task, nil
}

func ValidateAgentPath(path string) error {
	clean := strings.ToLower(filepath.ToSlash(filepath.Clean(path)))
	for _, part := range protectedParts {
		if strings.Contains(clean, part) {
			return fmt.Errorf("protected operational path is not agent-readable: %s", path)
		}
	}
	return nil
}

func CreateWorktree(repo, root string, task Task) (string, error) {
	worktree := filepath.Join(root, task.ID)
	branch := "agent/" + task.ID
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	cmd := exec.Command("git", "-C", repo, "worktree", "add", "-b", branch, worktree, "HEAD")
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("create worktree: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return worktree, nil
}

func Prompt(task Task, phase string) string {
	return fmt.Sprintf("Duku Net Lab supervised task phase: %s\nTask: %s\nObjective: %s\nAllowed paths: %s\nAcceptance criteria:\n- %s\nDo not read operational data, .env files, PCAP files, exports, backups, or volumes.", phase, task.ID, task.Objective, strings.Join(task.AllowedPaths, ", "), strings.Join(task.AcceptanceCriteria, "\n- "))
}
