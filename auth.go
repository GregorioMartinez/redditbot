package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type clientInfo struct {
	ClientID string `json:"client_id"`
	Secret   string
}

func getClient(s string) *http.Client {
	ClientID, ClientSecret := getAuthInfo(s)

	conf := &oauth2.Config{
		ClientID:     ClientID,
		ClientSecret: ClientSecret,
		Scopes:       []string{"read", "identity", "submit"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://www.reddit.com/api/v1/authorize",
			TokenURL: "https://www.reddit.com/api/v1/access_token",
		},
		RedirectURL: "http://localhost:3000",
	}

	// First try to Read Token
	token, err := getTokenFromFile("reddit-token.json")
	if err != nil {
		for {
			token, err = getTokenFromWeb(conf)
			if err != nil {
				log.Printf("Error: %s \n. Sleeping for 3 seconds before retrying. \n", err)
				time.Sleep(3 * time.Second)
			} else {
				break
			}
		}
	}

	return conf.Client(oauth2.NoContext, token)
}

func getTokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}

	err = json.NewDecoder(f).Decode(t)
	defer f.Close()

	return t, err
}

func saveTokenToFile(filename string, token *oauth2.Token) {
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(token)
	if err != nil {
		panic(err)
	}
}

func getTokenFromWeb(conf *oauth2.Config) (*oauth2.Token, error) {
	duration := oauth2.SetAuthURLParam("duration", "permanent")

	url := conf.AuthCodeURL("goingtoignorethiskindoffornow", oauth2.AccessTypeOffline, duration)
	fmt.Printf("Visit the URL for the auth dialog: %v \n", url)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		panic(err)
	}

	token, err := conf.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, errors.New("429")
	}

	saveTokenToFile("reddit-token.json", token)

	return token, nil
}

func getAuthInfo(filename string) (string, string) {

	ClientID, ClientSecret := os.Getenv("reddit.wikibot-ClientID"), os.Getenv("reddit.wikibot-ClientSecret")

	if ClientID == "" || ClientSecret == "" {
		log.Println("Env vars not set. Opening file.")
	} else {
		return ClientID, ClientSecret
	}

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal("File not found")
	}

	defer file.Close()

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	var info clientInfo
	err = json.Unmarshal(fileContents, &info)
	if err != nil {
		panic(err)
	}

	return info.ClientID, info.Secret
}
