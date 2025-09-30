package executor

// entire class is ai .... i cant lie

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
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
	Flags   []string // specific dangerous patterns detected
}

// Normalized command for pattern matching (lowercase, collapsed whitespace)
func normalizeCommand(cmd string) string {
	// Remove extra whitespace
	normalized := regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(cmd), " ")
	return strings.ToLower(normalized)
}

// Check for command obfuscation techniques
func detectObfuscation(cmd string) []string {
	var issues []string
	
	// Hex encoding
	if regexp.MustCompile(`\\x[0-9a-fA-F]{2}`).MatchString(cmd) {
		issues = append(issues, "hex-encoded characters detected (possible obfuscation)")
	}
	
	// Base64
	if regexp.MustCompile(`base64|b64decode|atob`).MatchString(cmd) {
		issues = append(issues, "base64 encoding/decoding detected (possible obfuscation)")
	}
	
	// Eval constructs
	if regexp.MustCompile(`\beval\b|\bexec\b`).MatchString(cmd) {
		issues = append(issues, "eval/exec detected (dynamic code execution)")
	}
	
	// Reverse operations
	if regexp.MustCompile(`\brev\b`).MatchString(cmd) {
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
func detectPrivilegeEscalation(cmd string) []string {
	var issues []string
	normalized := normalizeCommand(cmd)
	
	// Sudo variants
	patterns := []struct {
		pattern string
		desc    string
	}{
		{`\bsudo\s+`, "sudo privilege escalation"},
		{`\bsu\s+`, "su privilege escalation"},
		{`\bsu\s+-`, "su with privilege escalation"},
		{`\bdoas\b`, "doas privilege escalation"},
		{`\bpkexec\b`, "pkexec privilege escalation"},
	}
	
	for _, p := range patterns {
		if matched, _ := regexp.MatchString(p.pattern, normalized); matched {
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
	rmPatterns := []string{
		`\brm\s+.*-[a-z]*r[a-z]*.*-[a-z]*f`,  // rm -rf or -fr
		`\brm\s+.*--recursive.*--force`,       // long form
		`\brm\s+.*--force.*--recursive`,       // reversed
		`/bin/rm\s+.*-[a-z]*[rf]`,            // direct path
		`\$\(which\s+rm\)`,                    // which rm
	}
	
	for _, pattern := range rmPatterns {
		if matched, _ := regexp.MatchString(pattern, normalized); matched {
			// Check if targeting dangerous paths
			dangerousPaths := []string{
				`\s+/\s*$`,           // root
				`\s+/\*`,             // root with wildcard
				`\s+/home`,           // home dirs
				`\s+/etc`,            // system config
				`\s+/usr`,            // system binaries
				`\s+/var`,            // system data
				`\s+/boot`,           // boot files
				`\s+~`,               // home directory
				`\s+\$home`,          // $HOME variable
				`[a-z]:\\\?\*`,       // Windows drive root
			}
			
			for _, path := range dangerousPaths {
				if matched, _ := regexp.MatchString(pattern+`.*`+path, normalized); matched {
					issues = append(issues, fmt.Sprintf("destructive rm command targeting critical path"))
					break
				}
			}
			
			if len(issues) == 0 {
				issues = append(issues, "destructive rm -rf detected (verify target path)")
			}
			break
		}
	}
	
	// find -delete
	if regexp.MustCompile(`\bfind\b.*-delete`).MatchString(normalized) {
		issues = append(issues, "find -delete can remove many files (potentially destructive)")
	}
	
	// shred
	if regexp.MustCompile(`\bshred\b`).MatchString(normalized) {
		issues = append(issues, "shred detected (secure file deletion, unrecoverable)")
	}
	
	// truncate
	if regexp.MustCompile(`\btruncate\b.*-s\s*0`).MatchString(normalized) {
		issues = append(issues, "truncate to zero detected (data loss)")
	}
	
	return issues
}

// Check for disk/partition operations
func detectDiskOperations(cmd string) []string {
	var issues []string
	normalized := normalizeCommand(cmd)
	
	operations := []struct {
		pattern string
		desc    string
	}{
		{`\bdd\b.*of\s*=\s*/dev/`, "dd writing to raw device (can overwrite entire disk)"},
		{`>\s*/dev/(sd[a-z]|nvme|hd[a-z])`, "output redirection to block device"},
		{`\bmkfs\b`, "filesystem creation (will erase partition)"},
		{`\bfdisk\b`, "disk partitioning tool"},
		{`\bparted\b`, "partition editor"},
		{`\bgdisk\b`, "GPT partition tool"},
		{`\bcfdisk\b`, "curses-based partition tool"},
		{`\bmkswap\b`, "swap creation (will erase partition)"},
		{`\bsgdisk\b`, "GPT partition manipulation"},
	}
	
	for _, op := range operations {
		if matched, _ := regexp.MatchString(op.pattern, normalized); matched {
			issues = append(issues, op.desc)
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
	if matched, _ := regexp.MatchString(`\b(chmod|chown)\b.*/etc`, normalized); matched {
		issues = append(issues, "permission change on /etc directory")
	}
	
	if matched, _ := regexp.MatchString(`\bchmod\b.*\b0+\b`, normalized); matched {
		issues = append(issues, "chmod removing all permissions (files will be inaccessible)")
	}
	
	return issues
}

// Check for network/download operations
func detectNetworkOperations(cmd string) []string {
	var issues []string
	normalized := normalizeCommand(cmd)
	
	patterns := []struct {
		pattern string
		desc    string
	}{
		{`(curl|wget).*\|.*sh`, "piping download directly to shell (dangerous)"},
		{`(curl|wget).*\|.*bash`, "piping download to bash"},
		{`(curl|wget).*\|.*python`, "piping download to python"},
		{`(curl|wget).*>\s*/tmp/.*&&.*sh`, "download and execute pattern"},
		{`nc\b.*-l.*-e`, "netcat with command execution"},
		{`ncat\b.*--exec`, "ncat with command execution"},
	}
	
	for _, p := range patterns {
		if matched, _ := regexp.MatchString(p.pattern, normalized); matched {
			issues = append(issues, p.desc)
		}
	}
	
	return issues
}

// Check for fork bombs and resource exhaustion
func detectResourceExhaustion(cmd string) []string {
	var issues []string
	
	// Classic fork bomb
	if matched, _ := regexp.MatchString(`:\(\)\s*\{\s*:\|:&\s*\};?:`, cmd); matched {
		issues = append(issues, "fork bomb detected (will crash system)")
	}
	
	// Infinite loops
	if regexp.MustCompile(`while\s+true|while\s*\[\s*1\s*\]|for\s*\(\(\s*;;\s*\)\)`).MatchString(cmd) {
		if !regexp.MustCompile(`sleep|wait|read`).MatchString(cmd) {
			issues = append(issues, "infinite loop without delay (potential resource exhaustion)")
		}
	}
	
	// Massive file creation
	if regexp.MustCompile(`\bdd\b.*bs=.*count=.*[MGT]`).MatchString(cmd) {
		issues = append(issues, "large file creation with dd")
	}
	
	return issues
}

// Check for data exfiltration patterns
func detectDataExfiltration(cmd string) []string {
	var issues []string
	normalized := normalizeCommand(cmd)
	
	// Sending data over network
	patterns := []struct {
		pattern string
		desc    string
	}{
		{`tar.*\|.*nc`, "archiving and sending over network"},
		{`\bcurl\b.*--data.*@`, "uploading file via curl"},
		{`\bwget\b.*--post-file`, "uploading file via wget"},
		{`scp.*@.*:`, "secure copy to remote host"},
		{`rsync.*@.*:`, "rsync to remote host"},
	}
	
	for _, p := range patterns {
		if matched, _ := regexp.MatchString(p.pattern, normalized); matched {
			issues = append(issues, p.desc)
		}
	}
	
	return issues
}

// Main assessment function
func AssessCommandRisk(command string) RiskAssessment {
	trimmed := strings.TrimSpace(command)
	assessment := RiskAssessment{
		Level:   RiskNone,
		Reasons: []string{},
		Flags:   []string{},
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
	allIssues = append(allIssues, detectPrivilegeEscalation(trimmed))
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