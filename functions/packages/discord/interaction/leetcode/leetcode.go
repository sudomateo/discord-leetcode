package leetcode

import (
	"math/rand"
	"time"
)

// Difficulty represents the difficulty of LeetCode questions.
type Difficulty string

// The enumeration of difficulties.
const (
	DifficultyEasy   Difficulty = "EASY"
	DifficultyMedium Difficulty = "MEDIUM"
	DifficultyHard   Difficulty = "HARD"
)

type Client struct{}

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
