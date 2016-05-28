package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

func redditSearchNew(client *http.Client, params map[string]interface{}) Listing {
	resp, _ := request(client, "GET", "/r/all/comments.json", params)

	var listings Listing
	json.Unmarshal(resp, &listings)

	return listings
}

func postNewComment(client *http.Client, params map[string]interface{}) {
	request(client, "POST", "/api/comment", params)
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

	req.Header.Set("User-Agent", "User-Agent: wikipediaposterbot:v0.0.2 (by /u/WikipediaPoster)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return body, nil
}
