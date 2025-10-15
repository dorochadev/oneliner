package executor

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/dorochadev/oneliner/config"
)

var (
	whitespaceRegex    = regexp.MustCompile(`\s+`)
	hexEncodeRegex     = regexp.MustCompile(`\\x[0-9a-fA-F]{2}`)
	base64Regex        = regexp.MustCompile(`base64|b64decode|atob`)
	evalRegex          = regexp.MustCompile(`\beval\b|\bexec\b`)
	revRegex           = regexp.MustCompile(`\brev\b`)
	findDeleteRegex    = regexp.MustCompile(`\bfind\b.*-delete`)
	shredRegex         = regexp.MustCompile(`\bshred\b`)
	truncateRegex      = regexp.MustCompile(`\btruncate\b.*-s\s*0`)
	forkBombRegex      = regexp.MustCompile(`:\(\)\s*\{\s*:\|:&\s*\};?:`)
	infiniteLoopRegex  = regexp.MustCompile(`while\s+true|while\s*\[\s*1\s*\]|for\s*\(\(\s*;;\s*\)\)`)
	sleepWaitReadRegex = regexp.MustCompile(`\b(sleep|wait|read)\b`)
	ddLargeRegex       = regexp.MustCompile(`\bdd\b.*bs=.*count=.*[MGT]`)
	tarNcRegex         = regexp.MustCompile(`\btar\b.*\|.*\bnc\b`)
	curlUploadRegex    = regexp.MustCompile(`\bcurl\b.*--data.*@`)
	wgetPostRegex      = regexp.MustCompile(`\bwget\b.*--post-file`)
	scpRegex           = regexp.MustCompile(`\bscp\b.*@.*:`)
	rsyncRegex         = regexp.MustCompile(`\brsync\b.*@.*:`)
	chmodEtcRegex      = regexp.MustCompile(`\b(chmod|chown)\b.*/etc`)
	chmodZeroRegex     = regexp.MustCompile(`\bchmod\b.*\b0+\b`)
	// privilege escalation
	sudoRegex   = regexp.MustCompile(`\bsudo\s+`)
	suRegex     = regexp.MustCompile(`\bsu\s+`)
	suDashRegex = regexp.MustCompile(`\bsu\s+-`)
	doasRegex   = regexp.MustCompile(`\bdoas\b`)
	pkexecRegex = regexp.MustCompile(`\bpkexec\b`)
	// rm patterns
	rmRegexes = []*regexp.Regexp{
		regexp.MustCompile(`\brm\s+.*-[a-z]*r[a-z]*.*-[a-z]*f`),
		regexp.MustCompile(`\brm\s+.*--recursive.*--force`),
		regexp.MustCompile(`\brm\s+.*--force.*--recursive`),
		regexp.MustCompile(`/bin/rm\s+.*-[a-z]*[rf]`),
		regexp.MustCompile(`\$\((which\s+rm)\)`),
	}
	// dangerous path checks (independent checks)
	dangerousPathRegexes = []*regexp.Regexp{
		regexp.MustCompile(`\s+/\s*$`),
		regexp.MustCompile(`\s+/\*`),
		regexp.MustCompile(`\s+/home\b`),
		regexp.MustCompile(`\s+/etc\b`),
		regexp.MustCompile(`\s+/usr\b`),
		regexp.MustCompile(`\s+/var\b`),
		regexp.MustCompile(`\s+/boot\b`),
		regexp.MustCompile(`\s+~\s*($|/)`),
		regexp.MustCompile(`\s+\$home\b`),
		regexp.MustCompile(`[a-z]:\\\?\*`),
	}
	// disk/partition operations
	diskOpRegexes = []*regexp.Regexp{
		regexp.MustCompile(`\bdd\b.*of\s*=\s*/dev/`),
		regexp.MustCompile(`>\s*/dev/(sd[a-z]|nvme|hd[a-z])`),
		regexp.MustCompile(`\bmkfs\b`),
		regexp.MustCompile(`\bfdisk\b`),
		regexp.MustCompile(`\bparted\b`),
		regexp.MustCompile(`\bgdisk\b`),
		regexp.MustCompile(`\bcfdisk\b`),
		regexp.MustCompile(`\bmkswap\b`),
		regexp.MustCompile(`\bsgdisk\b`),
	}
	// network patterns
	networkRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(curl|wget).*\|.*\bsh\b`),
		regexp.MustCompile(`(curl|wget).*\|.*\bbash\b`),
		regexp.MustCompile(`(curl|wget).*\|.*\bpython\b`),
		regexp.MustCompile(`(curl|wget).*>\s*/tmp/.*&&.*\bsh\b`),
		regexp.MustCompile(`\bnc\b.*-l.*-e`),
		regexp.MustCompile(`\bncat\b.*--exec`),
	}
)

type RiskLevel int

const (
	RiskNone RiskLevel = iota
	RiskLow
	RiskMedium
	RiskHigh
	RiskCritical
)

type RiskAssessment struct {
	Level   RiskLevel
	Reasons []string
}

// Normalized command for pattern matching (lowercase, collapsed whitespace)
func normalizeCommand(cmd string) string {
	// Remove extra whitespace
	normalized := whitespaceRegex.ReplaceAllString(strings.TrimSpace(cmd), " ")
	return strings.ToLower(normalized)
}

// Check for command obfuscation techniques
func detectObfuscation(cmd string) []string {
	var issues []string

	// Hex encoding
	if hexEncodeRegex.MatchString(cmd) {
		issues = append(issues, "hex-encoded characters detected (possible obfuscation)")
	}

	// Base64
	if base64Regex.MatchString(cmd) {
		issues = append(issues, "base64 encoding/decoding detected (possible obfuscation)")
	}

	// Eval constructs
	if evalRegex.MatchString(cmd) {
		issues = append(issues, "eval/exec detected (dynamic code execution)")
	}

	// Reverse operations
	if revRegex.MatchString(cmd) {
		issues = append(issues, "reverse command detected (possible obfuscation)")
	}

	// Excessive escaping
	escapeCount := strings.Count(cmd, "\\")
	quoteCount := strings.Count(cmd, `"`) + strings.Count(cmd, "'")
	if escapeCount > 5 || quoteCount > 6 {
		issues = append(issues, "excessive escaping/quoting detected")
	}

	return issues
}

