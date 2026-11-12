package search

import (
	"os"
	"path/filepath"
	"strings"
)

func LoadCorpus() ([]Document, error) {
	dir := os.Getenv("DOCS_PATH")

	var corpus []Document
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".txt") {
			content, _ := os.ReadFile(path)
			corpus = append(corpus, Document{
				Name:    info.Name(),
				Content: string(content),
			})
		}
		return nil
	})
	return corpus, err
}
