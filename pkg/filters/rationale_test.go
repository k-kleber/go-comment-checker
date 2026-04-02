package filters

import (
	"testing"

	"github.com/k-kleber/go-comment-checker/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestRationaleFilter_ShouldSkip_BecauseComment_ReturnsTrue(t *testing.T) {
	filter := NewRationaleFilter()
	comment := models.CommentInfo{Text: "// using backoff because retries can cascade failures"}

	result := filter.ShouldSkip(comment)

	assert.True(t, result)
}

func TestRationaleFilter_ShouldSkip_IssueReferenceWorkaround_ReturnsTrue(t *testing.T) {
	filter := NewRationaleFilter()
	comment := models.CommentInfo{Text: "// workaround for parser limitation, see #847"}

	result := filter.ShouldSkip(comment)

	assert.True(t, result)
}

func TestRationaleFilter_ShouldSkip_ImportantConstraint_ReturnsTrue(t *testing.T) {
	filter := NewRationaleFilter()
	comment := models.CommentInfo{Text: "// important: this is needed to avoid double-processing"}

	result := filter.ShouldSkip(comment)

	assert.True(t, result)
}

func TestRationaleFilter_ShouldSkip_NarrationIncrement_ReturnsFalse(t *testing.T) {
	filter := NewRationaleFilter()
	comment := models.CommentInfo{Text: "// increment i"}

	result := filter.ShouldSkip(comment)

	assert.False(t, result)
}

func TestRationaleFilter_ShouldSkip_NarrationReturnResult_ReturnsFalse(t *testing.T) {
	filter := NewRationaleFilter()
	comment := models.CommentInfo{Text: "// return result"}

	result := filter.ShouldSkip(comment)

	assert.False(t, result)
}

func TestRationaleFilter_ShouldSkip_Docstring_ReturnsFalse(t *testing.T) {
	filter := NewRationaleFilter()
	comment := models.CommentInfo{Text: "\"\"\"explanation\"\"\"", IsDocstring: true}

	result := filter.ShouldSkip(comment)

	assert.False(t, result)
}
