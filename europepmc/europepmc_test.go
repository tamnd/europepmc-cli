package europepmc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testClient(baseURL string) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = baseURL
	cfg.Rate = 0
	return NewClient(cfg)
}

func TestUserAgentSent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte(`{"hitCount":0,"resultList":{"result":[]}}`))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	_, err := c.Search(context.Background(), "test", 5, "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSearchReturnsArticles(t *testing.T) {
	const body = `{
		"hitCount": 2,
		"resultList": {
			"result": [
				{
					"id": "37612345",
					"source": "MED",
					"pmid": "37612345",
					"doi": "10.1038/test",
					"title": "First Article",
					"authorString": "Doe J, Smith A, Johnson B.",
					"journalTitle": "Nature",
					"pubYear": "2023",
					"citedByCount": 42,
					"isOpenAccess": "Y"
				},
				{
					"id": "37612346",
					"source": "MED",
					"pmid": "37612346",
					"doi": "10.1038/test2",
					"title": "Second Article",
					"authorString": "Lee C.",
					"journalTitle": "Science",
					"pubYear": "2022",
					"citedByCount": 10,
					"isOpenAccess": "N"
				}
			]
		}
	}`

	var reqCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount++
		if reqCount > 1 {
			_, _ = w.Write([]byte(`{"hitCount":2,"resultList":{"result":[]}}`))
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	articles, err := c.Search(context.Background(), "CRISPR", 10, "cited")
	if err != nil {
		t.Fatal(err)
	}
	if len(articles) != 2 {
		t.Fatalf("got %d articles, want 2", len(articles))
	}
	if articles[0].Rank != 1 {
		t.Errorf("rank[0] = %d, want 1", articles[0].Rank)
	}
	if articles[0].PMID != "37612345" {
		t.Errorf("pmid[0] = %q, want 37612345", articles[0].PMID)
	}
	if !articles[0].OpenAccess {
		t.Error("article[0] should be open access")
	}
	if articles[1].OpenAccess {
		t.Error("article[1] should not be open access")
	}
	if articles[0].URL != "https://europepmc.org/article/MED/37612345" {
		t.Errorf("url[0] = %q", articles[0].URL)
	}
}

func TestArticleDetail(t *testing.T) {
	const body = `{
		"resultList": {
			"result": [
				{
					"id": "24906153",
					"source": "MED",
					"pmid": "24906153",
					"doi": "10.1126/science.1258096",
					"title": "Genome engineering using the CRISPR-Cas9 system",
					"authorString": "Ran FA, Hsu PD, Wright J, Agarwala V, Scott DA, Zhang F.",
					"journalTitle": "Science",
					"pubYear": "2013",
					"citedByCount": 51204,
					"isOpenAccess": "Y"
				}
			]
		}
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	art, err := c.Article(context.Background(), "MED", "24906153")
	if err != nil {
		t.Fatal(err)
	}
	if art.PMID != "24906153" {
		t.Errorf("pmid = %q, want 24906153", art.PMID)
	}
	if art.Citations != 51204 {
		t.Errorf("citations = %d, want 51204", art.Citations)
	}
	// 6 authors → truncated to 3 + et al.
	if art.Authors != "Ran FA, Hsu PD, Wright J et al." {
		t.Errorf("authors = %q", art.Authors)
	}
}

func TestCitationsReturnsArticles(t *testing.T) {
	const body = `{
		"citationList": {
			"citation": [
				{
					"id": "38901234",
					"source": "MED",
					"pmid": "38901234",
					"title": "Citing Article",
					"authorString": "Jones B, Lee C.",
					"journalTitle": "Cell",
					"pubYear": "2024"
				}
			]
		}
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	arts, err := c.Citations(context.Background(), "MED", "24906153", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 1 {
		t.Fatalf("got %d citations, want 1", len(arts))
	}
	if arts[0].PMID != "38901234" {
		t.Errorf("pmid = %q", arts[0].PMID)
	}
}

func TestReferencesReturnsArticles(t *testing.T) {
	const body = `{
		"referenceList": {
			"reference": [
				{
					"id": "23287718",
					"source": "MED",
					"pmid": "23287718",
					"title": "Referenced Article",
					"authorString": "Cong L, Ran FA, Cox D.",
					"journalTitle": "Science",
					"pubYear": "2013",
					"doi": "10.1126/science.1231143"
				}
			]
		}
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	arts, err := c.References(context.Background(), "MED", "24906153", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 1 {
		t.Fatalf("got %d references, want 1", len(arts))
	}
	if arts[0].DOI != "10.1126/science.1231143" {
		t.Errorf("doi = %q", arts[0].DOI)
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"hitCount":0,"resultList":{"result":[]}}`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := NewClient(cfg)

	start := time.Now()
	_, err := c.Search(context.Background(), "test", 5, "")
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestAuthorTruncation(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Doe J.", "Doe J"},
		{"Doe J, Smith A.", "Doe J, Smith A"},
		{"Doe J, Smith A, Johnson B.", "Doe J, Smith A, Johnson B"},
		{"Doe J, Smith A, Johnson B, Williams C.", "Doe J, Smith A, Johnson B et al."},
		{"A, B, C, D, E, F.", "A, B, C et al."},
		{"", ""},
	}
	for _, tc := range cases {
		got := truncateAuthors(tc.in)
		if got != tc.want {
			t.Errorf("truncateAuthors(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestResolveID(t *testing.T) {
	cases := []struct {
		raw    string
		source string
		id     string
	}{
		{"37612345", "MED", "37612345"},
		{"PMC1234567", "PMC", "PMC1234567"},
		{"10.1038/nature12345", "DOI", "10.1038/nature12345"},
	}
	for _, tc := range cases {
		src, id := ResolveID(tc.raw)
		if src != tc.source || id != tc.id {
			t.Errorf("ResolveID(%q) = (%q,%q), want (%q,%q)", tc.raw, src, id, tc.source, tc.id)
		}
	}
}
