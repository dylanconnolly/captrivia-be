package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
)

// Question is a read-only representation of a question, meaning it can be
// safely copied around despite the presence of a slice.
type Question struct {
	ID           string   `json:"id"`
	QuestionText string   `json:"questionText"`
	Options      []string `json:"options"`
	CorrectIndex int      `json:"correctIndex"` //TODO: remove this from frontend
}

// LoadQuestions reads a JSON file containing a list of questions to be used
// for games.
func LoadQuestions(filename string) ([]Question, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read questions file: %w", err)
	}

	var questions []Question
	err = json.Unmarshal(data, &questions)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal questions: %w", err)
	}

	return questions, nil
}

// ShuffleQuestions shuffles the questions and returns n questions, the original
// slice is not returned nor modified.
//
// It is advisable to call this function with an n that is much smaller than the
// length of the questions slice, as the implementation is not efficient as n
// approaches len(questions).
func ShuffleQuestions(questions []Question, n int) []Question {
	q := make([]Question, n)

	// This is no fisher-yates shuffle, but in theory the set of all questions
	// available for use in games should be massive.
	usedQuestions := map[int]struct{}{}
	for i := range q {
		for {
			index := rand.Intn(len(questions))
			if _, ok := usedQuestions[index]; !ok {
				usedQuestions[index] = struct{}{}
				q[i] = questions[index]
				break
			}
		}
	}

	return q
}
