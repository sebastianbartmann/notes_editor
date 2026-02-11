package claude

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"notes-editor/internal/linkedin"
	"notes-editor/internal/vault"

	"golang.org/x/net/html"
)

const (
	defaultBraveAPIKey        = "BSA4uayVZcDls43iE2p2exjSP6-VR_N"
	defaultWebSearchEndpoint  = "https://api.search.brave.com/res/v1/web/search"
	defaultWebSearchTimeout   = 8 * time.Second
	defaultWebSearchCacheTTL  = 15 * time.Minute
	defaultWebSearchMaxResult = 5
	maxWebSearchResultCount   = 20
	maxWebSearchDescChars     = 280
)

var (
	webSearchEndpoint = defaultWebSearchEndpoint
	webSearchClient   = &http.Client{Timeout: defaultWebSearchTimeout}
	webSearchCacheMu  sync.Mutex
	webSearchCache    = map[string]webSearchCacheEntry{}
)

type webSearchCacheEntry struct {
	expiresAt time.Time
	payload   string
}

// Tool definitions for Claude's tool use capability.
var ToolDefinitions = []map[string]any{
	{
		"name":        "read_file",
		"description": "Read the contents of a file from the notes vault",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file relative to the person's vault root",
				},
			},
			"required": []string{"path"},
		},
	},
	{
		"name":        "write_file",
		"description": "Write content to a file in the notes vault. Creates the file if it doesn't exist.",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file relative to the person's vault root",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Content to write to the file",
				},
			},
			"required": []string{"path", "content"},
		},
	},
	{
		"name":        "list_directory",
		"description": "List files and directories in the notes vault",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the directory relative to the person's vault root. Use '.' for root.",
				},
			},
			"required": []string{"path"},
		},
	},
	{
		"name":        "search_files",
		"description": "Search for text patterns in files within the notes vault",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{
					"type":        "string",
					"description": "Text pattern to search for (case-insensitive)",
				},
				"path": map[string]any{
					"type":        "string",
					"description": "Directory to search in, relative to vault root. Defaults to '.'",
				},
			},
			"required": []string{"pattern"},
		},
	},
	{
		"name":        "glob_files",
		"description": "Find files by glob pattern within the notes vault",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{
					"type":        "string",
					"description": "Glob pattern to match vault-relative file paths (supports **, *, ?)",
				},
				"path": map[string]any{
					"type":        "string",
					"description": "Directory to search in, relative to vault root. Defaults to '.'",
				},
				"limit": map[string]any{
					"type":        "number",
					"description": "Maximum number of results (default: 1000)",
				},
			},
			"required": []string{"pattern"},
		},
	},
	{
		"name":        "web_search",
		"description": "Search the web for information",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Search query",
				},
			},
			"required": []string{"query"},
		},
	},
	{
		"name":        "web_fetch",
		"description": "Fetch the content of a web page",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "URL to fetch",
				},
			},
			"required": []string{"url"},
		},
	},
	{
		"name":        "linkedin_post",
		"description": "Create a new LinkedIn post",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{
					"type":        "string",
					"description": "Post content",
				},
			},
			"required": []string{"text"},
		},
	},
	{
		"name":        "linkedin_read_comments",
		"description": "Read comments on a LinkedIn post",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"post_urn": map[string]any{
					"type":        "string",
					"description": "URN of the post to read comments from",
				},
			},
			"required": []string{"post_urn"},
		},
	},
	{
		"name":        "linkedin_post_comment",
		"description": "Post a comment on a LinkedIn post",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"post_urn": map[string]any{
					"type":        "string",
					"description": "URN of the post to comment on",
				},
				"text": map[string]any{
					"type":        "string",
					"description": "Comment text",
				},
			},
			"required": []string{"post_urn", "text"},
		},
	},
	{
		"name":        "linkedin_reply_comment",
		"description": "Reply to a comment on a LinkedIn post",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"post_urn": map[string]any{
					"type":        "string",
					"description": "URN of the post",
				},
				"parent_comment_urn": map[string]any{
					"type":        "string",
					"description": "URN of the comment to reply to",
				},
				"text": map[string]any{
					"type":        "string",
					"description": "Reply text",
				},
			},
			"required": []string{"post_urn", "parent_comment_urn", "text"},
		},
	},
}

