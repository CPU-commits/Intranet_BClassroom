package services

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (w *WorkSerice) GetForm(idWork, userId string) (map[string]interface{}, *res.ErrorRes) {
	// Recovery if close channel
	defer func() {
		recovery := recover()
		if recovery != nil {
			fmt.Printf("A channel closed")
		}
	}()

	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjUser, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get work
	work, err := workRepository.GetWorkFromId(idObjWork)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}

	if work.Type != "form" {
		return nil, &res.ErrorRes{
			Err:        fmt.Errorf("este trabajo no es de tipo formulario"),
			StatusCode: http.StatusBadRequest,
		}
	}
	if time.Now().Before(work.DateStart.Time()) {
		return nil, &res.ErrorRes{
			Err:        fmt.Errorf("no se puede acceder a este trabajo todavía"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	// Get form
	form, errRes := formService.GetForm(work.Form.Hex(), userId, false)
	if errRes != nil {
		return nil, errRes
	}
	// Get form access
	formAccess, err := w.getAccessFromIdStudentNIdWork(idObjUser, idObjWork)
	if err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}

	if formAccess == nil && time.Now().Before(work.DateLimit.Time()) {
		modelFormAccess := models.NewModelFormAccess(
			idObjUser,
			idObjWork,
		)
		inserted, err := formAccessModel.NewDocument(modelFormAccess)
		if err != nil {
			return nil, &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
		var diff time.Duration
		if work.FormAccess == "default" {
			// !Warning - before work.DateLimit.Time().Sub()
			diff = time.Until(work.DateLimit.Time())
		} else {
			diff = time.Duration(work.TimeFormAccess * int(time.Second))
		}
		err = nats.PublishEncode("close_student_form", &CloseForm{
			Work:    idWork,
			Student: userId,
			Diff:    diff.Hours(),
		})
		if err != nil {
			return nil, &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
		formAccess = &models.FormAccess{
			ID:      inserted.InsertedID.(primitive.ObjectID),
			Date:    primitive.NewDateTimeFromTime(time.Now()),
			Student: idObjUser,
			Work:    idObjWork,
			Status:  "opened",
		}
	}
	if formAccess == nil && time.Now().After(work.DateLimit.Time()) {
		return nil, &res.ErrorRes{
			Err:        fmt.Errorf("no accediste al formulario, no hay respuestas a revisar"),
			StatusCode: http.StatusBadRequest,
		}
	}
	var newItems []models.FormItemWLookup
	// Answers
	questionsLen := 0
	for _, item := range form[0].Items {
		for range item.Questions {
			questionsLen += 1
		}
	}
	var answers = make([]*AnswerRes, questionsLen)
	var wg sync.WaitGroup
	c := make(chan (int), 5)

	for j, item := range form[0].Items {
		var questions = make([]models.ItemQuestion, len(item.Questions))
		var err error

		for i, question := range item.Questions {
			wg.Add(1)
			c <- 1

			answerIndex := i
			for j != 0 {
				answerIndex += len(form[0].Items[i].Questions)
				j -= 1
			}
			go func(
				wg *sync.WaitGroup,
				question models.ItemQuestion,
				questions []models.ItemQuestion,
				iQuestion,
				iAnswer int,
				returnErr *error,
			) {
				defer wg.Done()
				var questionData models.ItemQuestion

				if formAccess.Status != "revised" {
					questionData = models.ItemQuestion{
						ID:       question.ID,
						Type:     question.Type,
						Question: question.Question,
						Points:   question.Points,
					}
					if question.Type != "written" {
						questionData.Answers = question.Answers
					}
				} else {
					questionData = question
				}
				answer, err := w.getAnswerStudent(idObjUser, idObjWork, question.ID)
				if err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
					*returnErr = err
					close(c)
					return
				}
				// Add answer
				if answer != nil {
					answers[iAnswer] = &AnswerRes{
						Answer: *answer,
					}
				}
				// Add evalute
				if work.IsRevised && question.Type == "written" {
					var evaluate []models.EvaluatedAnswersWLookup

					match := bson.D{{
						Key: "$match",
						Value: bson.M{
							"question": question.ID,
							"student":  idObjUser,
							"work":     idObjWork,
						},
					}}
					lookup := bson.D{{
						Key: "$lookup",
						Value: bson.M{
							"from":         models.USERS_COLLECTION,
							"localField":   "evaluator",
							"foreignField": "_id",
							"as":           "evaluator",
							"pipeline": bson.A{bson.D{{
								Key: "$project",
								Value: bson.M{
									"name":           1,
									"first_lastname": 1,
								},
							}}},
						},
					}}
					project := bson.D{{
						Key: "$project",
						Value: bson.M{
							"evaluator": bson.M{
								"$arrayElemAt": bson.A{"$evaluator", 0},
							},
							"points": 1,
							"date":   1,
						},
					}}
					cursor, err := evaluatedAnswersModel.Aggreagate(mongo.Pipeline{
						match,
						lookup,
						project,
					})
					if err != nil {
						*returnErr = err
						close(c)
						return
					}
					if err := cursor.All(db.Ctx, &evaluate); err != nil {
						*returnErr = err
						close(c)
						return
					}
					answers[iAnswer].Evaluate = evaluate[0]
				}
				// Add question
				questions[iQuestion] = questionData
				<-c
			}(&wg, question, questions, i, answerIndex, &err)
		}
		wg.Wait()
		if err != nil {
			return nil, &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
		newItems = append(newItems, models.FormItemWLookup{
			Title:      item.Title,
			PointsType: item.PointsType,
			Questions:  questions,
		})
	}

	form[0].Items = newItems
	// Calculate rest time
	var dateLimit time.Time
	if work.FormAccess == "wtime" {
		datePlusTime := formAccess.Date.Time().Add(time.Duration(work.TimeFormAccess * int(time.Second)))
		if datePlusTime.Before(work.DateLimit.Time()) {
			dateLimit = datePlusTime
		} else {
			dateLimit = work.DateLimit.Time()
		}
	}
	// Return response
	response := make(map[string]interface{})
	workResponse := make(map[string]interface{})
	workResponse["wtime"] = work.FormAccess == "wtime"
	workResponse["date_limit"] = dateLimit
	// Status
	isClosedWTime := work.FormAccess == "wtime" && time.Now().After(dateLimit)
	isClosed := time.Now().After(work.DateLimit.Time()) || formAccess.Status == "finished" || isClosedWTime
	if formAccess.Status == "revised" {
		workResponse["status"] = "revised"
	} else if isClosed {
		workResponse["status"] = "finished"
	} else {
		workResponse["status"] = "opened"
	}
	// Get points
	if work.IsRevised {
		var questionsWPoints []models.ItemQuestion
		maxPoints := 0
		for _, item := range form[0].Items {
			for _, question := range item.Questions {
				if question.Type != "alternatives" {
					maxPoints += question.Points
					questionsWPoints = append(questionsWPoints, question)
				}
			}
		}

		totalPoints, _, err := w.getStudentEvaluate(
			questionsWPoints,
			idObjUser,
			idObjWork,
		)
		if err != nil {
			return nil, &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
		// Points response
		pointsRes := make(map[string]interface{})
		pointsRes["max_points"] = maxPoints
		pointsRes["total_points"] = totalPoints

		response["points"] = pointsRes
	}
	// Response
	response["form"] = &form[0]
	response["answers"] = answers
	response["work"] = workResponse
	return response, nil
}

func (w *WorkSerice) GetFormStudent(
	idWork,
	idStudent string,
) (*models.FormWLookup, []AnswerRes, *res.ErrorRes) {
	// Recovery if close channel
	defer func() {
		recovery := recover()
		if recovery != nil {
			fmt.Printf("A channel closed")
		}
	}()

	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return nil, nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjStudent, err := primitive.ObjectIDFromHex(idStudent)
	if err != nil {
		return nil, nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get work
	work, err := workRepository.GetWorkFromId(idObjWork)
	if err != nil {
		return nil, nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if work.Type != "form" {
		return nil, nil, &res.ErrorRes{
			Err:        fmt.Errorf("el trabajo no es un formulario"),
			StatusCode: http.StatusBadRequest,
		}
	}
	if time.Now().Before(work.DateLimit.Time()) {
		return nil, nil, &res.ErrorRes{
			Err:        fmt.Errorf("este formulario todavía no se puede evaluar"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	// Get form
	form, errRes := formService.GetForm(work.Form.Hex(), primitive.NilObjectID.Hex(), false)
	if errRes != nil {
		return nil, nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Get access student
	formAccess, err := w.getAccessFromIdStudentNIdWork(idObjStudent, idObjWork)
	if err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
		return nil, nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if formAccess == nil {
		return nil, nil, &res.ErrorRes{
			Err:        fmt.Errorf("este alumno no ha tiene respuestas, ya que no abrió el formulario"),
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get answers
	questionsLen := 0
	for _, item := range form[0].Items {
		for range item.Questions {
			questionsLen += 1
		}
	}
	var answers = make([]AnswerRes, questionsLen)
	var wg sync.WaitGroup
	c := make(chan (int), 5)

	for i, item := range form[0].Items {
		for j, question := range item.Questions {
			wg.Add(1)
			c <- 1

			iAnswer := j
			for i != 0 {
				iAnswer += len(form[0].Items[i].Questions)
				i -= 1
			}
			go func(question models.ItemQuestion, iAnswer int, wg *sync.WaitGroup, errRet *error) {
				defer wg.Done()

				answer, err := w.getAnswerStudent(idObjStudent, idObjWork, question.ID)
				if err != nil {
					if err.Error() != db.NO_SINGLE_DOCUMENT {
						*errRet = err
					}
					close(c)
					return
				}
				// Get evaluate
				var evaluatedAnswer *models.EvaluatedAnswers
				if question.Type == "written" {
					cursor := evaluatedAnswersModel.GetOne(bson.D{
						{
							Key:   "question",
							Value: question.ID,
						},
						{
							Key:   "student",
							Value: idObjStudent,
						},
						{
							Key:   "work",
							Value: idObjWork,
						},
					})
					if err := cursor.Decode(&evaluatedAnswer); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
						*errRet = err
						close(c)
						return
					}
				}
				answers[iAnswer] = AnswerRes{
					Answer:   *answer,
					Evaluate: evaluatedAnswer,
				}
				<-c
			}(question, iAnswer, &wg, &err)
		}
	}
	wg.Wait()
	if err != nil {
		return nil, nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Get
	return &form[0], answers, nil
}

func (w *WorkSerice) getQuestionsFromIdForm(idForm primitive.ObjectID) ([]models.ItemQuestion, error) {
	// Get questions
	type QuestionsRes struct {
		ID        string                `bson:"_id"`
		Questions []models.ItemQuestion `bson:"questions"`
	}
	var questions []QuestionsRes
	match := bson.D{{
		Key: "$match",
		Value: bson.M{
			"_id": idForm,
		},
	}}
	unwindItems := bson.D{{
		Key: "$unwind",
		Value: bson.M{
			"path": "$items",
		},
	}}
	groupQuestionsArray := bson.D{{
		Key: "$group",
		Value: bson.M{
			"_id": "_id",
			"questions": bson.M{
				"$push": "$items.questions",
			},
		},
	}}
	unwindQuestions := bson.D{{
		Key: "$unwind",
		Value: bson.M{
			"path": "$questions",
		},
	}}
	groupQuestions := bson.D{{
		Key: "$group",
		Value: bson.M{
			"_id": "",
			"questions": bson.M{
				"$addToSet": "$questions",
			},
		},
	}}
	cursorQuestions, err := formModel.Aggreagate(mongo.Pipeline{
		match,
		unwindItems,
		formService.getLookupQuestions(),
		groupQuestionsArray,
		unwindQuestions,
		unwindQuestions,
		groupQuestions,
	})
	if err != nil {
		return nil, err
	}
	if err := cursorQuestions.All(db.Ctx, &questions); err != nil {
		return nil, err
	}
	return questions[0].Questions, nil
}
