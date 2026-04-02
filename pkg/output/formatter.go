package output

import (
	"fmt"
	"strings"

	"github.com/k-kleber/go-comment-checker/pkg/filters"
	"github.com/k-kleber/go-comment-checker/pkg/models"
)

// FormatHookMessage formats comment detection results for Claude Code hooks.
// Groups comments by file path and builds complete error message with
// instructions and XML blocks for each file.
// If customPrompt is provided, it replaces the default message template.
// Use {{comments}} placeholder in customPrompt to insert detected comments XML.
// Returns formatted hook error message, or empty string if no comments provided.
func FormatHookMessage(comments []models.CommentInfo, customPrompt string) string {
	if len(comments) == 0 {
		return ""
	}

	// Group comments by file path
	byFile := make(map[string][]models.CommentInfo)
	fileOrder := make([]string, 0)
	for _, comment := range comments {
		if _, exists := byFile[comment.FilePath]; !exists {
			fileOrder = append(fileOrder, comment.FilePath)
		}
		byFile[comment.FilePath] = append(byFile[comment.FilePath], comment)
	}

	// Build comments XML
	var commentsXML strings.Builder
	for _, filePath := range fileOrder {
		fileComments := byFile[filePath]
		commentsXML.WriteString(BuildCommentsXML(fileComments, filePath))
		commentsXML.WriteString("\n")
	}

	// If custom prompt is provided, use it with {{comments}} replacement
	if customPrompt != "" {
		return strings.ReplaceAll(customPrompt, "{{comments}}", commentsXML.String())
	}

	// Default message template
	// Detect agent memo comments
	agentMemoFilter := filters.NewAgentMemoFilter()
	var agentMemoComments []models.CommentInfo
	for _, comment := range comments {
		if agentMemoFilter.IsAgentMemo(comment) {
			agentMemoComments = append(agentMemoComments, comment)
		}
	}
	hasAgentMemo := len(agentMemoComments) > 0

	var sb strings.Builder

	// Header
	if hasAgentMemo {
		sb.WriteString("🚨 AGENT MEMO COMMENT DETECTED - CODE SMELL ALERT 🚨\n\n")
	} else {
		sb.WriteString("⚠️  POTENTIAL LOW-VALUE COMMENT DETECTED ⚠️\n\n")
	}

	// Agent memo warning (if detected)
	if hasAgentMemo {
		sb.WriteString("⚠️  AGENT MEMO COMMENTS DETECTED - THIS IS A CODE SMELL  ⚠️\n\n")
		sb.WriteString("You left \"memo-style\" comments that describe WHAT you changed or HOW you implemented something.\n")
		sb.WriteString("These are typically signs of an AI agent leaving notes for itself or the user.\n\n")
		sb.WriteString("Examples of agent memo patterns detected:\n")
		sb.WriteString("  - \"Changed from X to Y\", \"Modified to...\", \"Updated from...\"\n")
		sb.WriteString("  - \"Added new...\", \"Removed...\", \"Refactored...\"\n")
		sb.WriteString("  - \"This implements...\", \"Here we...\", \"Now this...\"\n")
		sb.WriteString("  - \"Note:\", \"Implementation of...\"\n")
		sb.WriteString("  - Korean: \"여기서 변경됨\", \"구현함\", \"추가함\", \"수정됨\"\n\n")
		sb.WriteString("WHY THIS IS BAD:\n")
		sb.WriteString("  1. Code should be self-documenting - if you need to explain what changed, the code isn't clear enough\n")
		sb.WriteString("  2. These comments become outdated instantly and mislead future readers\n")
		sb.WriteString("  3. Git history already tracks what changed - comments duplicate this poorly\n")
		sb.WriteString("  4. It's a sign of rushing without proper refactoring\n\n")
		sb.WriteString("ACTION REQUIRED:\n")
		sb.WriteString("  -> REMOVE these memo comments immediately\n")
		sb.WriteString("  -> If the code needs explanation, refactor it to be clearer instead\n")
		sb.WriteString("  -> Use meaningful variable/function names that explain intent\n")
		sb.WriteString("  -> Let git commit messages document the \"what\" and \"why\"\n\n")
		sb.WriteString("Detected agent memo comments:\n")
		for _, memo := range agentMemoComments {
			sb.WriteString(fmt.Sprintf("  - Line %d: %s\n", memo.LineNumber, strings.TrimSpace(memo.Text)))
		}
		sb.WriteString("\n---\n\n")
	}

	// Guidelines
	sb.WriteString("Your recent changes contain comments that may be redundant or low-value, triggering this hook.\n")
	sb.WriteString("While useful documentation is encouraged, redundant narration and memo comments should be avoided.\n")
	sb.WriteString("Please review the detected comments and take action according to the priority guidelines below:\n\n")

	sb.WriteString("CRITICAL WARNING: This hook message MUST NEVER be ignored, even if you receive it multiple times.\n")
	sb.WriteString("You MUST provide corresponding explanation or action for EACH occurrence of this message.\n")
	sb.WriteString("Ignoring this message or failing to respond appropriately is strictly prohibited.\n\n")

	sb.WriteString("PRIORITY-BASED ACTION GUIDELINES:\n\n")

	sb.WriteString("1. This is a comment/docstring that already existed before\n")
	sb.WriteString("\t-> Explain to the user that this is an existing comment/docstring and proceed (justify it)\n\n")

	sb.WriteString("2. This is a newly written comment: but it's in given, when, then format\n")
	sb.WriteString("\t-> Tell the user it's a BDD comment and proceed (justify it)\n")
	sb.WriteString("\t-> Note: This applies to comments only, not docstrings\n\n")

	sb.WriteString("3. This is a newly written docstring for a public API or declaration\n")
	sb.WriteString("\t-> Docstrings are EXPECTED for public APIs/declarations and do NOT trigger violations by default.\n")
	sb.WriteString("\t-> If you see this, ensure the docstring provides meaningful context beyond the function name.\n\n")

	sb.WriteString("4. This is a newly written comment/docstring: redundant narration or memo\n")
	sb.WriteString("\t-> Apologize to the user and remove the comment/docstring.\n")
	sb.WriteString("\t-> Make the code itself clearer so it can be understood without comments/docstrings.\n")
	sb.WriteString("\t-> Examples of redundant narration: // increment i, // return result, // calling function X\n\n")

	sb.WriteString("MANDATORY REQUIREMENT: You must acknowledge this hook message and take one of the above actions.\n")
	sb.WriteString("Review in the above priority order and take the corresponding action EVERY TIME this appears.\n\n")

	sb.WriteString("REMINDER: These rules apply to ALL your future code, not just this specific edit. Always be deliberate and cautious when writing comments - only add them when absolutely necessary.\n\n")

	sb.WriteString("Detected comments/docstrings:\n")
	sb.WriteString(commentsXML.String())

	return sb.String()
}
