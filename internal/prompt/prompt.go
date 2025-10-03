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
	
	// Validate query
	if err := validateQuery(trimmedQuery); err != nil {
		return "", err
	}

	shell := cfg.DefaultShell
	if shell == "" {
		shell = "bash"
	}

	var b strings.Builder
	b.Grow(512) // pre allocate approximate size

	b.WriteString(fmt.Sprintf("You are an expert in %s on %s systems.\n", shell, ctx.OS))
	b.WriteString(fmt.Sprintf("Write a single, safe one-liner in %s to:\n", shell))
	b.WriteString(fmt.Sprintf("%s\n\n", trimmedQuery))

	b.WriteString("System:\n")
	b.WriteString(fmt.Sprintf("  OS: %s\n", ctx.OS))
	b.WriteString(fmt.Sprintf("  Dir: %s\n", ctx.CWD))
	b.WriteString(fmt.Sprintf("  User: %s\n", ctx.Username))
	b.WriteString(fmt.Sprintf("  Shell: %s\n", ctx.Shell))

	appendShellSpecificInstructions(&b, shell)
	appendExplanationInstructions(&b, explain)

	return b.String(), nil
}

func validateQuery(query string) error {
	if len(query) < minQueryLength {
		return fmt.Errorf("query is too short (minimum %d characters); please provide a more detailed request", minQueryLength)
	}
	
	wordCount := len(strings.Fields(query))
	if wordCount < minWordCount {
		return fmt.Errorf("query is too vague (minimum %d words); please provide a more detailed request", minWordCount)
	}
	
	return nil
}

func appendShellSpecificInstructions(b *strings.Builder, shell string) {
	switch strings.ToLower(shell) {
	case "fish":
		b.WriteString("Use idiomatic fish syntax only.\n")
	case "powershell":
		b.WriteString("Use idiomatic PowerShell. No bash.\n")
	}
}

func appendExplanationInstructions(b *strings.Builder, explain bool) {
	if explain {
		b.WriteString("Respond with the command first, then add 'EXPLANATION:' on a new line with a *very* brief explanation.\n")
	} else {
		b.WriteString("Output only the command, nothing else.\n")
	}
}