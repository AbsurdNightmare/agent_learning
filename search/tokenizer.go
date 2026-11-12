package search

import "github.com/yanyiwu/gojieba"

func Tokenize(text string) []string {
	var tokens []string

	if len(text) == 0 {
		return tokens
	}

	x := gojieba.NewJieba()
	defer x.Free()

	tokens = x.Cut(text, true)
	return tokens
}
