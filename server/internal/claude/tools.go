package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"notes-editor/internal/linkedin"
	"notes-editor/internal/vault"
)

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
	// Web search would require external API integration
	// For now, return a placeholder indicating the feature isn't available
	return fmt.Sprintf("Web search for '%s' is not yet implemented. This would require integration with a search API.", query), nil
}

func (te *ToolExecutor) webFetch(input map[string]any) (string, error) {
	url, ok := input["url"].(string)
	if !ok {
		return "", fmt.Errorf("url is required")
	}
	// Basic URL validation
	if !regexp.MustCompile(`^https?://`).MatchString(url) {
		return "", fmt.Errorf("invalid URL: must start with http:// or https://")
	}
	// Web fetch would require HTTP client with proper handling
	// For now, return a placeholder
	return fmt.Sprintf("Web fetch for '%s' is not yet implemented. This would require proper HTTP client integration.", url), nil
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
