package main

type Messages struct {
	Data struct {
		After    string      `json:"after"`
		Before   interface{} `json:"before"`
		Children []struct {
			Data struct {
				Author           interface{} `json:"author"`
				Body             string      `json:"body"`
				BodyHtml         string      `json:"body_html"`
				Context          string      `json:"context"`
				Created          float64     `json:"created"`
				CreatedUtc       float64     `json:"created_utc"`
				Dest             string      `json:"dest"`
				Distinguished    string      `json:"distinguished"`
				FirstMessage     interface{} `json:"first_message"`
				FirstMessageName interface{} `json:"first_message_name"`
				ID               string      `json:"id"`
				Name             string      `json:"name"`
				New              bool        `json:"new"`
				ParentID         interface{} `json:"parent_id"`
				Replies          string      `json:"replies"`
				Subject          string      `json:"subject"`
				Subreddit        string      `json:"subreddit"`
				WasComment       bool        `json:"was_comment"`
			} `json:"data"`
			Kind string `json:"kind"`
		} `json:"children"`
		Modhash string `json:"modhash"`
	} `json:"data"`
	Kind string `json:"kind"`
}
