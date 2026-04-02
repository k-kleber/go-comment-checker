package core

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/k-kleber/go-comment-checker/pkg/models"
)

// CommentDetector detects comments in source code using tree-sitter.
type CommentDetector struct {
	registry *LanguageRegistry
}

// NewCommentDetector creates a new CommentDetector instance.
func NewCommentDetector() *CommentDetector {
	return &CommentDetector{
		registry: NewLanguageRegistry(),
	}
}

// Detect extracts comments from the given source code.
func (d *CommentDetector) Detect(content, filePath string) []models.CommentInfo {
	_ = "MVR-checked: content,filePath"
	ext := strings.TrimPrefix(filepath.Ext(filePath), ".")
	if ext == "" {
		// Handle files like "Dockerfile"
		ext = strings.ToLower(filepath.Base(filePath))
	}

	langName := d.registry.GetLanguageName(ext)
	if langName == "" {
		return nil
	}

	lang := GetLanguage(langName)
	if lang == nil {
		return nil
	}

	parser := sitter.NewParser()
	parser.SetLanguage(lang)

	sourceCode := []byte(content)
	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return nil
	}
	defer tree.Close()

	queryPattern := QueryTemplates[langName]
	if queryPattern == "" {
		queryPattern = "(comment) @comment"
	}

	query, err := sitter.NewQuery([]byte(queryPattern), lang)
	if err != nil {
		return nil
	}
	defer query.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	qc.Exec(query, tree.RootNode())

	var comments []models.CommentInfo
	for {
		match, ok := qc.NextMatch()
		if !ok {
			break
		}
		for _, capture := range match.Captures {
			node := capture.Node
			text := node.Content(sourceCode)
			lineNumber := int(node.StartPoint().Row) + 1

			commentType := d.determineCommentType(text, node.Type())
			isDocstring := commentType == models.CommentTypeDocstring

			comments = append(comments, models.CommentInfo{
				Text:        text,
				LineNumber:  lineNumber,
				FilePath:    filePath,
				CommentType: commentType,
				IsDocstring: isDocstring,
			})
		}
	}

	docstrings := d.detectDocstrings(sourceCode, filePath, lang, langName)
	comments = d.classifyDocstrings(comments, docstrings)

	return comments
}

func (d *CommentDetector) classifyDocstrings(comments, docstrings []models.CommentInfo) []models.CommentInfo {
	if len(docstrings) == 0 {
		return comments
	}

	docstringByKey := make(map[string]models.CommentInfo, len(docstrings))
	for _, doc := range docstrings {
		docstringByKey[commentIdentityKey(doc)] = doc
	}

	for i := range comments {
		key := commentIdentityKey(comments[i])
		if _, ok := docstringByKey[key]; ok {
			comments[i].CommentType = models.CommentTypeDocstring
			comments[i].IsDocstring = true
			delete(docstringByKey, key)
		}
	}

	for _, doc := range docstringByKey {
		comments = append(comments, doc)
	}

	return comments
}

func commentIdentityKey(comment models.CommentInfo) string {
	return strings.TrimSpace(comment.Text) + "|" + strconv.Itoa(comment.LineNumber)
}

// detectDocstrings extracts docstrings using language-specific queries.
func (d *CommentDetector) detectDocstrings(sourceCode []byte, filePath string, lang *sitter.Language, langName string) []models.CommentInfo {
	_ = "MVR-checked: sourceCode,filePath,lang,langName"
	docQuery, ok := DocstringQueries[langName]
	if !ok {
		return nil
	}

	parser := sitter.NewParser()
	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return nil
	}
	defer tree.Close()

	query, err := sitter.NewQuery([]byte(docQuery), lang)
	if err != nil {
		return nil
	}
	defer query.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	qc.Exec(query, tree.RootNode())

	var docstrings []models.CommentInfo
	for {
		match, ok := qc.NextMatch()
		if !ok {
			break
		}
		for _, capture := range match.Captures {
			node := capture.Node
			text := node.Content(sourceCode)
			if !d.matchesDocstringPolicy(text, langName, sourceCode, node.EndByte()) {
				continue
			}
			lineNumber := int(node.StartPoint().Row) + 1

			docstrings = append(docstrings, models.CommentInfo{
				Text:        text,
				LineNumber:  lineNumber,
				FilePath:    filePath,
				CommentType: models.CommentTypeDocstring,
				IsDocstring: true,
			})
		}
	}

	return docstrings
}

