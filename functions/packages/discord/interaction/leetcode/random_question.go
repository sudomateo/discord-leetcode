package leetcode

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// The GraphQL query to fetch a random LeetCode question.
const RandomQuestionQuery = `
query randomQuestion($categorySlug: String, $filters: QuestionListFilterInput) {
    randomQuestion(categorySlug: $categorySlug, filters: $filters) {
        titleSlug
    }
}`

// RandomQuestionRequest is the request that's sent to the randomQuestion
// GraphQL API.
type RandomQuestionRequest struct {
	Query     string                  `json:"query"`
	Variables RandomQuestionVariables `json:"variables"`
}

// RandomQuestionVariables are variables that can be set on the request to the
// randomQuestion GraphQL API.
type RandomQuestionVariables struct {
	CategorySlug string                `json:"categorySlug"`
	Filters      RandomQuestionFilters `json:"filters"`
}

// RandomQuestionFilters are filters that can be set on the request to the
// randomQuestion GraphQL API.
type RandomQuestionFilters struct {
	Difficulty Difficulty `json:"difficulty"`
	Tags       []string   `json:"tags,omitempty"`
}

// RandomQuestionResponse is the response sent back from the randomQuestion
// GraphQL API.
type RandomQuestionResponse struct {
	Data struct {
		RandomQuestion struct {
			TitleSlug string `json:"titleSlug"`
		} `json:"randomQuestion"`
	} `json:"data"`
}

// RandomQuestion retrieves a random LeetCode problem.
func (l Client) RandomQuestion(difficulty Difficulty) (RandomQuestionResponse, error) {
	requestBody := RandomQuestionRequest{
		Query: RandomQuestionQuery,
		Variables: RandomQuestionVariables{
			Filters: RandomQuestionFilters{
				Difficulty: difficulty,
			},
		},
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(requestBody); err != nil {
		return RandomQuestionResponse{}, err
	}

	req, err := http.NewRequest(http.MethodPost, "https://leetcode.com/graphql", buf)
	if err != nil {
		return RandomQuestionResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://leetcode.com")
	req.Header.Set("Referer", "https://leetcode.com")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return RandomQuestionResponse{}, err
	}
	defer resp.Body.Close()

	var lcResp RandomQuestionResponse
	if err := json.NewDecoder(resp.Body).Decode(&lcResp); err != nil {
		return RandomQuestionResponse{}, err
	}

	return lcResp, nil
}
