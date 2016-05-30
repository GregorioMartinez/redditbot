package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kennygrant/sanitize"
)

func main() {
	// Load blacklists
	blacklistSubs, blacklistUsers := getBlacklist("subs.txt"), getBlacklist("users.txt")

	// Get the client for making requests
	client := getClient("reddit-wikipediaposter-config.json")

	// Number of comments to request at once. Max 100
	commentLimit := 100

	// Used for storing comments already replied to
	commented := make([]string, 0, commentLimit)

	// Parameters for searching comments on reddit
	searchparams := make(map[string]interface{})
	searchparams["limit"] = commentLimit

	//Wikipedia API endpoint
	wikilink := "https://%s.wikipedia.org/w/api.php?format=json&action=query&prop=extracts&exintro&explaintext&formatversion=2&titles=%s"

	// Run
	for {
		// Get new comments from /r/all
		listings := redditSearchNew(client, searchparams)

		for _, listing := range listings.Data.Children {

			poster, sub, id := strings.ToLower(listing.Data.Author), strings.ToLower(listing.Data.Subreddit), listing.Data.Name

			if !canPost(poster, sub, id, blacklistSubs, blacklistUsers, commented) {
				continue
			}

			// Store a small cache of comments if we have space
			if len(commented) < commentLimit {
				log.Printf("Adding %s to commented list \n", id)
				commented = append(commented, id)
			} else {
				commented = make([]string, 0, 1)
			}

			link, lang, query, section, total := extractWikiLink(listing.Data.Body)
			// Extraction actually worked
			if total > 0 {

				var extract string
				var title string

				// If we have a section we need to make two api calls
				if section != "" {
					// Do additional calls
					wurl := fmt.Sprintf(`https://%s.wikipedia.org/w/api.php?action=parse&page=%s&prop=sections&format=json&formatversion=2`, lang, query)

					var sectionNum int64

					resp, err := http.Get(wurl)
					if err != nil {
						log.Println(err.Error())
						continue
					}

					body, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						panic(err)
					}

					var wikiSections WikipediaSectionResponse

					err = json.Unmarshal(body, &wikiSections)
					if err != nil {
						panic(err)
					}

					for _, wikiSection := range wikiSections.Parse.Sections {
						if wikiSection.Anchor == section {
							sectionNum, err = strconv.ParseInt(wikiSection.Index, 10, 32)
							if err != nil {
								sectionNum = 0
							}
							break
						}
					}

					if sectionNum == 0 {
						continue
					}

					surl := fmt.Sprintf(`https://%s.wikipedia.org/w/api.php?action=query&prop=revisions&titles=%s&rvprop=content&rvsection=%d&formatversion=2&rvlimit=1&rvparse&format=json`, lang, query, sectionNum)
					resp, err = http.Get(surl)
					if err != nil {
						log.Println(err.Error())
						continue
					}

					body, err = ioutil.ReadAll(resp.Body)
					if err != nil {
						panic(err)
					}

					var wikiRevisionSection WikipediaRevisionResponse

					err = json.Unmarshal(body, &wikiRevisionSection)
					if err != nil {
						panic(err)
					}

					r := regexp.MustCompile(`\[.*\]`)

					extract, title = sanitize.HTML(wikiRevisionSection.Query.Pages[0].Revisions[0].Content), wikiRevisionSection.Query.Pages[0].Title

					extract = r.ReplaceAllLiteralString(extract, " ")

				} else {
					wiki := wikiData(fmt.Sprintf(wikilink, lang, query))
					extract, title = wiki.Query.Pages[0].Extract, wiki.Query.Pages[0].Title
				}

				comment, err := formatComment(extract, title, link)
				if err != nil {
					log.Printf("%s", err.Error())
					continue
				}
				commentparams := make(map[string]interface{})
				commentparams["text"] = comment
				commentparams["parent"] = id
				postNewComment(client, commentparams)
			}
		}
		time.Sleep(3 * time.Second)
	}
}

func formatComment(extract string, title string, url string) (string, error) {
	// Remove whitespace
	commentBody := strings.TrimSpace(extract)

	if len(commentBody) == 0 {
		return commentBody, errors.New("Empty Comment Body")
	}

	// Format the output
	commentBody = strings.Replace(commentBody, "\n", "\n\n>", -1)

	// Only want 2 paragraphs
	paragraphs := strings.Split(commentBody, ">")

	if len(paragraphs) >= 2 {
		commentBody = fmt.Sprintf("%s >%s", paragraphs[0], paragraphs[1])
	}

	// Escape the ()'s found in links
	replaceUrl := strings.NewReplacer("(", "\\(", ")", "\\)")
	commentTitle, commentLink := title, replaceUrl.Replace(url)
	commentInfo := fmt.Sprint("^I ^am ^a ^bot. ^Please ^contact ^[/u/GregMartinez](https://www.reddit.com/user/GregMartinez) ^with ^any ^questions ^or ^feedback.")
	comment := fmt.Sprintf("**[%s](%s)** \n\n ---  \n\n>%s \n\n --- \n\n %s", commentTitle, commentLink, commentBody, commentInfo)

	return comment, nil
}

func canPost(poster string, sub string, id string, blacklistSubs []string, blacklistUsers []string, commented []string) bool {
	// Talking to a blacklisted person
	if contains(blacklistUsers, poster) {
		return false
	}

	// Talking in a blacklisted sub
	if contains(blacklistSubs, sub) {
		return false
	}

	// Already comment here
	if contains(commented, id) {
		return false
	}

	return true
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

func contains(s []string, b string) bool {
	sort.Strings(s)

	i := sort.SearchStrings(s, b)

	if i >= len(s) || s[i] != b {
		return false
	}
	return true
}

func getBlacklist(filename string) []string {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Unable to read blacklist %s", filename)
	}

	b := bytes.ToLower(contents)
	s := string(b[:])

	return strings.Split(s, "\n")
}
