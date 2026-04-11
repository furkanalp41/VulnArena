package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

const defaultServerURL = "http://localhost:8080/api/v1"

// Config holds the CLI configuration stored on disk.
type Config struct {
	APIKey    string `json:"api_key"`
	ServerURL string `json:"server_url"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "auth":
		handleAuth(os.Args[2:])
	case "arena":
		handleArena(os.Args[2:])
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`
╔══════════════════════════════════════════╗
║          [VULNARENA] CLI v1.0            ║
╚══════════════════════════════════════════╝

USAGE:
  vulnarena <command> <subcommand> [flags]

COMMANDS:
  auth login <api_key>                Authenticate with your API key
  arena list                          List available challenges
  arena submit <id> -m "explanation"  Submit a solution

EXAMPLES:
  vulnarena auth login va_abc123...
  vulnarena arena list
  vulnarena arena submit 550e8400-e29b-41d4-a716-446655440000 -m "SQL injection via unsanitized input"`)
}

// --- Config management ---

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".vulnarena")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.ServerURL == "" {
		cfg.ServerURL = defaultServerURL
	}

	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	if cfg.ServerURL == "" {
		cfg.ServerURL = defaultServerURL
	}

	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath(), data, 0600)
}

func requireConfig() *Config {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: not authenticated. Run 'vulnarena auth login <api_key>' first.")
		os.Exit(1)
	}
	return cfg
}

// --- HTTP helper ---

func apiRequest(method, path string, body any) ([]byte, int, error) {
	cfg := requireConfig()

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("encoding request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, cfg.ServerURL+path, reqBody)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("X-API-Key", cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// --- Auth commands ---

func handleAuth(args []string) {
	if len(args) < 2 || args[0] != "login" {
		fmt.Println("Usage: vulnarena auth login <api_key>")
		os.Exit(1)
	}

	apiKey := args[1]
	if !strings.HasPrefix(apiKey, "va_") {
		fmt.Fprintln(os.Stderr, "Error: invalid API key format. Keys start with 'va_'.")
		os.Exit(1)
	}

	// Save config first so apiRequest can use it
	cfg := &Config{
		APIKey:    apiKey,
		ServerURL: defaultServerURL,
	}
	if err := saveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	// Validate by fetching user info
	body, status, err := apiRequest("GET", "/users/me", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		// Remove invalid config
		os.Remove(configPath())
		os.Exit(1)
	}

	if status != 200 {
		fmt.Fprintf(os.Stderr, "Error: authentication failed (HTTP %d)\n", status)
		os.Remove(configPath())
		os.Exit(1)
	}

	var user struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Remove(configPath())
		os.Exit(1)
	}

	fmt.Printf("[+] Authenticated as %s (role: %s)\n", user.Username, user.Role)
	fmt.Printf("[+] Config saved to %s\n", configPath())
}

// --- Arena commands ---

func handleArena(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: vulnarena arena <list|submit>")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		arenaList()
	case "submit":
		arenaSubmit(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown arena subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func arenaList() {
	body, status, err := apiRequest("GET", "/arena/challenges?limit=50", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if status != 200 {
		fmt.Fprintf(os.Stderr, "Error: HTTP %d\n", status)
		os.Exit(1)
	}

	var resp struct {
		Challenges []struct {
			ID         string `json:"id"`
			Title      string `json:"title"`
			Difficulty int    `json:"difficulty"`
			Points     int    `json:"points"`
			Language   struct {
				Name string `json:"name"`
			} `json:"language"`
			VulnCategory struct {
				Name string `json:"name"`
			} `json:"vuln_category"`
		} `json:"challenges"`
		Total int `json:"total"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n[VULNARENA] %d challenges available\n\n", resp.Total)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tDIFF\tPTS\tLANG\tCATEGORY")
	fmt.Fprintln(w, "──\t─────\t────\t───\t────\t────────")

	for _, c := range resp.Challenges {
		// Truncate ID to first 8 chars for readability
		shortID := c.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		fmt.Fprintf(w, "%s\t%s\t%d/10\t%d\t%s\t%s\n",
			shortID, c.Title, c.Difficulty, c.Points, c.Language.Name, c.VulnCategory.Name)
	}
	w.Flush()
	fmt.Println()
}

func arenaSubmit(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: vulnarena arena submit <challenge-id> -m \"explanation\"")
		os.Exit(1)
	}

	challengeID := args[0]
	message := ""

	// Parse -m flag
	for i := 1; i < len(args); i++ {
		if args[i] == "-m" && i+1 < len(args) {
			message = args[i+1]
			break
		}
	}

	if message == "" {
		fmt.Fprintln(os.Stderr, "Error: -m flag is required. Provide your vulnerability analysis.")
		os.Exit(1)
	}

	fmt.Println("[*] Submitting analysis...")

	payload := map[string]string{
		"answer_text": message,
	}

	body, status, err := apiRequest("POST", "/arena/challenges/"+challengeID+"/submit", payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if status != 200 && status != 201 {
		var errResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(body, &errResp)
		fmt.Fprintf(os.Stderr, "Error: %s (HTTP %d)\n", errResp.Error, status)
		os.Exit(1)
	}

	var resp struct {
		Submission struct {
			Score     float64 `json:"score"`
			IsCorrect bool   `json:"is_correct"`
		} `json:"submission"`
		Feedback struct {
			TerminalLog []string `json:"terminal_log"`
			Passed      bool     `json:"passed"`
		} `json:"feedback"`
		FirstBlood bool `json:"first_blood"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║          ANALYSIS RESULTS                ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println()

	for _, line := range resp.Feedback.TerminalLog {
		fmt.Println("  " + line)
	}

	fmt.Println()
	fmt.Printf("  Score: %.0f%%\n", resp.Submission.Score)

	if resp.Submission.IsCorrect {
		fmt.Println("  Status: [+] PASSED")
	} else {
		fmt.Println("  Status: [-] INSUFFICIENT")
	}

	if resp.FirstBlood {
		fmt.Println("  [!] FIRST BLOOD! You are the first to solve this challenge!")
	}

	fmt.Println()
}