// ToolExecutor handles tool execution with access to vault and services.
type ToolExecutor struct {
	store    *vault.Store
	linkedin *linkedin.Service
	person   string
}

// NewToolExecutor creates a new tool executor.
func NewToolExecutor(store *vault.Store, linkedin *linkedin.Service, person string) *ToolExecutor {
	return &ToolExecutor{
		store:    store,
		linkedin: linkedin,
		person:   person,
	}
}

// ExecuteTool executes a tool call and returns the result.
func (te *ToolExecutor) ExecuteTool(name string, input map[string]any) (string, error) {
	switch name {
	case "read_file":
		return te.readFile(input)
	case "write_file":
		return te.writeFile(input)
	case "list_directory":
		return te.listDirectory(input)
	case "search_files":
		return te.searchFiles(input)
	case "glob_files":
		return te.globFiles(input)
	case "web_search":
		return te.webSearch(input)
	case "web_fetch":
		return te.webFetch(input)
	case "linkedin_post":
		return te.linkedinPost(input)
	case "linkedin_read_comments":
		return te.linkedinReadComments(input)
	case "linkedin_post_comment":
		return te.linkedinPostComment(input)
	case "linkedin_reply_comment":
		return te.linkedinReplyComment(input)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func (te *ToolExecutor) readFile(input map[string]any) (string, error) {
	path, ok := input["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}
	return te.store.ReadFile(te.person, path)
}

func (te *ToolExecutor) writeFile(input map[string]any) (string, error) {
	path, ok := input["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}
	content, ok := input["content"].(string)
	if !ok {
		return "", fmt.Errorf("content is required")
	}
	err := te.store.WriteFile(te.person, path, content)
	if err != nil {
		return "", err
	}
	return "File written successfully", nil
}

func (te *ToolExecutor) listDirectory(input map[string]any) (string, error) {
	path, ok := input["path"].(string)
	if !ok {
		path = "."
	}
	entries, err := te.store.ListDir(te.person, path)
	if err != nil {
		return "", err
	}

	result, err := json.Marshal(entries)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func (te *ToolExecutor) globFiles(input map[string]any) (string, error) {
	pattern, ok := input["pattern"].(string)
	if !ok || strings.TrimSpace(pattern) == "" {
		return "", fmt.Errorf("pattern is required")
	}
	searchPath, ok := input["path"].(string)
	if !ok || strings.TrimSpace(searchPath) == "" {
		searchPath = "."
	}

	limit := 1000
	if rawLimit, ok := input["limit"]; ok {
		switch v := rawLimit.(type) {
		case float64:
			if int(v) > 0 {
				limit = int(v)
			}
		case int:
			if v > 0 {
				limit = v
			}
		}
	}

	fullPath, err := vault.ResolvePath(te.store.RootPath(), te.person, searchPath)
	if err != nil {
		return "", err
	}
	re, err := compileVaultGlob(pattern)
	if err != nil {
		return "", err
	}

	var matches []string

	err = filepath.Walk(fullPath, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Match against path relative to the search root, but return vault-relative paths.
		relForMatch, relErr := filepath.Rel(fullPath, p)
		if relErr != nil {
			return nil
		}
		relForMatch = filepath.ToSlash(relForMatch)

		if re.MatchString(relForMatch) {
			relVault, relVaultErr := filepath.Rel(filepath.Join(te.store.RootPath(), te.person), p)
			if relVaultErr != nil {
				return nil
			}
			matches = append(matches, filepath.ToSlash(relVault))
			if len(matches) >= limit {
				return filepath.SkipAll
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	result, err := json.Marshal(matches)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// compileVaultGlob converts a minimal glob syntax to a regex.
// Supported:
// - `*` matches any run of non-separator characters
// - `?` matches one non-separator character
// - `**` matches across separators
func compileVaultGlob(pattern string) (*regexp.Regexp, error) {
	pattern = strings.TrimSpace(pattern)
	pattern = filepath.ToSlash(pattern)

	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		switch ch {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				// Special-case "**/" to also match "no directory" (root-level files).
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					b.WriteString(`(?:.*/)?`)
					i += 2
				} else {
					b.WriteString(".*")
					i++
				}
			} else {
				b.WriteString(`[^/]*`)
			}
		case '?':
			b.WriteString(`[^/]`)
		case '.', '+', '(', ')', '|', '^', '$', '[', ']', '{', '}', '\\':
			b.WriteByte('\\')
			b.WriteByte(ch)
		default:
			b.WriteByte(ch)
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

func (te *ToolExecutor) searchFiles(input map[string]any) (string, error) {
	pattern, ok := input["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("pattern is required")
	}
	searchPath, ok := input["path"].(string)
	if !ok {
		searchPath = "."
	}

	// Build the full path to search
	fullPath, err := vault.ResolvePath(te.store.RootPath(), te.person, searchPath)
	if err != nil {
		return "", err
	}

	var results []map[string]any
	patternLower := strings.ToLower(pattern)

	err = filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't read
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Read file and search for pattern
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		contentLower := strings.ToLower(string(content))
		if strings.Contains(contentLower, patternLower) {
			// Get relative path from vault root
			relPath, _ := filepath.Rel(filepath.Join(te.store.RootPath(), te.person), path)

			// Find matching lines
			lines := strings.Split(string(content), "\n")
			var matches []map[string]any
			for i, line := range lines {
				if strings.Contains(strings.ToLower(line), patternLower) {
					matches = append(matches, map[string]any{
						"line_number": i + 1,
						"content":     line,
					})
				}
			}

			results = append(results, map[string]any{
				"file":    relPath,
				"matches": matches,
			})
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	result, err := json.Marshal(results)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func (te *ToolExecutor) webSearch(input map[string]any) (string, error) {
	query, ok := input["query"].(string)
	if !ok {
		return "", fmt.Errorf("query is required")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("query is required")
	}

	apiKey := strings.TrimSpace(os.Getenv("BRAVE_API_KEY"))
	if apiKey == "" {
		apiKey = defaultBraveAPIKey
	}

	maxResults := parsePositiveIntEnv("WEB_SEARCH_MAX_RESULTS", defaultWebSearchMaxResult)
	if maxResults > maxWebSearchResultCount {
		maxResults = maxWebSearchResultCount
	}
	cacheTTL := parsePositiveDurationEnv("WEB_SEARCH_CACHE_TTL", defaultWebSearchCacheTTL)
	timeout := parsePositiveDurationEnv("WEB_SEARCH_TIMEOUT", defaultWebSearchTimeout)

	cacheKey := strings.ToLower(strings.Join(strings.Fields(query), " "))
	if cached, ok := getWebSearchCache(cacheKey); ok {
		return cached, nil
	}

	searchURL, err := url.Parse(webSearchEndpoint)
	if err != nil {
		return "", fmt.Errorf("web search endpoint invalid: %w", err)
	}
	params := searchURL.Query()
	params.Set("q", query)
	params.Set("count", strconv.Itoa(maxResults))
	params.Set("safesearch", "moderate")
	searchURL.RawQuery = params.Encode()

	req, err := http.NewRequest(http.MethodGet, searchURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("web search request creation failed: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Subscription-Token", apiKey)
	req.Header.Set("User-Agent", "notes-editor-websearch/1.0")

	client := webSearchClient
	if client == nil || client.Timeout != timeout {
		client = &http.Client{Timeout: timeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("web search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("web search failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	var parsed struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 512*1024)).Decode(&parsed); err != nil {
		return "", fmt.Errorf("web search decode failed: %w", err)
	}

	results := make([]map[string]string, 0, maxResults)
	for _, r := range parsed.Web.Results {
		if len(results) >= maxResults {
			break
		}
		title := strings.TrimSpace(r.Title)
		link := strings.TrimSpace(r.URL)
		desc := strings.TrimSpace(collapseSpaces(r.Description))
		if desc != "" && len(desc) > maxWebSearchDescChars {
			desc = desc[:maxWebSearchDescChars] + "..."
		}
		if title == "" && link == "" && desc == "" {
			continue
		}
		results = append(results, map[string]string{
			"title":       title,
			"url":         link,
			"description": desc,
		})
	}

	payload := map[string]any{
		"query":   query,
		"count":   len(results),
		"results": results,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	wrapped := "<web_search_result_json>\n" + string(encoded) + "\n</web_search_result_json>"
	setWebSearchCache(cacheKey, wrapped, cacheTTL)
	return wrapped, nil
}

func (te *ToolExecutor) webFetch(input map[string]any) (string, error) {
	rawURL, ok := input["url"].(string)
	if !ok {
		return "", fmt.Errorf("url is required")
	}
	trimmedURL := strings.TrimSpace(rawURL)
	u, err := url.Parse(trimmedURL)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("invalid url: only https:// is allowed")
	}
	if u.User != nil {
		return "", fmt.Errorf("invalid url: credentials in url are not allowed")
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return "", fmt.Errorf("invalid url: missing host")
	}
	if err := validateWebFetchHostname(host); err != nil {
		return "", err
	}

	// Convenience: rewrite GitHub "blob" URLs to raw content URLs.
	// Users frequently paste https://github.com/<org>/<repo>/blob/<ref>/<path>.
	// This avoids returning the HTML UI, keeps content smaller, and reduces token waste.
	if strings.EqualFold(host, "github.com") {
		if ru, ok := rewriteGitHubBlobToRaw(u); ok {
			u = ru
			host = strings.TrimSpace(u.Hostname())
		}
	}

	const (
		maxRedirects       = 5
		timeout            = 10 * time.Second
		maxDownloadBytes   = 256 * 1024
		maxReturnedRunes   = 40000
		userAgent          = "notes-editor-webfetch/1.0"
		maxExtractedPrefix = 4 * 1024
	)

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           webFetchDialContext(timeout),
		DialTLSContext:        webFetchDialTLSContext(timeout),
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("stopped after too many redirects")
			}
			if req.URL.Scheme != "https" {
				return errors.New("redirect to non-https url is not allowed")
			}
			if req.URL.User != nil {
				return errors.New("redirect url contains credentials")
			}
			h := strings.TrimSpace(req.URL.Hostname())
			if h == "" {
				return errors.New("redirect url missing host")
			}
			return validateWebFetchHostname(h)
		},
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,text/plain,application/json;q=0.9,*/*;q=0.1")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("web fetch failed: %w", err)
	}
	defer resp.Body.Close()

	ct := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if ct == "" {
		// Best-effort sniffing when servers don't send Content-Type.
		// We'll still keep strict size limits.
		prefix, _ := io.ReadAll(io.LimitReader(resp.Body, maxExtractedPrefix))
		if len(prefix) > 0 {
			ct = http.DetectContentType(prefix)
			resp.Body = io.NopCloser(io.MultiReader(bytes.NewReader(prefix), resp.Body))
		}
	}

	limited := io.LimitReader(resp.Body, maxDownloadBytes+1)
	body, readErr := io.ReadAll(limited)
	if readErr != nil {
		return "", fmt.Errorf("web fetch read failed: %w", readErr)
	}
	downloadTruncated := len(body) > maxDownloadBytes
	if downloadTruncated {
		body = body[:maxDownloadBytes]
	}

	// Only allow a narrow set of content types to avoid binary garbage.
	ctLower := strings.ToLower(ct)
	var extracted string
	switch {
	case strings.Contains(ctLower, "text/html"):
		extracted = extractTextFromHTML(body)
	case strings.HasPrefix(ctLower, "text/") || strings.Contains(ctLower, "application/json"):
		extracted = string(body)
	default:
		return "", fmt.Errorf("unsupported content type: %s", ct)
	}
	extracted = normalizeWebFetchText(extracted)

	extractTruncated := false
	if utf8.RuneCountInString(extracted) > maxReturnedRunes {
		extracted = truncateRunes(extracted, maxReturnedRunes)
		extractTruncated = true
	}

	result := map[string]any{
		"url":            rawURL,
		"final_url":      resp.Request.URL.String(),
		"status":         resp.StatusCode,
		"content_type":   ct,
		"truncated":      downloadTruncated || extractTruncated,
		"returned_chars": len(extracted),
		"unsafe_content": extracted,
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	// Wrap in an obvious container and keep the fetched content inside JSON only.
	// This prevents the fetched content from "breaking out" of the wrapper.
	return "<web_fetch_result_json>\n" + string(encoded) + "\n</web_fetch_result_json>", nil
}

func validateWebFetchHostname(host string) error {
	hostLower := strings.ToLower(strings.TrimSpace(host))
	if hostLower == "localhost" || hostLower == "localhost." {
		return fmt.Errorf("host is not allowed")
	}
	if ip := net.ParseIP(hostLower); ip != nil {
		if isBlockedWebFetchIP(ip) {
			return fmt.Errorf("ip is not allowed")
		}
	}
	return nil
}

func webFetchDialContext(timeout time.Duration) func(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: timeout}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ip, err := resolveAllowedIP(ctx, host)
		if err != nil {
			return nil, err
		}
		return d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
	}
}

func webFetchDialTLSContext(timeout time.Duration) func(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: timeout}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ip, err := resolveAllowedIP(ctx, host)
		if err != nil {
			return nil, err
		}
		conn, err := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err != nil {
			return nil, err
		}
		// Use SNI/cert validation for the original hostname.
		cfg := &tls.Config{ServerName: host}
		tlsConn := tls.Client(conn, cfg)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return nil, err
		}
		return tlsConn, nil
	}
}

func resolveAllowedIP(ctx context.Context, host string) (net.IP, error) {
	h := strings.TrimSpace(host)
	if err := validateWebFetchHostname(h); err != nil {
		return nil, err
	}
	if ip := net.ParseIP(h); ip != nil {
		if isBlockedWebFetchIP(ip) {
			return nil, fmt.Errorf("ip is not allowed")
		}
		return ip, nil
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", h)
	if err != nil {
		return nil, err
	}
	for _, ip := range ips {
		if isBlockedWebFetchIP(ip) {
			continue
		}
		return ip, nil
	}
	return nil, fmt.Errorf("no allowed ip addresses for host")
}

func isBlockedWebFetchIP(ip net.IP) bool {
	// Normalize IPv4-mapped IPv6.
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	// Carrier-grade NAT 100.64.0.0/10 (not covered by IsPrivate).
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 100 && ip4[1]&0xC0 == 0x40 {
			return true
		}
	}
	// IPv6 unique local addresses fc00::/7.
	if ip.To4() == nil && len(ip) == net.IPv6len {
		if ip[0]&0xFE == 0xFC {
			return true
		}
	}
	return false
}

func extractTextFromHTML(body []byte) string {
	root, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		// Fallback: very naive strip if parsing fails.
		return string(body)
	}

	var b strings.Builder
	var walk func(n *html.Node, inScriptStyle bool)
	walk = func(n *html.Node, inScriptStyle bool) {
		if n.Type == html.ElementNode {
			switch strings.ToLower(n.Data) {
			case "script", "style", "noscript":
				inScriptStyle = true
			case "br", "p", "div", "li", "h1", "h2", "h3", "h4", "h5", "h6":
				b.WriteByte('\n')
			}
		}
		if n.Type == html.TextNode && !inScriptStyle {
			b.WriteString(n.Data)
			b.WriteByte('\n')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, inScriptStyle)
		}
	}
	walk(root, false)
	return b.String()
}

func normalizeWebFetchText(s string) string {
	// Collapse CRLF and excessive whitespace. Keep it simple: we want readable text,
	// not a perfect renderer.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	// Drop runs of empty lines.
	var out []string
	empty := 0
	for _, line := range lines {
		if line == "" {
			empty++
			if empty > 1 {
				continue
			}
			out = append(out, "")
			continue
		}
		empty = 0
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	count := 0
	for _, r := range s {
		b.WriteRune(r)
		count++
		if count >= max {
			break
		}
	}
	return b.String()
}

func rewriteGitHubBlobToRaw(u *url.URL) (*url.URL, bool) {
	if u == nil {
		return nil, false
	}
	// Expected: /<org>/<repo>/blob/<ref>/<path...>
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) < 5 {
		return nil, false
	}
	if parts[2] != "blob" {
		return nil, false
	}
	org := parts[0]
	repo := parts[1]
	ref := parts[3]
	path := strings.Join(parts[4:], "/")

	ru := &url.URL{
		Scheme: "https",
		Host:   "raw.githubusercontent.com",
		Path:   "/" + org + "/" + repo + "/" + ref + "/" + path,
	}
	return ru, true
}

func parsePositiveIntEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func parsePositiveDurationEnv(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := time.ParseDuration(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func getWebSearchCache(key string) (string, bool) {
	webSearchCacheMu.Lock()
	defer webSearchCacheMu.Unlock()

	entry, ok := webSearchCache[key]
	if !ok {
		return "", false
	}
	if time.Now().After(entry.expiresAt) {
		delete(webSearchCache, key)
		return "", false
	}
	return entry.payload, true
}

func setWebSearchCache(key, payload string, ttl time.Duration) {
	webSearchCacheMu.Lock()
	defer webSearchCacheMu.Unlock()

	now := time.Now()
	// Best-effort cleanup of expired entries to bound memory usage.
	for k, v := range webSearchCache {
		if now.After(v.expiresAt) {
			delete(webSearchCache, k)
		}
	}
	webSearchCache[key] = webSearchCacheEntry{
		expiresAt: now.Add(ttl),
		payload:   payload,
	}
}

func (te *ToolExecutor) linkedinPost(input map[string]any) (string, error) {
	if te.linkedin == nil {
		return "", fmt.Errorf("LinkedIn service not configured")
	}
	text, ok := input["text"].(string)
	if !ok {
		return "", fmt.Errorf("text is required")
	}
	return te.linkedin.CreatePost(text, te.person)
}

func (te *ToolExecutor) linkedinReadComments(input map[string]any) (string, error) {
	if te.linkedin == nil {
		return "", fmt.Errorf("LinkedIn service not configured")
	}
	postURN, ok := input["post_urn"].(string)
	if !ok {
		return "", fmt.Errorf("post_urn is required")
	}
	return te.linkedin.ReadComments(postURN)
}

func (te *ToolExecutor) linkedinPostComment(input map[string]any) (string, error) {
	if te.linkedin == nil {
		return "", fmt.Errorf("LinkedIn service not configured")
	}
	postURN, ok := input["post_urn"].(string)
	if !ok {
		return "", fmt.Errorf("post_urn is required")
	}
	text, ok := input["text"].(string)
	if !ok {
		return "", fmt.Errorf("text is required")
	}
	return te.linkedin.CreateComment(postURN, text, "", te.person)
}

func (te *ToolExecutor) linkedinReplyComment(input map[string]any) (string, error) {
	if te.linkedin == nil {
		return "", fmt.Errorf("LinkedIn service not configured")
	}
	postURN, ok := input["post_urn"].(string)
	if !ok {
		return "", fmt.Errorf("post_urn is required")
	}
	parentURN, ok := input["parent_comment_urn"].(string)
	if !ok {
		return "", fmt.Errorf("parent_comment_urn is required")
	}
	text, ok := input["text"].(string)
	if !ok {
		return "", fmt.Errorf("text is required")
	}
	return te.linkedin.CreateComment(postURN, text, parentURN, te.person)
}
