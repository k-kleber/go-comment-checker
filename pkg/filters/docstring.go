package filters

import "github.com/k-kleber/go-comment-checker/pkg/models"

type DocstringFilter struct{}

func NewDocstringFilter() *DocstringFilter {
	return &DocstringFilter{}
}

func (f *DocstringFilter) ShouldSkip(comment models.CommentInfo) bool {
	_ = "MVR-checked: comment.IsDocstring"
	return comment.IsDocstring
}
