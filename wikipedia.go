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

type WikipediaSectionResponse struct {
	Parse struct {
		Pageid   float64 `json:"pageid"`
		Sections []struct {
			Anchor     string  `json:"anchor"`
			Byteoffset float64 `json:"byteoffset"`
			Fromtitle  string  `json:"fromtitle"`
			Index      string  `json:"index"`
			Level      string  `json:"level"`
			Line       string  `json:"line"`
			Number     string  `json:"number"`
			Toclevel   float64 `json:"toclevel"`
		} `json:"sections"`
		Title string `json:"title"`
	} `json:"parse"`
}

type WikipediaRevisionResponse struct {
	Continue struct {
		Continue   string `json:"continue"`
		Rvcontinue string `json:"rvcontinue"`
	} `json:"continue"`
	Query struct {
		Normalized []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"normalized"`
		Pages []struct {
			Ns        float64 `json:"ns"`
			Pageid    float64 `json:"pageid"`
			Revisions []struct {
				Content string `json:"content"`
			} `json:"revisions"`
			Title string `json:"title"`
		} `json:"pages"`
	} `json:"query"`
}
