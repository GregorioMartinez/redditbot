package main

type WikipediaResponse struct {
	Batchcomplete bool `json:"batchcomplete"`
	Query         struct {
		Normalized []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"normalized"`
		Pages []struct {
			Extract string  `json:"extract"`
			Ns      float64 `json:"ns"`
			Pageid  float64 `json:"pageid"`
			Title   string  `json:"title"`
		} `json:"pages"`
	} `json:"query"`
}
