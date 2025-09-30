package prompt

import (
	"fmt"
	"strings"

	"github.com/dorochadev/oneliner/config"
)

type Context struct {
	Query    string
	OS       string
	CWD      string
	Username string
	Shell    string
}

func Build(ctx Context, cfg *config.Config, explain bool) string {
	shell := cfg.DefaultShell
	if shell == "" {
		shell = "bash"
	}

	var b strings.Builder

	// Instruction header
	b.WriteString(fmt.Sprintf("You are an expert in %s on %s systems.\n", shell, ctx.OS))
	b.WriteString(fmt.Sprintf("Write a single, safe one-liner in %s to:\n", shell))
	b.WriteString(fmt.Sprintf("%s\n\n", ctx.Query))

	// Minimal context
	b.WriteString("System:\n")
	b.WriteString(fmt.Sprintf("  OS: %s\n", ctx.OS))
	b.WriteString(fmt.Sprintf("  Dir: %s\n", ctx.CWD))
	b.WriteString(fmt.Sprintf("  User: %s\n", ctx.Username))
	b.WriteString(fmt.Sprintf("  Shell: %s\n", ctx.Shell))

	// Shell hints
	switch strings.ToLower(shell) {
	case "fish":
		b.WriteString("Use idiomatic fish syntax only.\n")
	case "powershell":
		b.WriteString("Use idiomatic PowerShell. No bash.\n")
	}

	// Explanation toggle
	if explain {
		b.WriteString("Respond with the command first, then add 'EXPLANATION:' on a new line with a brief explanation.\n")
	} else {
		b.WriteString("Output only the command, nothing else.\n")
	}

	// Safety
	if cfg.SafeExecution {
		b.WriteString("Avoid destructive or dangerous operations.\n")
	}

	return b.String()
}