// Check for privilege escalation
func detectPrivilegeEscalation(cmd string, intentionalSudo bool) []string {
	var issues []string

	// If sudo was intentionally added via --sudo flag, skip sudo checks
	if intentionalSudo {
		return issues
	}

	normalized := normalizeCommand(cmd)

	// Sudo variants
	patterns := []struct {
		pattern *regexp.Regexp
		desc    string
	}{
		{sudoRegex, "sudo privilege escalation"},
		{suRegex, "su privilege escalation"},
		{suDashRegex, "su with privilege escalation"},
		{doasRegex, "doas privilege escalation"},
		{pkexecRegex, "pkexec privilege escalation"},
	}

	for _, p := range patterns {
		if p.pattern.MatchString(normalized) {
			issues = append(issues, p.desc)
		}
	}

	return issues
}

// Check for destructive file operations
func detectDestructiveFileOps(cmd string) []string {
	var issues []string
	normalized := normalizeCommand(cmd)

	// rm variations
	for _, r := range rmRegexes {
		if r.MatchString(normalized) {
			// Check if targeting dangerous paths
			foundDanger := false
			for _, pathRe := range dangerousPathRegexes {
				if pathRe.MatchString(normalized) {
					issues = append(issues, "destructive rm command targeting critical path")
					foundDanger = true
					break
				}
			}

			if !foundDanger {
				issues = append(issues, "destructive rm -rf detected (verify target path)")
			}
			break
		}
	}

	// find -delete
	if findDeleteRegex.MatchString(normalized) {
		issues = append(issues, "find -delete can remove many files (potentially destructive)")
	}

	// shred
	if shredRegex.MatchString(normalized) {
		issues = append(issues, "shred detected (secure file deletion, unrecoverable)")
	}

	// truncate
	if truncateRegex.MatchString(normalized) {
		issues = append(issues, "truncate to zero detected (data loss)")
	}

	return issues
}

