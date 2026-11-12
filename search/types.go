package search

type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type EmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

type Document struct {
	Name    string
	Content string
}

type Result struct {
	Doc   Document
	Score float64
	Rank  int
}
