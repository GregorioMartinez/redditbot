package main

import (
	"bytes"
	"container/ring"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Load blacklists
	blacklistSubs, blacklistUsers := getBlacklist("subs.txt"), getBlacklist("users.txt")

	log.Printf("Blacklisted Subs: %v \n", blacklistSubs)
	log.Printf("Blacklisted Users: %v \n", blacklistUsers)

	// Get the client for making requests
	client := getClient("reddit-wikipediaposter-config.json")

	// Number of comments to request at once. Max 100
	commentLimit := 100

	commented := ring.New(commentLimit)

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

			link, lang, query, section, total := extractWikiLink(listing.Data.Body)

			if total > 0 {

				var extract string
				var title string

				// If we have a section we need to make two api calls
				if section != "" {

					wikiSections, err := wikiGetAllSections(lang, query)
					if err != nil {
						continue
					}

					var sectionNum int64

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

					wikiRevisionSection, err := wikiGetSection(lang, query, sectionNum)
					if err != nil {
						continue
					}

					extract, title = wikiRevisionSection.Query.Pages[0].Revisions[0].Content, wikiRevisionSection.Query.Pages[0].Title

					// Extract paragraphs.
					extract = extractWikiSection(extract)

				} else {
					wiki := wikiData(fmt.Sprintf(wikilink, lang, query))
					extract, title = wiki.Query.Pages[0].Extract, wiki.Query.Pages[0].Title
				}

				comment, err := formatComment(extract, title, link)
				if err != nil {
					continue
				}
				commentparams := make(map[string]interface{})
				commentparams["text"] = comment
				commentparams["parent"] = id

				err = postNewComment(client, commentparams)
				if err != nil {
					log.Printf("Error posting comment in /r/%s with parent id: %s: %s", sub, id, err.Error())
					continue
				}

				log.Printf("Posted comment in /r/%s with parent id: %s \n", sub, id)
				// Store a small cache of comments if we have space
				commented = commented.Next()
				commented.Value = id
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

func canPost(poster string, sub string, id string, blacklistSubs []string, blacklistUsers []string, commented *ring.Ring) bool {
	// Talking to a blacklisted person
	if contains(blacklistUsers, poster) {
		return false
	}

	// Talking in a blacklisted sub
	if contains(blacklistSubs, sub) {
		return false
	}

	// Already comment here
	if ringContains(commented, id) {
		return false
	}

	return true
}

func ringContains(r *ring.Ring, b string) bool {

	fmt.Println(r.Len())
	for i := 0; i < r.Len(); i++ {
		if b == r.Value {
			return true
		}
	}
	return false

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
