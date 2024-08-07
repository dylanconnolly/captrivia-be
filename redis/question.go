package redis

// func (db *DB) getQuestion(id string) (*captrivia.Question, error) {
// 	questionKey := fmt.Sprintf(questionKey, id)

// 	fields, err := db.client.HGetAll(ctx, questionKey).Result()
// 	if err != nil {
// 		return nil, err
// 	}

// 	correctIndex, err := strconv.Atoi(fields["correct_index"])
// 	if err != nil {
// 		return nil, err
// 	}

// 	optionsKey := fmt.Sprintf(questionOptionsKey, id)

// 	options, err := db.client.LRange(ctx, optionsKey, 0, -1).Result()
// 	if err != nil {
// 		return nil, err
// 	}

// 	q := captrivia.NewQuestion(fields["id"], fields["question_text"], options, correctIndex)

// 	return q, nil
// }

// func (db *DB) generateRandomQuestionIDs(count int) ([]string, error) {
// 	questionIDs, err := db.client.SRandMemberN(ctx, allQuestionsKey, int64(count)).Result()
// 	if err != nil {
// 		return nil, err
// 	}
// 	log.Println("count is ", count)
// 	log.Printf("generated question IDs: %+v", questionIDs)
// 	return questionIDs, nil
// }

// func (db *DB) GetNextQuestion(gameID uuid.UUID) (*captrivia.Question, error) {
// 	gameQuestionsKey := fmt.Sprintf(gameQuestionsKey, gameID)

// 	id, err := db.client.LPop(ctx, gameQuestionsKey).Result()
// 	if err != nil {
// 		return nil, err
// 	}

// 	q, err := db.getQuestion(id)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return q, nil
// }

// func (db *DB) GetQuestionCorrectIndex(id string) (int, error) {
// 	questionKey := fmt.Sprintf(questionKey, id)

// 	correctIndex, err := db.client.HGet(ctx, questionKey, "correct_index").Result()
// 	if err != nil {
// 		return -1, err
// 	}

// 	index, err := strconv.Atoi(correctIndex)
// 	if err != nil {
// 		return -1, err
// 	}

// 	return index, nil
// }