func (d *CommentDetector) matchesDocstringPolicy(text, langName string, sourceCode []byte, nodeEndByte uint32) bool {
	_ = "MVR-checked: text,langName,sourceCode,nodeEndByte"
	stripped := strings.TrimSpace(text)

	if stripped == "" {
		return false
	}

	switch langName {
	case "javascript", "typescript", "tsx", "java":
		if !strings.HasPrefix(stripped, "/**") {
			return false
		}
		return d.hasDeclarationAfter(sourceCode, nodeEndByte, langName)
	default:
		return true
	}
}

func (d *CommentDetector) hasDeclarationAfter(sourceCode []byte, nodeEndByte uint32, langName string) bool {
	if int(nodeEndByte) >= len(sourceCode) {
		return false
	}

	remainder := strings.TrimSpace(string(sourceCode[nodeEndByte:]))
	if remainder == "" {
		return false
	}

	firstLine := remainder
	if idx := strings.IndexByte(firstLine, '\n'); idx >= 0 {
		firstLine = firstLine[:idx]
	}
	firstLine = strings.TrimSpace(firstLine)
	if firstLine == "" {
		return false
	}

	if langName == "java" {
		return strings.HasPrefix(firstLine, "public ") ||
			strings.HasPrefix(firstLine, "private ") ||
			strings.HasPrefix(firstLine, "protected ") ||
			strings.HasPrefix(firstLine, "static ") ||
			strings.HasPrefix(firstLine, "final ") ||
			strings.HasPrefix(firstLine, "abstract ") ||
			strings.HasPrefix(firstLine, "class ") ||
			strings.HasPrefix(firstLine, "interface ") ||
			strings.HasPrefix(firstLine, "enum ") ||
			strings.HasPrefix(firstLine, "@")
	}

	if strings.HasPrefix(firstLine, "export ") ||
		strings.HasPrefix(firstLine, "default ") ||
		strings.HasPrefix(firstLine, "async function ") ||
		strings.HasPrefix(firstLine, "function ") ||
		strings.HasPrefix(firstLine, "class ") ||
		strings.HasPrefix(firstLine, "interface ") ||
		strings.HasPrefix(firstLine, "type ") ||
		strings.HasPrefix(firstLine, "const ") ||
		strings.HasPrefix(firstLine, "let ") ||
		strings.HasPrefix(firstLine, "var ") ||
		strings.HasPrefix(firstLine, "declare ") ||
		strings.HasPrefix(firstLine, "public ") ||
		strings.HasPrefix(firstLine, "private ") ||
		strings.HasPrefix(firstLine, "protected ") ||
		strings.HasPrefix(firstLine, "readonly ") ||
		strings.HasPrefix(firstLine, "abstract ") {
		return true
	}

	openParen := strings.IndexByte(firstLine, '(')
	closeParen := strings.IndexByte(firstLine, ')')
	if openParen > 0 && closeParen > openParen && strings.Contains(firstLine[closeParen:], "{") {
		return true
	}

	return false
}

// determineCommentType determines the type of comment based on its text and node type.
func (d *CommentDetector) determineCommentType(text, nodeType string) models.CommentType {
	stripped := strings.TrimSpace(text)

	// Check node type first (for Rust)
	if nodeType == "line_comment" {
		return models.CommentTypeLine
	}
	if nodeType == "block_comment" {
		return models.CommentTypeBlock
	}

	// Check for docstrings
	if strings.HasPrefix(stripped, `"""`) || strings.HasPrefix(stripped, "'''") {
		return models.CommentTypeDocstring
	}

	// Check for line comments
	if strings.HasPrefix(stripped, "//") || strings.HasPrefix(stripped, "#") {
		return models.CommentTypeLine
	}

	// Check for block comments
	if strings.HasPrefix(stripped, "/*") || strings.HasPrefix(stripped, "<!--") || strings.HasPrefix(stripped, "--") {
		return models.CommentTypeBlock
	}

	return models.CommentTypeLine
}
