package main

type Pr struct {
	Action string `json:"action"`
	Number int `json:"number"`
	PullRequest struct {
		Url string `json:"url"`
		State string `json:"state"`
		Head struct {
			Repo struct {
				FullName string `json:"full_name"`
			} `json:"repo"`
		} `json:"head"`
	} `json:"pull_request"`
}
