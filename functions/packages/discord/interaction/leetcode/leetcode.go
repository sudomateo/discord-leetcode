package leetcode

import (
	"math/rand"
	"net/http"
	"time"
)

// The LeetCode API client.
type Client struct {
	httpClient *http.Client
}

// NewClient builds and returns a LeetCode API client ready for use.
func NewClient() Client {
	return Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Difficulty represents the difficulty of LeetCode questions.
type Difficulty string

// The enumeration of difficulties.
const (
	DifficultyEasy   Difficulty = "EASY"
	DifficultyMedium Difficulty = "MEDIUM"
	DifficultyHard   Difficulty = "HARD"
)


// RandomDifficulty computes a random LeetCode problem difficulty.
func RandomDifficulty() Difficulty {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	switch r.Int() % 3 {
	case 0:
		return DifficultyEasy
	case 1:
		return DifficultyMedium
	case 2:
		return DifficultyHard
	default:
		return DifficultyEasy
	}
}
