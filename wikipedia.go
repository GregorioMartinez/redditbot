package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/kennygrant/sanitize"
)

type WikipediaResponse struct {
	Batchcomplete bool `json:"-"`
	Query         struct {
		Normalized []struct {
			From string `json:"-"`
			To   string `json:"-"`
		} `json:"-"`
		Pages []struct {
			Extract string  `json:"extract"`
			Ns      float64 `json:"-"`
			Pageid  float64 `json:"-"`
			Title   string  `json:"title"`
		} `json:"pages"`
	} `json:"query"`
}

type WikipediaSectionResponse struct {
	Parse struct {
		Pageid   float64 `json:"pageid"`
		Sections []struct {
			Anchor     string  `json:"anchor"`
			Byteoffset float64 `json:"-"`
			Fromtitle  string  `json:"-"`
			Index      string  `json:"index"`
			Level      string  `json:"level"`
			Line       string  `json:"line"`
			Number     string  `json:"number"`
			Toclevel   float64 `json:"-"`
		} `json:"sections"`
		Title string `json:"title"`
	} `json:"parse"`
}

type WikipediaRevisionResponse struct {
	Continue struct {
		Continue   string `json:"continue"`
		Rvcontinue string `json:"rvcontinue"`
	} `json:"continue"`
	Query struct {
		Normalized []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"normalized"`
		Pages []struct {
			Ns        float64 `json:"ns"`
			Pageid    float64 `json:"pageid"`
			Revisions []struct {
				Content string `json:"content"`
			} `json:"revisions"`
			Title string `json:"title"`
		} `json:"pages"`
	} `json:"query"`
}

func wikiGetSection(lang string, query string, sectionNum int64) (WikipediaRevisionResponse, error) {
	var wikiRevisionSection WikipediaRevisionResponse

	url := fmt.Sprintf(`https://%s.wikipedia.org/w/api.php?action=query&prop=revisions&titles=%s&rvprop=content&rvsection=%d&formatversion=2&rvlimit=1&rvparse&format=json`, lang, query, sectionNum)
	resp, err := http.Get(url)
	if err != nil {
		return wikiRevisionSection, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return wikiRevisionSection, err
	}

	err = json.Unmarshal(body, &wikiRevisionSection)
	if err != nil {
		return wikiRevisionSection, err
	}

	return wikiRevisionSection, nil
}

func wikiGetAllSections(lang string, query string) (WikipediaSectionResponse, error) {
	var wikiSections WikipediaSectionResponse

	url := fmt.Sprintf(`https://%s.wikipedia.org/w/api.php?action=parse&page=%s&prop=sections&format=json&formatversion=2`, lang, query)

	resp, err := http.Get(url)
	if err != nil {
		return wikiSections, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return wikiSections, err
	}

	err = json.Unmarshal(body, &wikiSections)
	if err != nil {
		return wikiSections, err
	}

	return wikiSections, nil
}

func extractWikiSection(s string) string {

	r := regexp.MustCompile(`<p>.+?(?:\n+)?</p>`)
	tmp := strings.Join(r.FindAllString(s, -1), "")

	re := regexp.MustCompile(`\[.+?\]`)
	extract := re.ReplaceAllLiteralString(tmp, "")

	return sanitize.HTML(extract)
}

func extractWikiLink(url string) (string, string, string, string, int) {
	// RegEx for finding wikipedia links
	r := regexp.MustCompile(`http(?:s)?://([a-zA-Z]{2}).(?:m\.)?wikipedia.org/wiki/([^\s|#]+)(?:#(.+))?`)

	matches := r.FindStringSubmatch(url)

	if len(matches) > 0 {
		link, lang, url, section, total := matches[0], matches[1], matches[2], matches[3], len(matches)
		return link, lang, url, section, total
	}

	return "", "", "", "", 0

}

func wikiData(link string) WikipediaResponse {
	resp, err := http.Get(link)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var wiki WikipediaResponse

	err = json.Unmarshal(body, &wiki)
	if err != nil {
		panic(err)
	}

	return wiki
}
