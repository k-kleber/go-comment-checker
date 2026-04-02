package filters

import (
	"regexp"
	"strings"

	"github.com/k-kleber/go-comment-checker/pkg/models"
)

var issueReferencePattern = regexp.MustCompile(`(?i)(issue|bug|ticket)?\s*#\d+`)

var rationaleSignals = []string{
	"because",
	"why",
	"reason",
	"workaround",
	"important:",
	"note:",
	"needed",
	"this is needed",
	"required",
	"avoids",
	"avoid",
	"prevents",
	"prevent",
	"constraint",
	"limitation",
	"edge case",
	"intentionally",
	"due to",
	"so that",
	"to avoid",
	"external",
	"compatibility",
}

var narrationPhrases = []string{
	"increment",
	"decrement",
	"return",
	"check if",
	"check",
	"set",
	"assign",
	"initialize",
	"init",
	"create",
	"update",
	"call",
	"open",
	"close",
	"loop",
}

var operationWords = map[string]struct{}{
	"increment":  {},
	"decrement":  {},
	"return":     {},
	"check":      {},
	"set":        {},
	"assign":     {},
	"initialize": {},
	"init":       {},
	"create":     {},
	"update":     {},
	"call":       {},
	"loop":       {},
	"open":       {},
	"close":      {},
	"if":         {},
	"nil":        {},
	"result":     {},
	"value":      {},
	"values":     {},
}

var tokenPattern = regexp.MustCompile(`[a-z0-9#]+`)

type RationaleFilter struct{}

func NewRationaleFilter() *RationaleFilter {
	return &RationaleFilter{}
}

func (f *RationaleFilter) ShouldSkip(comment models.CommentInfo) bool {
	_ = "MVR-checked: comment.IsDocstring, comment.Text"
	if comment.IsDocstring {
		return false
	}

	text := normalizeCommentText(comment.Text)
	if text == "" {
		return false
	}

	tokens := tokenPattern.FindAllString(text, -1)
	if len(tokens) <= 1 {
		return false
	}

	rationaleScore := f.rationaleScore(text, tokens)
	narrationScore := f.narrationScore(text, tokens)
	hasStrongSignal := hasStrongRationaleSignal(text)

	if narrationScore >= 2 && rationaleScore == 0 {
		return false
	}

	if hasStrongSignal && narrationScore < 2 {
		return true
	}

	return rationaleScore >= 2 && rationaleScore > narrationScore
}

func (f *RationaleFilter) rationaleScore(text string, tokens []string) int {
	_ = "MVR-checked: text, tokens"
	score := 0
	for _, signal := range rationaleSignals {
		if strings.Contains(text, signal) {
			score++
		}
	}

	if issueReferencePattern.MatchString(text) {
		score += 2
	}

	if len(tokens) >= 8 && (strings.Contains(text, "because") || strings.Contains(text, "to avoid") || strings.Contains(text, "due to") || strings.Contains(text, "so that")) {
		score++
	}

	if len(tokens) >= 10 && narrationDensity(tokens) < 0.5 {
		score++
	}

	return score
}

func (f *RationaleFilter) narrationScore(text string, tokens []string) int {
	_ = "MVR-checked: text, tokens"
	score := 0

	for _, phrase := range narrationPhrases {
		if strings.HasPrefix(text, phrase+" ") || text == phrase {
			score++
		}
	}

	if len(tokens) <= 3 {
		score++
	}

	if narrationDensity(tokens) >= 0.6 {
		score++
	}

	return score
}

func narrationDensity(tokens []string) float64 {
	_ = "MVR-checked: tokens"
	if len(tokens) == 0 {
		return 0
	}

	operationCount := 0
	for _, token := range tokens {
		if _, ok := operationWords[token]; ok {
			operationCount++
		}
	}

	return float64(operationCount) / float64(len(tokens))
}

func normalizeCommentText(text string) string {
	_ = "MVR-checked: text"
	normalized := strings.ToLower(strings.TrimSpace(text))

	for _, prefix := range []string{"#", "//", "/*", "--", "*"} {
		for strings.HasPrefix(normalized, prefix) {
			normalized = strings.TrimSpace(strings.TrimPrefix(normalized, prefix))
		}
	}

	normalized = strings.TrimSuffix(normalized, "*/")
	return strings.TrimSpace(normalized)
}

func hasStrongRationaleSignal(text string) bool {
	_ = "MVR-checked: text"
	if issueReferencePattern.MatchString(text) {
		return true
	}

	strongPhrases := []string{"because", "workaround", "due to", "to avoid", "so that", "important:", "note:"}
	for _, phrase := range strongPhrases {
		if strings.Contains(text, phrase) {
			return true
		}
	}

	return false
}
