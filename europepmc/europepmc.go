// Package europepmc is the library behind the epmc command: the HTTP client,
// request shaping, and the typed data models for Europe PMC.
//
// The REST API at https://www.ebi.ac.uk/europepmc/webservices/rest is open and
// requires no authentication. The client sets a real User-Agent, paces
// requests, and retries transient 429/5xx failures with exponential backoff.
package europepmc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to Europe PMC.
const DefaultUserAgent = "epmc/dev (+https://github.com/tamnd/europepmc-cli)"

// ErrNotFound is returned when the API returns no results for a lookup.
var ErrNotFound = errors.New("not found")

// Config holds constructor parameters.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://www.ebi.ac.uk/europepmc/webservices/rest",
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the Europe PMC REST API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
	rate       time.Duration
	retries    int
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client with the given config.
func NewClient(cfg Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		userAgent:  cfg.UserAgent,
		rate:       cfg.Rate,
		retries:    cfg.Retries,
	}
}

// ── public API methods ────────────────────────────────────────────────────────

// Search queries the /search endpoint and returns up to limit articles sorted
// by sort. sort values: "cited" → CITED desc, "date" → P_PDATE_D desc,
// anything else → relevance (no sort parameter).
func (c *Client) Search(ctx context.Context, query string, limit int, sort string) ([]Article, error) {
	if limit <= 0 {
		limit = 20
	}

	sortParam := sortAPIParam(sort)

	var out []Article
	page := 0
	pageSize := limit
	if pageSize > 100 {
		pageSize = 100
	}

	for {
		params := url.Values{}
		params.Set("query", query)
		params.Set("format", "json")
		params.Set("pageSize", fmt.Sprintf("%d", pageSize))
		params.Set("page", fmt.Sprintf("%d", page))
		if sortParam != "" {
			params.Set("sort", sortParam)
		}

		rawURL := c.baseURL + "/search?" + params.Encode()
		var resp searchResp
		if err := c.getJSON(ctx, rawURL, &resp); err != nil {
			return out, err
		}
		for _, r := range resp.ResultList.Result {
			out = append(out, toArticle(r, len(out)+1))
			if len(out) >= limit {
				return out, nil
			}
		}
		if len(resp.ResultList.Result) == 0 {
			break
		}
		page++
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// Article fetches a single article by source and id.
func (c *Client) Article(ctx context.Context, source, id string) (Article, error) {
	rawURL := fmt.Sprintf("%s/article/%s/%s?format=json",
		c.baseURL,
		url.PathEscape(source),
		url.PathEscape(id),
	)
	var resp articleDetailResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return Article{}, err
	}
	if len(resp.ResultList.Result) == 0 {
		return Article{}, ErrNotFound
	}
	return toArticle(resp.ResultList.Result[0], 1), nil
}

// ArticleByID resolves a user-supplied id string (PMID, PMC ID, or DOI)
// to an Article.
//
// Resolution rules:
//   - Starts with "PMC" → source=PMC, id=numeric portion.
//   - Contains "/" → treat as DOI; search DOI:{id}.
//   - Otherwise → source=MED, id as-is.
func (c *Client) ArticleByID(ctx context.Context, rawID string) (Article, error) {
	source, id := ResolveID(rawID)
	if source == "DOI" {
		// DOI resolution: search for it and take the first hit.
		hits, err := c.Search(ctx, "DOI:"+id, 1, "")
		if err != nil {
			return Article{}, err
		}
		if len(hits) == 0 {
			return Article{}, ErrNotFound
		}
		return hits[0], nil
	}
	return c.Article(ctx, source, id)
}

// Citations lists articles that cite the given article.
func (c *Client) Citations(ctx context.Context, source, id string, limit int) ([]Article, error) {
	if limit <= 0 {
		limit = 25
	}
	pageSize := limit
	if pageSize > 100 {
		pageSize = 100
	}
	rawURL := fmt.Sprintf("%s/article/%s/%s/citations?format=json&pageSize=%d",
		c.baseURL,
		url.PathEscape(source),
		url.PathEscape(id),
		pageSize,
	)
	var resp citationResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	items := resp.CitationList.Citation
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]Article, len(items))
	for i, r := range items {
		out[i] = toArticle(r, i+1)
	}
	return out, nil
}

// References lists articles referenced by the given article.
func (c *Client) References(ctx context.Context, source, id string, limit int) ([]Article, error) {
	if limit <= 0 {
		limit = 25
	}
	pageSize := limit
	if pageSize > 100 {
		pageSize = 100
	}
	rawURL := fmt.Sprintf("%s/article/%s/%s/references?format=json&pageSize=%d",
		c.baseURL,
		url.PathEscape(source),
		url.PathEscape(id),
		pageSize,
	)
	var resp referenceResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	items := resp.ReferenceList.Reference
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]Article, len(items))
	for i, r := range items {
		out[i] = toArticle(r, i+1)
	}
	return out, nil
}

// ResolveID parses a raw user id into (source, id) ready for API calls.
// Returns ("DOI", doi) when the id contains "/" so the caller can handle
// DOI resolution via search.
func ResolveID(raw string) (source, id string) {
	if strings.HasPrefix(raw, "PMC") && len(raw) > 3 {
		return "PMC", raw
	}
	if strings.Contains(raw, "/") {
		return "DOI", raw
	}
	return "MED", raw
}

// ── HTTP internals ────────────────────────────────────────────────────────────

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

func sortAPIParam(sort string) string {
	switch sort {
	case "cited":
		return "CITED desc"
	case "date":
		return "P_PDATE_D desc"
	default:
		return ""
	}
}
