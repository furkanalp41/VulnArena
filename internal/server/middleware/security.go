package middleware

import "net/http"

// Security adds production security headers to every response.
func Security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()

		// HSTS: enforce HTTPS for 1 year, include subdomains, allow preload
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Prevent clickjacking
		h.Set("X-Frame-Options", "DENY")

		// Prevent MIME-type sniffing
		h.Set("X-Content-Type-Options", "nosniff")

		// Disable browser-side XSS filters (modern CSP replaces this)
		h.Set("X-XSS-Protection", "0")

		// Restrict referrer leakage
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Prevent embedding in other origins
		h.Set("Cross-Origin-Opener-Policy", "same-origin")

		// Restrict resource loading to same-origin and trusted CDNs
		h.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: https:; "+
				"font-src 'self'; "+
				"connect-src 'self' ws: wss:; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'")

		// Control browser features
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), interest-cohort=()")

		next.ServeHTTP(w, r)
	})
}
