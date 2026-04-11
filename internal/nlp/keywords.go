package nlp

import "strings"

// securityTerms maps canonical vulnerability/fix concepts to their
// synonyms and related phrases. When a user's answer contains any
// synonym, the canonical term is considered "matched."
//
// This vocabulary is the foundation of v1 keyword scoring and also
// serves as a concept-extraction layer for future embedding-based scoring.
var securityTerms = map[string][]string{
	// --- Injection family ---
	"sql injection": {
		"sql injection", "sqli", "sql-injection", "sql inject",
		"inject sql", "injection attack", "injection vulnerability",
		"sql query injection", "database injection",
	},
	"string concatenation": {
		"string concatenation", "string interpolation", "string formatting",
		"sprintf", "fmt.sprintf", "format string", "string building",
		"concatenat", "template literal",
	},
	"unsanitized input": {
		"unsanitized", "unvalidated", "untrusted input", "user input",
		"user-supplied", "user controlled", "tainted", "unescaped",
		"no validation", "without validation", "not sanitized", "not validated",
		"raw input", "direct input",
	},
	"parameterized query": {
		"parameterized", "prepared statement", "bind parameter", "placeholder",
		"query parameter", "$1", "?", "bind variable", "parameteriz",
	},
	"input validation": {
		"input validation", "validate input", "sanitize", "sanitization",
		"whitelist", "allowlist", "escape", "encoding",
	},

	// --- Command injection ---
	"command injection": {
		"command injection", "os command injection", "shell injection",
		"cmd injection", "system command", "command execution",
		"os injection", "shell command injection",
	},
	"shell execution": {
		"exec", "child_process", "system(", "popen", "subprocess",
		"os.exec", "exec.command", "shell_exec", "backtick",
		"eval(", "os.system", "runtime.exec",
	},
	"shell metacharacter": {
		"semicolon", "pipe", "&&", "||", "backtick",
		"shell metacharacter", "command separator", "command chain",
		"special character",
	},
	"avoid shell": {
		"avoid shell", "no shell", "execfile", "allowlist command",
		"whitelist command", "restrict command", "api instead",
		"library function", "direct api",
	},

	// --- Memory corruption ---
	"buffer overflow": {
		"buffer overflow", "buffer overrun", "stack overflow",
		"heap overflow", "out of bounds", "oob write", "oob read",
		"memory overflow", "overflow vulnerability", "overflows the buffer",
	},
	"bounds checking": {
		"bounds check", "boundary check", "length check", "size check",
		"array bounds", "buffer size", "sizeof", "strnlen",
		"bounds validation", "limit check",
	},
	"unsafe function": {
		"strcpy", "strcat", "gets(", "sprintf", "scanf",
		"unsafe function", "dangerous function", "deprecated function",
		"no length limit",
	},
	"safe alternative": {
		"strncpy", "strncat", "snprintf", "fgets", "safe function",
		"safe alternative", "bounded copy", "size-limited",
		"strlcpy", "strlcat", "memcpy_s",
	},
	"memory corruption": {
		"memory corruption", "memory safety", "memory violation",
		"segfault", "segmentation fault", "use after free",
		"dangling pointer", "double free", "heap corruption",
	},

	// --- Auth / Access Control ---
	"authentication bypass": {
		"authentication bypass", "auth bypass", "login bypass",
		"bypass authentication", "skip auth", "always true",
		"tautology", "or 1=1", "' or '",
	},

	// --- Generic security concepts ---
	"remote code execution": {
		"remote code execution", "rce", "arbitrary code", "code execution",
		"execute arbitrary", "run arbitrary",
	},
	"privilege escalation": {
		"privilege escalation", "privesc", "escalate privilege",
		"root access", "admin access", "elevated privilege",
	},
	"denial of service": {
		"denial of service", "dos", "resource exhaustion",
		"crash", "hang", "infinite loop",
	},
}

// extractMatchedTerms finds which canonical security concepts appear
// in the given text by checking all synonyms.
func extractMatchedTerms(text string, targetTerms map[string][]string) []string {
	lower := strings.ToLower(text)
	var matched []string

	for canonical, synonyms := range targetTerms {
		for _, syn := range synonyms {
			if strings.Contains(lower, strings.ToLower(syn)) {
				matched = append(matched, canonical)
				break
			}
		}
	}

	return matched
}

// buildRelevantTerms selects the subset of securityTerms that are relevant
// to the given target vulnerability and conceptual fix descriptions.
func buildRelevantTerms(targetVuln, conceptualFix string) (vulnTerms, fixTerms map[string][]string) {
	combined := strings.ToLower(targetVuln + " " + conceptualFix)
	vulnTerms = make(map[string][]string)
	fixTerms = make(map[string][]string)

	for canonical, synonyms := range securityTerms {
		for _, syn := range synonyms {
			if strings.Contains(combined, strings.ToLower(syn)) {
				// Classify as vuln-related or fix-related based on which
				// target description contains it
				if containsAnySynonym(strings.ToLower(targetVuln), synonyms) {
					vulnTerms[canonical] = synonyms
				}
				if containsAnySynonym(strings.ToLower(conceptualFix), synonyms) {
					fixTerms[canonical] = synonyms
				}
				break
			}
		}
	}

	// If no specific terms matched, use all terms (fallback for unusual challenges)
	if len(vulnTerms) == 0 {
		vulnTerms = securityTerms
	}
	if len(fixTerms) == 0 {
		fixTerms = securityTerms
	}

	return vulnTerms, fixTerms
}

func containsAnySynonym(text string, synonyms []string) bool {
	for _, syn := range synonyms {
		if strings.Contains(text, strings.ToLower(syn)) {
			return true
		}
	}
	return false
}
