package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/duku/net-lab/internal/harness"
)

func main() {
	taskPath := flag.String("task", "", "path to a JSON task manifest")
	phase := flag.String("phase", "plan", "plan, implement, review, or verify")
	createWorktree := flag.Bool("create-worktree", false, "create a dedicated git worktree")
	repo := flag.String("repo", ".", "repository root")
	root := flag.String("worktrees", "worktrees", "worktree parent directory")
	flag.Parse()
	if *taskPath == "" {
		fmt.Fprintln(os.Stderr, "-task is required")
		os.Exit(2)
	}
	task, err := harness.LoadTask(*taskPath)
	if err != nil {
		fatal(err)
	}
	if *createWorktree {
		path, err := harness.CreateWorktree(*repo, *root, task)
		if err != nil {
			fatal(err)
		}
		fmt.Println(path)
		return
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"task": task.ID, "phase": *phase, "maxConcurrentAgents": harness.MaxConcurrentAgents, "prompt": harness.Prompt(task, *phase), "next": "Review the prompt, then invoke codex-cli in the isolated worktree."})
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
