package view

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/anton-dovnar/git-tree/structs"
	"github.com/go-git/go-git/v5/plumbing"

	svg "github.com/ajstarks/svgo"

	mapset "github.com/deckarep/golang-set/v2"
)

//go:embed resources/*
var resources embed.FS

type CommitMessage struct {
	Type       string `json:"type,omitempty"`
	Scope      string `json:"scope,omitempty"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	IsBreaking bool   `json:"is_breaking"`
}

type CommitData struct {
	Hash             string        `json:"hash"`
	Author           string        `json:"author"`
	Committer        string        `json:"committer"`
	Message          CommitMessage `json:"message"`
	AuthoredDate     string        `json:"authored_date"`
	CommittedDate    string        `json:"committed_date"`
	AuthoredDateDelta string       `json:"authored_date_delta"`
	CommittedDateDelta string      `json:"committed_date_delta"`
}

var issueRegex = regexp.MustCompile(`(\w+)#(\d+)`)

func prettyDate(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	}
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	if diff < 30*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	if diff < 365*24*time.Hour {
		months := int(diff.Hours() / (24 * 30))
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
	years := int(diff.Hours() / (24 * 365))
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

func issueLink(text string, ghSlug string) string {
	if ghSlug == "" {
		return text
	}
	replaced := issueRegex.ReplaceAllStringFunc(text, func(match string) string {
		parts := issueRegex.FindStringSubmatch(match)
		if len(parts) == 3 {
			org := parts[1]
			num := parts[2]
			if strings.HasPrefix(ghSlug, org+"/") {
				return fmt.Sprintf(`<a target="_blank" href="https://github.com/%s/issues/%s">%s#%s</a>`, ghSlug, num, org, num)
			}
			return fmt.Sprintf(`<a target="_blank" href="https://github.com/%s/issues/%s">%s#%s</a>`, org, num, org, num)
		}
		return match
	})
	return replaced
}

func parseCommitMessage(message string) (string, string, string) {
	colonIdx := strings.Index(message, ": ")
	if colonIdx < 0 {
		return "", "", message
	}

	prefix := strings.TrimSpace(message[:colonIdx])
	title := strings.TrimSpace(message[colonIdx+2:])

	parenIdx := strings.Index(prefix, "(")
	if parenIdx >= 0 {
		commitType := strings.TrimSpace(prefix[:parenIdx])
		rest := prefix[parenIdx+1:]
		closeParenIdx := strings.Index(rest, ")")
		if closeParenIdx >= 0 {
			scope := strings.TrimSpace(rest[:closeParenIdx])
			if strings.Contains(commitType, " ") {
				return "", "", message
			}
			return commitType, scope, title
		}
	}

	if strings.Contains(prefix, " ") {
		return "", "", message
	}
	return prefix, "", title
}

func GenerateCommitData(
	commits map[plumbing.Hash]*structs.CommitInfo,
	ghSlug string,
) map[string]CommitData {
	result := make(map[string]CommitData)

	for hash, ci := range commits {
		if ci == nil || ci.Commit == nil {
			continue
		}
		commit := ci.Commit
		fullMessage := commit.Message
		summary := strings.Split(fullMessage, "\n")[0]
		commitType, scope, title := parseCommitMessage(summary)

		body := ""
		lines := strings.Split(fullMessage, "\n")
		if len(lines) > 1 {
			bodyLines := lines[1:]
			for len(bodyLines) > 0 && strings.TrimSpace(bodyLines[0]) == "" {
				bodyLines = bodyLines[1:]
			}
			body = strings.Join(bodyLines, "\n")
			body = strings.TrimSpace(body)
			body = strings.ReplaceAll(body, " \n", " ")
			body = strings.ReplaceAll(body, " \r\n", " ")
		}

		title = issueLink(title, ghSlug)
		body = issueLink(body, ghSlug)

		authorHTML := fmt.Sprintf(`<a href="mailto:%s">%s</a>`, html.EscapeString(commit.Author.Email), html.EscapeString(commit.Author.Name))
		committerHTML := fmt.Sprintf(`<a href="mailto:%s">%s</a>`, html.EscapeString(commit.Committer.Email), html.EscapeString(commit.Committer.Name))

		authoredDate := commit.Author.When.Format(time.RFC3339)
		committedDate := commit.Committer.When.Format(time.RFC3339)
		authoredDateDelta := prettyDate(commit.Author.When)
		committedDateDelta := prettyDate(commit.Committer.When)
		isBreaking := strings.Contains(fullMessage, "BREAKING CHANGE:")

		hashStr := hash.String()
		if len(hashStr) > 7 {
			hashStr = hashStr[:7]
		}

		result[hash.String()] = CommitData{
			Hash:              hashStr,
			Author:            authorHTML,
			Committer:         committerHTML,
			Message: CommitMessage{
				Type:       commitType,
				Scope:      scope,
				Title:      title,
				Body:       body,
				IsBreaking: isBreaking,
			},
			AuthoredDate:      authoredDate,
			CommittedDate:     committedDate,
			AuthoredDateDelta: authoredDateDelta,
			CommittedDateDelta: committedDateDelta,
		}
	}

	return result
}

func getResource(name string) (string, error) {
	data, err := resources.ReadFile("resources/" + name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func replacePlaceholders(text string, placeholders map[string]string) string {
	result := text
	for key, value := range placeholders {
		placeholder := fmt.Sprintf("((%% %s %%))", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

func replaceReferences(text string) (string, error) {
	result := text
	begin := 0

	for {
		startIdx := strings.Index(result[begin:], "{{")
		if startIdx < 0 {
			break
		}
		startIdx += begin

		endIdx := strings.Index(result[startIdx+2:], "}}")
		if endIdx < 0 {
			break
		}
		endIdx += startIdx + 2
		reference := strings.TrimSpace(result[startIdx+2 : endIdx])

		resourceContent, err := getResource(reference)
		if err != nil {
			return "", fmt.Errorf("failed to load resource %s: %w", reference, err)
		}

		resourceContent, err = replaceReferences(resourceContent)
		if err != nil {
			return "", err
		}

		placeholder := result[startIdx : endIdx+2]
		result = strings.Replace(result, placeholder, resourceContent, 1)
		begin = startIdx + len(resourceContent)
	}

	return result, nil
}

func GenerateSVGString(
	commits map[plumbing.Hash]*structs.CommitInfo,
	positions map[plumbing.Hash][2]int,
	heads map[plumbing.Hash][]*plumbing.Reference,
	tags map[plumbing.Hash][]*plumbing.Reference,
	children map[plumbing.Hash]mapset.Set[plumbing.Hash],
) (string, error) {
	var buf bytes.Buffer
	canvas := svg.New(&buf)
	DrawRailway(canvas, commits, positions, heads, tags, children)
	return buf.String(), nil
}

func WriteHTML(
	w io.Writer,
	svgContent string,
	commitData map[string]CommitData,
	title string,
) error {
	template, err := getResource("html_template.html")
	if err != nil {
		return fmt.Errorf("failed to load HTML template: %w", err)
	}

	commitDataJSON, err := json.Marshal(commitData)
	if err != nil {
		return fmt.Errorf("failed to marshal commit data: %w", err)
	}

	if !strings.Contains(svgContent, `id="railway_svg"`) && !strings.Contains(svgContent, `id='railway_svg'`) {
		svgTagStart := strings.Index(svgContent, "<svg")
		if svgTagStart >= 0 {
			svgTagEnd := strings.Index(svgContent[svgTagStart:], ">")
			if svgTagEnd >= 0 {
				svgTagEnd += svgTagStart
				svgTag := svgContent[svgTagStart:svgTagEnd]
				if !strings.Contains(svgTag, "id=") {
					svgContent = svgContent[:svgTagEnd] + ` id="railway_svg"` + svgContent[svgTagEnd:]
				}
			}
		}
	}

	template, err = replaceReferences(template)
	if err != nil {
		return fmt.Errorf("failed to replace resource references: %w", err)
	}

	placeholders := map[string]string{
		"title": html.EscapeString(title),
		"svg":   svgContent,
		"data":  string(commitDataJSON),
	}
	template = replacePlaceholders(template, placeholders)
	_, err = w.Write([]byte(template))
	return err
}
