package prompt

import (
	"errors"
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

const (
	minQueryLength = 5
	minWordCount   = 2
)

// Build constructs the prompt for the LLM. Returns an error if the query is too short or vague.
func Build(ctx Context, cfg *config.Config, explain bool) (string, error) {
	trimmedQuery := strings.TrimSpace(ctx.Query)
	wordCount := len(strings.Fields(trimmedQuery))
	if len(trimmedQuery) < minQueryLength || wordCount < minWordCount {
		return "", errors.New("query is too short or vague; please provide a more detailed request")
	}

	shell := cfg.DefaultShell
	if shell == "" {
		shell = "bash"
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("You are an expert in %s on %s systems.\n", shell, ctx.OS))
	b.WriteString(fmt.Sprintf("Write a single, safe one-liner in %s to:\n", shell))
	b.WriteString(fmt.Sprintf("%s\n\n", ctx.Query))

	b.WriteString("System:\n")
	b.WriteString(fmt.Sprintf("  OS: %s\n", ctx.OS))
	b.WriteString(fmt.Sprintf("  Dir: %s\n", ctx.CWD))
	b.WriteString(fmt.Sprintf("  User: %s\n", ctx.Username))
	b.WriteString(fmt.Sprintf("  Shell: %s\n", ctx.Shell))

	switch strings.ToLower(shell) {
	case "fish":
		b.WriteString("Use idiomatic fish syntax only.\n")
	case "powershell":
		b.WriteString("Use idiomatic PowerShell. No bash.\n")
	}

	if explain {
		b.WriteString("Respond with the command first, then add 'EXPLANATION:' on a new line with a *very* brief explanation.\n")
	} else {
		b.WriteString("Output only the command, nothing else.\n")
	}

	return b.String(), nil
}
