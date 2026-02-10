package claude

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"notes-editor/internal/vault"
)

type availableSkill struct {
	Name        string
	Description string
	Location    string
}

// BuildAvailableSkillsPromptAddon scans agent/skills/*.md under the person's vault and returns
// a small XML index to append to the system prompt. The model can then decide to read the full
// skill file when needed.
func BuildAvailableSkillsPromptAddon(store *vault.Store, person string) string {
	if store == nil {
		return ""
	}

	entries, err := store.ListDir(person, "agent/skills")
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		// Non-fatal: skip skills if directory can't be read.
		return ""
	}

	var skills []availableSkill
	for _, e := range entries {
		if e.IsDir {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(e.Name), ".md") {
			continue
		}

		location := filepath.ToSlash(e.Path)
		raw, err := store.ReadFile(person, location)
		if err != nil {
			continue
		}

		name := strings.TrimSuffix(e.Name, filepath.Ext(e.Name))
		desc := extractSkillDescription(raw)
		if strings.TrimSpace(desc) == "" {
			desc = "Skill instructions."
		}
		desc = collapseSpaces(desc)
		if len(desc) > 200 {
			desc = desc[:200] + "..."
		}

		skills = append(skills, availableSkill{
			Name:        name,
			Description: desc,
			Location:    location,
		})
	}

	if len(skills) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\nThe following skills provide specialized instructions for specific tasks.\n")
	b.WriteString("Use the read tool to load a skill's file when the task matches its description.\n\n")
	b.WriteString("<available_skills>\n")
	for _, s := range skills {
		b.WriteString("  <skill>\n")
		b.WriteString(fmt.Sprintf("    <name>%s</name>\n", escapeXML(s.Name)))
		b.WriteString(fmt.Sprintf("    <description>%s</description>\n", escapeXML(s.Description)))
		b.WriteString(fmt.Sprintf("    <location>%s</location>\n", escapeXML(s.Location)))
		b.WriteString("  </skill>\n")
	}
	b.WriteString("</available_skills>")
	return b.String()
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

func extractSkillDescription(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// Minimal YAML frontmatter parse: if the file starts with '---', scan key/value pairs
	// until the next '---'. Prefer "description:".
	if strings.HasPrefix(raw, "---\n") || strings.HasPrefix(raw, "---\r\n") {
		sc := bufio.NewScanner(strings.NewReader(raw))
		if !sc.Scan() { // first ---
			return ""
		}
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "---" {
				break
			}
			if !strings.Contains(line, ":") {
				continue
			}
			key, val, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			key = strings.ToLower(strings.TrimSpace(key))
			val = strings.TrimSpace(val)
			val = strings.Trim(val, "\"'")
			if key == "description" {
				return val
			}
		}
	}

	// Fallback: first non-empty, non-heading line.
	lines := strings.Split(raw, "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		if strings.HasPrefix(ln, "#") {
			continue
		}
		return ln
	}
	return ""
}

func collapseSpaces(s string) string {
	// Keep it tiny: collapse consecutive whitespace to single spaces.
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

