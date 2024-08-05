package redis

import (
	"fmt"
	"strconv"

	"github.com/dylanconnolly/captrivia-be/captrivia"
)

func (db *DB) getQuestion(id string) (*captrivia.Question, error) {
	questionKey := fmt.Sprintf(questionKey, id)

	fields, err := db.client.HGetAll(ctx, questionKey).Result()
	if err != nil {
		return nil, err
	}

	correctIndex, err := strconv.Atoi(fields["correct_index"])
	if err != nil {
		return nil, err
	}

	optionsKey := fmt.Sprintf(questionOptionsKey, id)

	options, err := db.client.LRange(ctx, optionsKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	q := captrivia.NewQuestion(fields["id"], fields["question_text"], options, correctIndex)

	return q, nil
}

func (db *DB) generateRandomQuestions(count int) ([]captrivia.Question, error) {
	var questions []captrivia.Question

	questionIDs, err := db.client.SRandMemberN(ctx, allQuestionsKey, int64(count)).Result()
	if err != nil {
		return nil, err
	}

	for _, id := range questionIDs {
		q, err := db.getQuestion(id)
		if err != nil {
			return nil, err
		}
		questions = append(questions, *q)
	}

	return questions, nil
}
