package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type PostResponse struct {
	Jquery  [][]interface{} `json:"-"`
	Success bool            `json:"success"`
}

func redditSearchNew(client *http.Client, params map[string]interface{}) Listing {
	resp, _ := request(client, "GET", "/r/personalbotplayground/comments.json", params)

	var listings Listing
	json.Unmarshal(resp, &listings)

	return listings
}

func deleteComment(client *http.Client, params map[string]interface{}) {
	_, err := request(client, "POST", "/api/del", params)
	if err != nil {
		panic(err)
	}
}

// Handle error better
func getUnreadMsgs(client *http.Client) (Messages, error) {
	var msgs Messages

	resp, err := request(client, "GET", "/message/unread", nil)
	if err != nil {
		return msgs, err
	}

	err = json.Unmarshal(resp, &msgs)

	return msgs, err
}

func setMsgRead(client *http.Client, params map[string]interface{}) {
	_, err := request(client, "POST", "/api/read_message", params)
	if err != nil {
		panic(err)
	}
}

func postNewComment(client *http.Client, params map[string]interface{}) error {

	fmt.Println(params)

	resp, _ := request(client, "POST", "/api/comment", params)

	var postResponse PostResponse

	json.Unmarshal(resp, &postResponse)

	if postResponse.Success == false {
		return errors.New("Error posting comment")
	}

	return nil
}

func request(client *http.Client, method string, path string, params map[string]interface{}) ([]byte, error) {

	values := make(url.Values)
	for k, v := range params {
		values.Set(k, fmt.Sprintf("%v", v))
	}

	url := fmt.Sprintf("https://oauth.reddit.com%s?%s", path, values.Encode())

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "User-Agent: wikipediaposterbot:v0.0.3 (by /u/WikipediaPoster)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	//fmt.Printf("%s, \n", body)

	return body, nil
}
