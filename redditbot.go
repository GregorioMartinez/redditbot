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

	_ "github.com/mattn/go-sqlite3"

)

func main() {

	// Open DB connection
	db, err := sql.Open("sqlite3", "./wikibot.db")
	if err != nil {
		panic(err)
	}

	// Make sure table exists
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS \"blacklist\" (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, type TEXT, datetime INTEGER, reddit_id TEXT, comment_id TEXT)")
	if err != nil {
		log.Fatal(err)
	}

	_, err = statement.Exec()
	if err != nil {
		log.Fatal(err)
	}

	// Check if DB has bot already in it
	q, err := db.Query("SELECT COUNT(*) FROM blacklist WHERE reddit_id = ?", "xurih")
	if err != nil {
		log.Fatal(err)
	}
	defer q.Close()

	var count int
	for q.Next() {
		err = q.Scan(&count)
	}

	// If bot user isn't there then seed the db
	if count == 0 {
		// populate
		seedStatement, err := db.Prepare("INSERT INTO blacklist (name, type, datetime, reddit_id) VALUES(?, ?, ?, ?)")
		if err != nil {
			log.Fatal(err)
		}

		//@TODO This should actually make a request to me.json to pull this info dynamically
		now := time.Now()
		_, err = seedStatement.Exec("WikipediaPoster", "user", now.Unix(), "xurih")
		if err != nil {
			panic(err)
		}

		log.Println("Seeded database with bot user")

	}

	// Number of comments to request at once. Max 100
	commentLimit := 100
	// Store posted comments so do not comment twice
	commented := ring.New(commentLimit)
	searchparams := make(map[string]interface{})
	searchparams["limit"] = commentLimit

	client := getClient("reddit-wikipediaposter-config.json")

	//@TODO
	// Delete any comments users replied Please deleteto
	// Block any users who replied Please do not reply to me

	//replyChan := make(chan string)
	wikilink := "https://%s.wikipedia.org/w/api.php?format=json&action=query&prop=extracts&exintro&explaintext&formatversion=2&titles=%s"

	// Run
	for {

		// Get new comments from /r/all
		listings := redditSearchNew(client, searchparams)

		for _, listing := range listings.Data.Children {

			poster, sub, id := listing.Data.Author, listing.Data.Subreddit, listing.Data.Name
			if !canPost(poster, sub, id, commented, db) {
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

				//err = postNewComment(client, commentparams)
				//if err != nil {
				//	log.Printf("Error posting comment in /r/%s with parent id: %s: %s", sub, id, err.Error())
				//	continue
				//}

				log.Printf("Posted comment in /r/%s with parent id: %s \n", sub, id)
				// Store a small cache of comments if we have space
				commented = commented.Next()
				commented.Value = id
			}
		}
		time.Sleep(3 * time.Second)
	}

}

func readMessages(client *http.Client){
	msgs, err := getUnreadMsgs(client)
	if err != nil {
		log.Printf(err.Error())
		log.Fatal("Error reading messages")
	}

	// Read all unread messages.
	for _, msg := range msgs.Data.Children {
		// If the message was a comment and the comment said Delete then remove it.
		if msg.Data.WasComment && msg.Data.Body == "Delete" {
			delparams := make(map[string]interface{})
			delparams["id"] = msg.Data.ParentID
			deleteComment(client, delparams)
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
	q, err := db.Query("SELECT COUNT(*) FROM blacklist WHERE (LOWER(name) = LOWER(?) AND type = \"user\") OR  (LOWER(name) = LOWER(?) AND type = \"subreddit\")", poster, sub)
	if err != nil {
		log.Fatal(err)
	}

	defer q.Close()

	for q.Next() {
		err = q.Scan(&count)
	}

	if count != 0 {
		log.Printf("Blocked from responding to /u/%s or /r/%s \n", poster, sub)
		return false
	}

	// Already comment here
	if ringContains(commented, id) {
		log.Printf("Already replied to comment: %s \n", id)
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