// Check for disk/partition operations
func detectDiskOperations(cmd string) []string {
	var issues []string
	normalized := normalizeCommand(cmd)

	for _, op := range diskOpRegexes {
		if op.MatchString(normalized) {
			desc := ""
			switch {
			case op == diskOpRegexes[0]:
				desc = "dd writing to raw device (can overwrite entire disk)"
			case op == diskOpRegexes[1]:
				desc = "output redirection to block device"
			default:
				// Extract a more meaningful description from the regex pattern
				desc = "disk/partition operation detected"
			}
			issues = append(issues, desc)
		}
	}

	return issues
}

// Check for system file modifications
func detectSystemFileModification(cmd string) []string {
	var issues []string
	normalized := normalizeCommand(cmd)

	// Critical files
	criticalFiles := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/etc/sudoers",
		"/etc/fstab",
		"/etc/hosts",
		"/boot/",
		"/etc/systemd",
		"/etc/init",
	}

	// Check for writes to critical files
	writeOps := []string{`>`, `>>`, `\btee\b`, `\bsed\b.*-i`}

	for _, file := range criticalFiles {
		for _, op := range writeOps {
			pattern := op + `.*` + regexp.QuoteMeta(file)
			if matched, _ := regexp.MatchString(pattern, normalized); matched {
				issues = append(issues, fmt.Sprintf("modification to critical system file: %s", file))
				break
			}
			// Also check reverse (file ... op)
			reversePattern := regexp.QuoteMeta(file) + `.*` + op
			if matched, _ := regexp.MatchString(reversePattern, normalized); matched {
				issues = append(issues, fmt.Sprintf("modification to critical system file: %s", file))
				break
			}
		}
	}

	// Chmod/chown on system dirs
	if chmodEtcRegex.MatchString(normalized) {
		issues = append(issues, "permission change on /etc directory")
	}

	if chmodZeroRegex.MatchString(normalized) {
		issues = append(issues, "chmod removing all permissions (files will be inaccessible)")
	}

	return issues
}

// Check for network/download operations
func detectNetworkOperations(cmd string) []string {
	var issues []string
	normalized := normalizeCommand(cmd)

	for _, p := range networkRegexes {
		if p.MatchString(normalized) {
			switch {
			case p == networkRegexes[0]:
				issues = append(issues, "piping download directly to shell (dangerous)")
			case p == networkRegexes[1]:
				issues = append(issues, "piping download to bash")
			case p == networkRegexes[2]:
				issues = append(issues, "piping download to python")
			case p == networkRegexes[3]:
				issues = append(issues, "download and execute pattern")
			case p == networkRegexes[4]:
				issues = append(issues, "netcat with command execution")
			case p == networkRegexes[5]:
				issues = append(issues, "ncat with command execution")
			default:
				issues = append(issues, "network operation detected")
			}
		}
	}

	return issues
}

// Check for fork bombs and resource exhaustion
func detectResourceExhaustion(cmd string) []string {
	var issues []string

	// Classic fork bomb
	if forkBombRegex.MatchString(cmd) {
		issues = append(issues, "fork bomb detected (will crash system)")
	}

	// Infinite loops
	if infiniteLoopRegex.MatchString(cmd) {
		if !sleepWaitReadRegex.MatchString(cmd) {
			issues = append(issues, "infinite loop without delay (potential resource exhaustion)")
		}
	}

	// Massive file creation
	if ddLargeRegex.MatchString(cmd) {
		issues = append(issues, "large file creation with dd")
	}

	return issues
}

// Check for data exfiltration patterns
func detectDataExfiltration(cmd string) []string {
	var issues []string
	normalized := normalizeCommand(cmd)

	patterns := []struct {
		pattern *regexp.Regexp
		desc    string
	}{
		{tarNcRegex, "archiving and sending over network"},
		{curlUploadRegex, "uploading file via curl"},
		{wgetPostRegex, "uploading file via wget"},
		{scpRegex, "secure copy to remote host"},
		{rsyncRegex, "rsync to remote host"},
	}

	for _, p := range patterns {
		if p.pattern.MatchString(normalized) {
			issues = append(issues, p.desc)
		}
	}

	return issues
}

