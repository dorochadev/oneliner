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

const (
	minQueryLength = 5
	minWordCount   = 2
)

// Build constructs the prompt for the LLM. Returns an error if the query is too short or vague.
func Build(ctx Context, cfg *config.Config, explain, breakdown bool) (string, error) {
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
	b.WriteString(fmt.Sprintf("Output only a single safe %s one-liner that accomplishes the following task:\n", shell))
	b.WriteString(fmt.Sprintf("%s\n\n", trimmedQuery))

	b.WriteString("System:\n")
	b.WriteString(fmt.Sprintf("  OS: %s\n", ctx.OS))
	b.WriteString(fmt.Sprintf("  Dir: %s\n", ctx.CWD))
	b.WriteString(fmt.Sprintf("  User: %s\n", ctx.Username))
	b.WriteString(fmt.Sprintf("  Shell: %s\n", ctx.Shell))

	appendShellSpecificInstructions(&b, shell)
	appendExplanationInstructions(&b, explain, breakdown)

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

func appendExplanationInstructions(b *strings.Builder, explain, breakdown bool) {
	if explain && breakdown {
		b.WriteString(`Output ONLY the command first (no code fences, no commentary before).
Then add 'EXPLANATION:' on a new line.
In the explanation:
- Briefly describe what the command does overall
- Mention what each main flag or pipe stage contributes
- Explain *how* and *why* the command works
Do NOT restate the user's question or describe concepts generally.
Keep it under 4 sentences.

After the explanation add a 'BREAKDOWN:' section on a new line.
In 'BREAKDOWN:' provide a **detailed numbered pipeline** describing each stage of the command in execution order.
- Include every relevant flag, pipe, redirection, or expansion.
- Explain what data is input/output at each step.
- Clarify how intermediate transformations work.
- Use as many numbered points as necessary; don't limit to a small fixed number.
- Each item should be 1-3 sentences, clear and precise.
Do NOT use code fences. Keep the focus on teaching the command's mechanics.
`)
		return
	}

	if explain {
		b.WriteString(`Output ONLY the command first (no code fences, no commentary before).
Then add 'EXPLANATION:' on a new line.
In the explanation:
- Briefly describe what the command does overall
- Mention what each main flag or pipe stage contributes
- Explain *how* and *why* the command works
Do NOT restate the user's question or describe concepts generally.
Keep it under 4 sentences.
`)
		return
	}

	if breakdown {
		b.WriteString(`Output ONLY the command first (no code fences, no commentary before).
		Then add a 'BREAKDOWN:' section on a new line.
		In 'BREAKDOWN:' provide a **detailed numbered pipeline** explaining each part of the command:
		- Cover every flag, pipe, redirection, or expansion used.
		- Explain what each stage does to the data or environment.
		- Include multiple numbered points if needed; do not limit the explanation.
		- Each point should be 1-3 sentences.
		- Focus on making the command fully understandable, as a learning guide.
		Do NOT use code fences.
`)
		return
	}

	b.WriteString("Output only the command, nothing else.\n")
}
