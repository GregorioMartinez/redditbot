package main

type Listing struct {
	Data struct {
		Children []struct {
			Data struct {
				Author      string `json:"author"`
				Body        string `json:"body"`
				Name        string `json:"name"`
				Subreddit   string `json:"subreddit"`
				SubredditID string `json:"subreddit_id"`
				Title       string `json:"title"`
			} `json:"data"`
			Kind string `json:"kind"`
		} `json:"children"`
		Modhash interface{} `json:"modhash"`
	} `json:"data"`
	Kind string `json:"kind"`
}
