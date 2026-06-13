package europepmc

import "strings"

// Article is the record emitted for all article operations: search, article
// detail, citations, and references.
type Article struct {
	Rank       int    `json:"rank"`
	PMID       string `json:"pmid"`
	Source     string `json:"source"`
	DOI        string `json:"doi"`
	Title      string `json:"title"`
	Authors    string `json:"authors"`
	Journal    string `json:"journal"`
	Year       string `json:"year"`
	Citations  int    `json:"citations"`
	OpenAccess bool   `json:"open_access"`
	URL        string `json:"url"`
}

// ── wire types ───────────────────────────────────────────────────────────────

// apiResult is one item in search/article/citation/reference lists.
type apiResult struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	PMID         string `json:"pmid"`
	PMCID        string `json:"pmcid"`
	DOI          string `json:"doi"`
	Title        string `json:"title"`
	AuthorString string `json:"authorString"`
	JournalTitle string `json:"journalTitle"`
	PubYear      string `json:"pubYear"`
	AbstractText string `json:"abstractText"`
	CitedByCount int    `json:"citedByCount"`
	IsOpenAccess string `json:"isOpenAccess"`
	PubType      string `json:"pubType"`
}

type searchResp struct {
	HitCount   int `json:"hitCount"`
	ResultList struct {
		Result []apiResult `json:"result"`
	} `json:"resultList"`
}

// articleDetailResp is the envelope returned by /article/{source}/{id}.
type articleDetailResp struct {
	ResultList struct {
		Result []apiResult `json:"result"`
	} `json:"resultList"`
}

type citationResp struct {
	CitationList struct {
		Citation []apiResult `json:"citation"`
	} `json:"citationList"`
}

type referenceResp struct {
	ReferenceList struct {
		Reference []apiResult `json:"reference"`
	} `json:"referenceList"`
}

// ── helpers ──────────────────────────────────────────────────────────────────

func toArticle(r apiResult, rank int) Article {
	src := r.Source
	if src == "" {
		src = "MED"
	}
	id := r.ID
	if id == "" {
		id = r.PMID
	}
	return Article{
		Rank:       rank,
		PMID:       r.PMID,
		Source:     src,
		DOI:        r.DOI,
		Title:      r.Title,
		Authors:    truncateAuthors(r.AuthorString),
		Journal:    r.JournalTitle,
		Year:       r.PubYear,
		Citations:  r.CitedByCount,
		OpenAccess: r.IsOpenAccess == "Y",
		URL:        articleURL(src, id),
	}
}

// truncateAuthors trims an authorString to at most 3 names.
// Input: "Doe J, Smith A, Johnson B, Williams C."
// Output with >3 authors: "Doe J, Smith A, Johnson B et al."
func truncateAuthors(s string) string {
	s = strings.TrimSuffix(strings.TrimSpace(s), ".")
	if s == "" {
		return ""
	}
	parts := strings.Split(s, ", ")
	if len(parts) <= 3 {
		return s
	}
	return strings.Join(parts[:3], ", ") + " et al."
}

func articleURL(source, id string) string {
	return "https://europepmc.org/article/" + source + "/" + id
}
