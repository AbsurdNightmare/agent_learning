package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/richardwooding/bm25"
)

type Client struct {
	config Config
	http   *http.Client
}

func NewClient(config Config) *Client {
	return &Client{
		config: config,
		http:   &http.Client{},
	}
}

func (c *Client) send(ctx context.Context, input []string) (*http.Response, error) {
	payload, err := json.Marshal(EmbeddingRequest{
		Model: c.config.Model,
		Input: input,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.URL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	res, err := c.http.Do(req)
	if err == nil {
		return res, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	fmt.Printf("网络错误: %v，正在重试\n", err)

	return nil, errors.New("网络请求失败，超出最大重试次数")
}

func (c *Client) GetEmbedding(ctx context.Context, input []string) ([]float64, error) {
	res, err := c.send(ctx, input)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		content, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("API报错，状态码: %d, 内容: %s", res.StatusCode, content)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var embed EmbeddingResponse
	if err = json.Unmarshal(body, &embed); err != nil {
		fmt.Println("解析Embedding失败:", err)
		return nil, err
	}

	if len(embed.Data) == 0 {
		return nil, errors.New("API返回的 data 数组为空")
	}

	return embed.Data[0].Embedding, nil
}

func (c *Client) BM25Search(query string, corpus []Document) ([]Result, error) {
	queryTerms := Tokenize(query)

	var statsList []bm25.DocStats
	docMap := make(map[int]Document)
	for i, doc := range corpus {
		docTerms := Tokenize(doc.Content)
		stats := bm25.StatsForTokens(docTerms, queryTerms)
		statsList = append(statsList, stats)
		docMap[i] = doc
	}

	scorer := bm25.NewScorer(queryTerms, statsList, bm25.DefaultK1, bm25.DefaultB)

	var results []Result
	for i, stats := range statsList {
		score := scorer.Score(stats)
		if score > 0 {
			results = append(results, Result{Doc: docMap[i], Score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	for i := range results {
		results[i].Rank = i + 1
	}

	return results, nil
}

func (c *Client) VectorSearch(ctx context.Context, query string, corpus []Document) ([]Result, error) {
	queryVec, err := c.GetEmbedding(ctx, []string{query})
	if err != nil {
		fmt.Println("获取 query 向量失败")
		return nil, err
	}

	var results []Result
	for _, doc := range corpus {
		docVec, err := c.GetEmbedding(ctx, []string{doc.Content})
		if err != nil {
			fmt.Printf("获取 %s 的向量失败\n", doc.Name)
			continue
		}
		scores := CosineSimilarity(queryVec, docVec)
		results = append(results, Result{Doc: doc, Score: scores})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	for i := range results {
		results[i].Rank = i + 1
	}
	return results, nil
}

func (c *Client) HybridSearch(ctx context.Context, query string, corpus []Document) ([]Result, error) {
	bm25Results, err := c.BM25Search(query, corpus)
	if err != nil {
		fmt.Println("获取 BM25 结果失败")
		return nil, err
	}

	vectorResults, err := c.VectorSearch(ctx, query, corpus)
	if err != nil {
		fmt.Println("获取 Vector 结果失败")
		return nil, err
	}

	rrfScores := make(map[string]float64)
	docMap := make(map[string]Document)

	for _, res := range bm25Results {
		rrfScores[res.Doc.Name] += 1.0 / float64(60+res.Rank)
		docMap[res.Doc.Name] = res.Doc
	}

	for _, res := range vectorResults {
		rrfScores[res.Doc.Name] += 1.0 / float64(60+res.Rank)
		docMap[res.Doc.Name] = res.Doc
	}

	var finalResults []Result
	for name, score := range rrfScores {
		finalResults = append(finalResults, Result{
			Doc:   docMap[name],
			Score: score,
		})
	}

	sort.Slice(finalResults, func(i, j int) bool {
		return finalResults[i].Score > finalResults[j].Score
	})

	for i := range finalResults {
		finalResults[i].Rank = i + 1
	}

	return finalResults, nil
}
