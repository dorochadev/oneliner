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

	b.WriteString(fmt.Sprintf("You are a %s shell expert for %s systems.\n", shell, ctx.OS))
	b.WriteString(fmt.Sprintf("Generate a single, safe one-liner for %s that does the following:\n\n", shell))
	b.WriteString(fmt.Sprintf("%s\n\n", ctx.Query))
	b.WriteString("Context:\n")
	b.WriteString(fmt.Sprintf("- OS: %s\n", ctx.OS))
	b.WriteString(fmt.Sprintf("- Current directory: %s\n", ctx.CWD))
	b.WriteString(fmt.Sprintf("- Username: %s\n", ctx.Username))
	b.WriteString(fmt.Sprintf("- Shell: %s\n\n", ctx.Shell))

	if explain {
		b.WriteString("Output the command on the first line, then add 'EXPLANATION:' on a new line, followed by a brief explanation of what the command does.\n")
	} else {
		b.WriteString("Output only the command, no explanation or additional text.\n")
	}

	if cfg.SafeExecution {
		b.WriteString("Ensure the command is safe and does not perform destructive operations without explicit confirmation.\n")
	}

	return b.String()
}