// Main assessment function
func AssessCommandRisk(command string, usedSudoFlag bool) RiskAssessment {
	trimmed := strings.TrimSpace(command)
	assessment := RiskAssessment{
		Level:   RiskNone,
		Reasons: []string{},
	}

	if trimmed == "" {
		assessment.Reasons = append(assessment.Reasons, "empty command")
		return assessment
	}

	// Control character check
	for _, r := range trimmed {
		if r == '\x00' || (!unicode.IsPrint(r) && !unicode.IsSpace(r)) {
			assessment.Reasons = append(assessment.Reasons, "contains invalid control characters")
			assessment.Level = RiskHigh
			return assessment
		}
	}

	// Run all detection functions
	var allIssues [][]string

	allIssues = append(allIssues, detectObfuscation(trimmed))
	allIssues = append(allIssues, detectPrivilegeEscalation(trimmed, usedSudoFlag))
	allIssues = append(allIssues, detectDestructiveFileOps(trimmed))
	allIssues = append(allIssues, detectDiskOperations(trimmed))
	allIssues = append(allIssues, detectSystemFileModification(trimmed))
	allIssues = append(allIssues, detectNetworkOperations(trimmed))
	allIssues = append(allIssues, detectResourceExhaustion(trimmed))
	allIssues = append(allIssues, detectDataExfiltration(trimmed))

	// Flatten and deduplicate
	seen := make(map[string]bool)
	for _, issues := range allIssues {
		for _, issue := range issues {
			if !seen[issue] {
				seen[issue] = true
				assessment.Reasons = append(assessment.Reasons, issue)
			}
		}
	}

	// Check for blacklisted binaries from config and mark critical if found
	normalized := normalizeCommand(trimmed)
	if cfg, err := config.Load(""); err == nil {
		if len(cfg.BlacklistedBinaries) > 0 {
			for _, bin := range cfg.BlacklistedBinaries {
				pattern := `\b` + regexp.QuoteMeta(strings.ToLower(bin)) + `\b`
				if matched, _ := regexp.MatchString(pattern, normalized); matched {
					assessment.Reasons = append(assessment.Reasons, fmt.Sprintf("executes blacklisted binary: %s", bin))
					assessment.Level = RiskCritical
					return assessment
				}
			}
		}
	}

	// Determine risk level based on issues found
	if len(assessment.Reasons) == 0 {
		assessment.Level = RiskNone
	} else {
		// Calculate risk based on specific patterns
		criticalKeywords := []string{"fork bomb", "disk", "partition", "/etc/passwd", "/etc/shadow", "crash system"}
		highKeywords := []string{"destructive", "rm -rf", "overwrite", "erase", "unrecoverable"}
		mediumKeywords := []string{"sudo", "privilege", "critical"}

		for _, reason := range assessment.Reasons {
			lowerReason := strings.ToLower(reason)

			for _, kw := range criticalKeywords {
				if strings.Contains(lowerReason, kw) {
					assessment.Level = RiskCritical
					goto done
				}
			}

			for _, kw := range highKeywords {
				if strings.Contains(lowerReason, kw) && assessment.Level < RiskHigh {
					assessment.Level = RiskHigh
				}
			}

			for _, kw := range mediumKeywords {
				if strings.Contains(lowerReason, kw) && assessment.Level < RiskMedium {
					assessment.Level = RiskMedium
				}
			}
		}

		if assessment.Level == RiskNone {
			assessment.Level = RiskLow
		}
	}

done:
	return assessment
}

// Get risk level as string
func (r RiskLevel) String() string {
	switch r {
	case RiskNone:
		return "None"
	case RiskLow:
		return "Low"
	case RiskMedium:
		return "Medium"
	case RiskHigh:
		return "High"
	case RiskCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}
