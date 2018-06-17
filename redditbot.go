package main

import (
	"container/ring"
	"errors"
	"fmt"
	"log"
	"strings"
	"database/sql"
	"time"
	"strconv"
	"net/http"
	"context"

	"golang.org/x/time/rate"

	_ "github.com/mattn/go-sqlite3"
)

func main() {

	db, err := sql.Open("sqlite3", "./wikibot.db")
	if err != nil {
		panic(err)
	}

	// Set up tables and seed
	seedDatabase(db)

	// Limit to one event every two seconds
	// With a max of 2 events at once
	limit := rate.Every(time.Second * 2)
	limiter := rate.NewLimiter(limit, 2)

	// Number of comments to request at once. Max 100
	commentLimit := 100
	// Store posted comments so do not comment twice
	commented := ring.New(commentLimit)
	searchParams := make(map[string]interface{})
	searchParams["limit"] = commentLimit

	client := getClient("reddit-wikipediaposter-config.json")

	//replyChan := make(chan string)
	wikilink := "https://%s.wikipedia.org/w/api.php?format=json&action=query&prop=extracts&exintro&explaintext&formatversion=2&titles=%s"

	listingChan := make(chan Listing)

	// Check for delete comments once per minute
	tickChan := time.NewTicker(time.Minute * 1).C

	for {
		// Use a request token or wait until we have one available
		err := limiter.Wait(context.Background())
		if err !=nil {
			panic(err)
		}

		select {
			case <-tickChan :
				log.Println("Reading Messages")
				readMessages(client, db)
			case listings := <-listingChan:
				log.Println("Checking new comments for Wikipedia links")
				for _, listing := range listings.Data.Children {

					poster, sub, id := listing.Data.Author, listing.Data.Subreddit, listing.Data.Name
					if !canPost(poster, sub, id, commented, db) {
						continue
					}

					link, lang, query, section, total := extractWikiLink(listing.Data.Body)

					if total > 0 {

						extract, title, err := getComment(section, lang, query, wikilink)
						if err != nil {
							continue
						}

						comment, err := formatComment(extract, title, link)
						if err != nil {
							continue
						}

						err = postComment(comment, id,sub, client)
						if err != nil {
							panic(err)
						}

						log.Printf("Posted comment in /r/%s with parent id: %s \n", sub, id)
						// Store a small cache of comments if we have space
						commented = commented.Next()
						commented.Value = id

					}
				}
			default:
				go func() {
					log.Println("Searching for new comments")
					// Get new comments from /r/all
					listingChan <- redditSearchNew(client, searchParams)
				}()
			}
	}

}

func getComment(section string, lang string, query string, wikilink string) (string, string, error){
	var extract string
	var title string

	// If we have a section we need to make two api calls
	if section != "" {

		wikiSections, err := wikiGetAllSections(lang, query)
		if err != nil {
			return extract, title, err
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
			return extract, title, err
		}

		wikiRevisionSection, err := wikiGetSection(lang, query, sectionNum)
		if err != nil {
			return extract, title, err
		}

		extract, title = wikiRevisionSection.Query.Pages[0].Revisions[0].Content, wikiRevisionSection.Query.Pages[0].Title

		// Extract paragraphs.
		extract = extractWikiSection(extract)

	} else {
		wiki := wikiData(fmt.Sprintf(wikilink, lang, query))
		extract, title = wiki.Query.Pages[0].Extract, wiki.Query.Pages[0].Title
	}

	return extract, title, nil
}

func postComment(comment string, id string, sub string, client *http.Client) error {
	commentParams := make(map[string]interface{})
	commentParams["text"] = comment
	commentParams["parent"] = id

	err := postNewComment(client, commentParams)
	if err != nil {
		log.Printf("Error posting comment in /r/%s with parent id: %s: %s", sub, id, err.Error())
	}

	return err
}

func readMessages(client *http.Client, db *sql.DB) {
	msgs, err := getUnreadMsgs(client)
	if err != nil {
		log.Printf(err.Error())
		log.Fatal("Error reading messages")
	}

	// Read all unread messages.
	for _, msg := range msgs.Data.Children {
		// If the message was a comment and the comment said Delete then remove it.
		if msg.Data.WasComment && strings.TrimSpace(strings.ToLower(msg.Data.Body)) == "!delete" {
			delparams := make(map[string]interface{})
			delparams["id"] = msg.Data.ParentID
			deleteComment(client, delparams)
		}

		if msg.Data.WasComment && strings.TrimSpace(strings.ToLower(msg.Data.Body)) == "!block" {
			// 		seedStatement, err := db.Prepare("INSERT INTO blacklist (name, type, datetime, reddit_id) VALUES(?, ?, ?, ?)")

			statement, err := db.Prepare("INSERT INTO blacklist (name, type, reddit_id, comment_id) VALUES(?, ?, ?, ?)")
			if err != nil {
				log.Println("Failed to enter user into blacklist", err)
			}

			name := strings.ToLower(msg.Data.Author)

			//@TODO Get user id. Need to make request to /u/name/about.json
			statement.Exec(name, "user", null, msg.Data.ID)
		}

		msgparams := make(map[string]interface{})
		msgparams["id"] = fmt.Sprintf("t1_%s", msg.Data.ID)
		// Mark as read
		setMsgRead(client, msgparams)
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

func canPost(poster string, sub string, id string, commented *ring.Ring, db *sql.DB) bool {

	var count int
	q, err := db.Query("SELECT COUNT(*) FROM blacklist WHERE (LOWER(blacklist.name) = LOWER(?) AND blacklist.type = 'user') OR (LOWER(blacklist.name) = LOWER(?) AND blacklist.type = 'subreddit')", poster, sub)
	if err != nil {
		log.Fatal(err)
	}

	defer q.Close()

	for q.Next() {
		err = q.Scan(&count)
	}

	if count != 0 {
		//log.Printf("Blocked from responding to /u/%s or /r/%s \n", poster, sub)
		return false
	}

	// Already comment here
	if ringContains(commented, id) {
		//log.Printf("Already replied to comment: %s \n", id)
		return false
	}

	return true
}

func ringContains(r *ring.Ring, s string) bool {

	b := false

	r.Do(func(x interface{}) {
		if x == s {
			b = true
		}
	})
	return b
}
