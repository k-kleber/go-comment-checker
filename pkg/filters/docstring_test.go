package filters

import (
	"testing"

	"github.com/k-kleber/go-comment-checker/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestDocstringFilter_ShouldSkip_Docstring_ReturnsTrue(t *testing.T) {
	filter := NewDocstringFilter()
	comment := models.CommentInfo{
		Text:        "\"\"\"module docs\"\"\"",
		CommentType: models.CommentTypeDocstring,
		IsDocstring: true,
	}

	result := filter.ShouldSkip(comment)

	assert.True(t, result)
}

func TestDocstringFilter_ShouldSkip_RegularComment_ReturnsFalse(t *testing.T) {
	filter := NewDocstringFilter()
	comment := models.CommentInfo{
		Text:        "# regular comment",
		CommentType: models.CommentTypeLine,
		IsDocstring: false,
	}

	result := filter.ShouldSkip(comment)

	assert.False(t, result)
}
