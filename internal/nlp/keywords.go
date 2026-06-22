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
		"crash", "hang", "infinite loop", "catastrophic backtracking",
	},

	// --- Path / file access ---
	"path traversal": {
		"path traversal", "directory traversal", "dot dot slash", "../",
		"..\\", "zip slip", "arbitrary file read", "arbitrary file write",
		"file path injection", "lfi", "local file inclusion",
		"remote file inclusion", "rfi", "file inclusion",
	},
	"path canonicalization": {
		"canonicaliz", "normalize path", "filepath.clean", "path.resolve",
		"realpath", "base directory", "allowlist path", "restrict path",
		"chroot", "validate path", "reject ..",
	},

	// --- SSRF ---
	"server side request forgery": {
		"server side request forgery", "server-side request forgery", "ssrf",
		"internal endpoint", "metadata endpoint", "169.254.169.254", "imds",
		"cloud metadata", "internal service", "blind ssrf",
	},
	"ssrf mitigation": {
		"block internal", "deny private ip", "allowlist host", "allow-list host",
		"dns rebinding", "validate url", "egress filter", "imdsv2",
		"resolve and check", "private ip range",
	},

	// --- XXE / XML ---
	"xml external entity": {
		"xml external entity", "xxe", "external entity", "doctype", "<!entity",
		"system entity", "entity expansion", "billion laughs",
	},
	"disable external entities": {
		"disable doctype", "disable external entities", "disallow-doctype-decl",
		"disable entity", "secure processing", "xxe protection", "no external dtd",
	},

	// --- XSS ---
	"cross site scripting": {
		"cross site scripting", "cross-site scripting", "xss", "stored xss",
		"reflected xss", "dom xss", "script injection", "innerhtml",
		"document.write", "dangerouslysetinnerhtml",
	},
	"output encoding": {
		"output encoding", "html encode", "html escape", "context encoding",
		"escape output", "sanitize html", "content security policy", "csp",
		"textcontent",
	},

	// --- CSRF / SSRF-adjacent web ---
	"cross site request forgery": {
		"cross site request forgery", "cross-site request forgery", "csrf",
		"anti-csrf", "csrf token", "samesite", "state-changing request",
	},
	"origin validation": {
		"origin check", "verify origin", "validate origin", "check referer",
		"allowed origins", "cors policy", "same-origin",
	},

	// --- Access control / IDOR ---
	"broken access control": {
		"broken access control", "access control", "idor",
		"insecure direct object reference", "bola", "broken object level",
		"missing authorization", "authorization bypass", "horizontal privilege",
		"vertical privilege", "missing access check", "object level authorization",
	},
	"enforce authorization": {
		"enforce authorization", "ownership check", "verify ownership",
		"check permission", "authorization check", "scope check",
		"per-object check", "validate ownership", "acl",
	},

	// --- SSTI ---
	"server side template injection": {
		"server side template injection", "server-side template injection",
		"ssti", "template injection", "jinja2", "render_template_string",
		"sandbox escape", "{{", "expression language", "ognl", "spel",
	},
	"safe templating": {
		"sandbox", "autoescape", "logic-less template", "static template",
		"do not render user", "render with context only", "disable eval",
	},

	// --- Insecure deserialization ---
	"insecure deserialization": {
		"insecure deserialization", "unsafe deserialization", "deserialization",
		"pickle", "yaml.load", "objectinputstream", "readobject",
		"binaryformatter", "marshal", "gadget chain", "unserialize",
	},
	"safe deserialization": {
		"safe loader", "yaml.safe_load", "json instead", "allowlist class",
		"whitelist class", "signed payload", "do not deserialize untrusted",
		"hmac", "integrity check", "type filter",
	},

	// --- Prototype pollution ---
	"prototype pollution": {
		"prototype pollution", "__proto__", "constructor.prototype",
		"prototype chain", "object prototype", "deep merge pollution",
	},
	"prototype pollution mitigation": {
		"object.create(null)", "freeze prototype", "reject __proto__",
		"map instead of object", "hasownproperty", "null prototype",
		"block proto key",
	},

	// --- Race condition ---
	"race condition": {
		"race condition", "toctou", "time of check", "time-of-check",
		"time of use", "check-then-act", "non-atomic", "data race",
		"concurrent access", "double spend",
	},
	"atomic operation": {
		"atomic", "mutex", "lock", "synchroniz", "transaction",
		"compare-and-swap", "serializable isolation", "select for update",
		"critical section",
	},

	// --- Authentication / JWT / crypto ---
	"authentication weakness": {
		"authentication weakness", "broken authentication", "jwt none",
		"alg none", "algorithm confusion", "kid injection", "token forgery",
		"signature bypass", "weak secret", "hardcoded secret",
		"missing signature verification", "accept unsigned",
	},
	"verify signature": {
		"verify signature", "validate signature", "enforce algorithm",
		"allowlist algorithm", "reject none", "rotate secret", "strong secret",
		"verify issuer", "verify audience", "validate aud", "validate iss",
		"constant time compare",
	},
	"weak cryptography": {
		"weak cryptography", "weak crypto", "type juggling", "magic hash",
		"loose comparison", "== comparison", "md5", "sha1 password",
		"predictable", "insufficient entropy", "ecb mode", "static iv",
	},

	// --- Mass assignment ---
	"mass assignment": {
		"mass assignment", "over-posting", "overposting", "auto-binding",
		"autobind", "unrestricted binding", "extra fields", "privilege field",
	},
	"allowlist fields": {
		"allowlist field", "whitelist field", "explicit binding", "dto",
		"bind only", "permit params", "strong parameters", "field filter",
		"ignore unknown fields",
	},

	// --- HTTP request smuggling / cache ---
	"request smuggling": {
		"request smuggling", "http smuggling", "cl.te", "te.cl",
		"content-length", "transfer-encoding", "desync", "smuggl",
	},
	"cache poisoning": {
		"cache poisoning", "unkeyed header", "cache key", "vary header",
		"web cache deception", "poison the cache",
	},

	// --- NoSQL injection (extends injection family) ---
	"nosql injection": {
		"nosql injection", "mongodb injection", "$where", "$ne", "$gt",
		"operator injection", "query operator", "json injection",
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

	// NOTE: deliberately NO "fall back to the entire securityTerms map" here.
	// That old behaviour was a scoring footgun: for any challenge whose
	// vulnerability class isn't in the canonical map (path traversal, SSRF,
	// XXE, SSTI, prototype pollution, race conditions, …) every canonical term
	// became "expected", so even a perfect answer matched only a tiny fraction
	// and the challenge was mathematically unpassable. When the canonical map
	// has no entry for a challenge, conceptCoverage falls back to keyword
	// overlap against the challenge's own ground-truth text instead, which is
	// always category-appropriate. Empty maps here are intentional.
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

// stopwords are high-frequency English/doc words that carry no discriminating
// security signal. extractKeywords drops them so the per-challenge vocabulary
// reflects the actual vulnerability, not boilerplate prose.
var stopwords = map[string]bool{
	"the": true, "this": true, "that": true, "these": true, "those": true,
	"with": true, "from": true, "into": true, "over": true, "under": true,
	"which": true, "where": true, "when": true, "what": true, "while": true,
	"would": true, "could": true, "should": true, "their": true, "there": true,
	"then": true, "than": true, "they": true, "them": true, "your": true,
	"will": true, "have": true, "has": true, "had": true, "been": true,
	"being": true, "because": true, "about": true, "above": true, "below": true,
	"also": true, "such": true, "only": true, "more": true, "most": true,
	"some": true, "any": true, "all": true, "each": true, "both": true,
	"using": true, "uses": true, "used": true, "make": true, "makes": true,
	"made": true, "does": true, "done": true, "doing": true, "here": true,
	"however": true, "therefore": true, "thus": true, "hence": true,
	"example": true, "instead": true, "without": true, "within": true,
	"between": true, "against": true, "before": true, "after": true,
	"code": true, "line": true, "lines": true, "function": true, "method": true,
	"value": true, "values": true, "field": true, "data": true, "system": true,
	"application": true, "service": true, "server": true, "request": true,
	"attacker": true, "vulnerability": true, "vulnerable": true, "security": true,
	"developer": true, "user": true, "users": true,
	// NOTE: discriminating security tokens like "input", "query", "token",
	// "shell", "exec", "deserialize" are deliberately NOT stopwords.
}

// extractKeywords pulls the discriminating content tokens out of a challenge's
// ground-truth text (TargetVulnerability or ConceptualFix). It lowercases,
// splits on non-identifier characters (keeping "_" so identifiers like
// render_template_string survive), and keeps tokens of length >= 4 that aren't
// stopwords. The result is the per-challenge "expected vocabulary" that
// conceptCoverage scores a user's answer against — this is what makes scoring
// work for vulnerability classes the canonical map doesn't enumerate.
func extractKeywords(text string) []string {
	lower := strings.ToLower(text)
	fields := strings.FieldsFunc(lower, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_')
	})

	seen := make(map[string]bool, len(fields))
	var out []string
	for _, tok := range fields {
		if len(tok) < 4 {
			continue
		}
		if stopwords[tok] {
			continue
		}
		if seen[tok] {
			continue
		}
		seen[tok] = true
		out = append(out, tok)
	}
	return out
}

// countKeywordHits returns how many of the expected keywords appear (as a
// substring) in the lowercased user answer.
func countKeywordHits(lowerAnswer string, keywords []string) int {
	hits := 0
	for _, kw := range keywords {
		if strings.Contains(lowerAnswer, kw) {
			hits++
		}
	}
	return hits
}
