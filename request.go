package main

func searchNew(client *http.Client, params map[string]interface{}) Listing {

	resp := request(client, "GET", "/r/all/personalbotplayground.json", params)

	var listings Listing
	json.Unmarshal(resp, &listings)

	return listings
}

func postNewComment(client *http.Client, params map[string]interface{}) {
	request(client, "POST", "/api/comment", params)
}

func request(client *http.Client, method string, path string, params map[string]interface{}) []byte {

	values := make(url.Values)
	for k, v := range params {
		values.Set(k, fmt.Sprintf("%v", v))
	}

	url := fmt.Sprintf("https://oauth.reddit.com%s?%s", path, values.Encode())

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("User-Agent", "User-Agent: wikipediaposterbot:v0.0.1 (by /u/WikipediaPoster)")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return body
}
