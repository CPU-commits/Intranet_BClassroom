package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"mime/multipart"
	"strings"
	"sync"
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/stack"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zip"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var workService *WorkSerice

type WorkSerice struct{}

type CloseForm struct {
	Work    string
	Student string
	Diff    float64
}

type Student struct {
	ID                 string                               `json:"_id"`
	User               models.SimpleUser                    `json:"user"`
	V                  int                                  `json:"__v"`
	RegistrationNumber string                               `json:"registration_number"`
	Course             string                               `json:"course"`
	AccessForm         *models.FormAccess                   `json:"access"`
	FilesUploaded      *models.FileUploadedClassroomWLookup `json:"files_uploaded"`
	Evuluate           map[string]int                       `json:"evaluate"`
}

type AnswerRes struct {
	Answer   models.Answer `json:"answer"`
	Evaluate interface{}   `json:"evaluate,omitempty"`
}

func (w *WorkSerice) GetLookupUser() bson.D {
	return bson.D{
		{
			Key: "$lookup",
			Value: bson.M{
				"from":         models.USERS_COLLECTION,
				"localField":   "author",
				"foreignField": "_id",
				"as":           "author",
				"pipeline": bson.A{bson.M{
					"$project": bson.M{
						"name":           1,
						"first_lastname": 1,
					},
				}},
			},
		},
	}
}

func (w *WorkSerice) GetLookupGrade() bson.D {
	return bson.D{
		{
			Key: "$lookup",
			Value: bson.M{
				"from":         models.GRADES_PROGRAM_COLLECTION,
				"localField":   "grade",
				"foreignField": "_id",
				"as":           "grade",
				"pipeline": bson.A{bson.M{
					"$project": bson.M{
						"module": 0,
					},
				}},
			},
		},
	}
}

func (w *WorkSerice) GetWorks(idModule string) ([]models.WorkWLookup, error) {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return nil, err
	}
	// Get
	var works []models.WorkWLookup

	match := bson.D{
		{
			Key: "$match",
			Value: bson.M{
				"module": idObjModule,
			},
		},
	}
	lookupUser := w.GetLookupUser()
	lookupGrade := w.GetLookupGrade()
	project := bson.D{
		{
			Key: "$project",
			Value: bson.M{
				"title":        1,
				"is_qualified": 1,
				"type":         1,
				"date_start":   1,
				"date_limit":   1,
				"date_upload":  1,
				"is_revised":   1,
				"date_update":  1,
				"acumulative":  1,
				"author": bson.M{
					"$arrayElemAt": bson.A{"$author", 0},
				},
				"grade": bson.M{
					"$arrayElemAt": bson.A{"$grade", 0},
				},
			},
		},
	}
	order := bson.D{
		{
			Key: "$sort",
			Value: bson.M{
				"date_upload": -1,
			},
		},
	}
	cursor, err := workModel.Aggreagate(mongo.Pipeline{
		match,
		lookupUser,
		lookupGrade,
		project,
		order,
	})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(db.Ctx, &works); err != nil {
		return nil, err
	}

	for i, work := range works {
		if !work.Acumulative.IsZero() {
			var acumulative []models.Acumulative
			for _, acu := range work.Grade.Acumulative {
				if acu.ID == work.Acumulative {
					acumulative = append(acumulative, acu)
				}
			}
			works[i].Grade.Acumulative = acumulative
		}
	}
	return works, nil
}

func (w *WorkSerice) getWorkFromId(idWork primitive.ObjectID) (*models.Work, error) {
	var work *models.Work
	cursor := workModel.GetByID(idWork)
	if err := cursor.Decode(&work); err != nil {
		return nil, err
	}
	return work, nil
}

func (w *WorkSerice) GetWork(idWork string, claims *Claims) (map[string]interface{}, error) {
	idObjUser, err := primitive.ObjectIDFromHex(claims.ID)
	if err != nil {
		return nil, err
	}
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return nil, err
	}
	// Get
	var works []models.WorkWLookupNFiles

	match := bson.D{{
		Key: "$match",
		Value: bson.M{
			"_id": idObjWork,
		},
	}}
	setAuthorNGrade := bson.D{{
		Key: "$set",
		Value: bson.M{
			"author": bson.M{
				"$first": "$author",
			},
			"grade": bson.M{
				"$first": "$grade",
			},
		},
	}}
	unwindAttached := bson.D{{
		Key: "$unwind",
		Value: bson.M{
			"path":                       "$attached",
			"preserveNullAndEmptyArrays": true,
		},
	}}
	lookupFile := bson.D{{
		Key: "$lookup",
		Value: bson.M{
			"from":         models.FILES_COLLECTION,
			"localField":   "attached.file",
			"foreignField": "_id",
			"as":           "attached.file",
		},
	}}
	setFirstFile := bson.D{{
		Key: "$set",
		Value: bson.M{
			"attached.file": bson.M{
				"$first": "$attached.file",
			},
		},
	}}
	groupAttached := bson.D{{
		Key: "$group",
		Value: bson.M{
			"_id": "$_id",
			"attached": bson.M{
				"$push": "$attached",
			},
		},
	}}
	lookupWork := bson.D{{
		Key: "$lookup",
		Value: bson.M{
			"from":         models.WORKS_COLLECTION,
			"localField":   "_id",
			"foreignField": "_id",
			"as":           "result",
			"pipeline": bson.A{bson.D{{
				Key: "$project",
				Value: bson.M{
					"attached": 0,
				},
			}}},
		},
	}}
	unwindResult := bson.D{{
		Key: "$unwind",
		Value: bson.M{
			"path": "$result",
		},
	}}
	addFields := bson.D{{
		Key: "$addFields",
		Value: bson.M{
			"result.attached": "$attached",
		},
	}}
	replaceRoot := bson.D{{
		Key: "$replaceRoot",
		Value: bson.M{
			"newRoot": "$result",
		},
	}}

	cursor, err := workModel.Aggreagate(mongo.Pipeline{
		match,
		unwindAttached,
		lookupFile,
		setFirstFile,
		groupAttached,
		lookupWork,
		unwindResult,
		addFields,
		replaceRoot,
		w.GetLookupUser(),
		w.GetLookupGrade(),
		setAuthorNGrade,
	})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(db.Ctx, &works); err != nil {
		return nil, err
	}

	work := &works[0]
	if time.Now().Before(work.DateStart.Time()) && (claims.UserType == models.STUDENT || claims.UserType == models.STUDENT_DIRECTIVE) {
		return nil, fmt.Errorf("No se puede acceder a este trabajo todavía")
	}
	if !work.Acumulative.IsZero() {
		var acumulative []models.Acumulative
		for _, acu := range work.Grade.Acumulative {
			if acu.ID == work.Acumulative {
				acumulative = append(acumulative, acu)
			}
		}
		work.Grade.Acumulative = acumulative
	}
	// Response
	response := make(map[string]interface{})
	response["work"] = work
	// Get grade
	if work.IsRevised && (claims.UserType == models.STUDENT || claims.UserType == models.STUDENT_DIRECTIVE) {
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
				"grade": 1,
				"date":  1,
				"evaluator": bson.M{
					"$arrayElemAt": bson.A{"$evaluator", 0},
				},
			},
		}}
		if work.IsQualified {
			var grade []models.GradeWLookup
			match := bson.D{{
				Key: "$match",
				Value: bson.M{
					"module":  work.Module,
					"program": work.Grade.ID,
					"student": idObjUser,
				},
			}}
			project := bson.D{{
				Key: "$project",
				Value: bson.M{
					"grade": 1,
					"date":  1,
					"evaluator": bson.M{
						"$arrayElemAt": bson.A{"$evaluator", 0},
					},
				},
			}}
			cursor, err := gradeModel.Aggreagate(mongo.Pipeline{
				match,
				lookup,
				project,
			})
			if err != nil {
				return nil, err
			}
			if err := cursor.All(db.Ctx, &grade); err != nil {
				return nil, err
			}
			response["grade"] = grade[0]
		} else {
			var grade []models.WorkGradeWLookup
			match := bson.D{{
				Key: "$match",
				Value: bson.M{
					"module":  work.Module,
					"student": idObjUser,
					"work":    work.ID,
				},
			}}
			cursor, err := workGradeModel.Aggreagate(mongo.Pipeline{
				match,
				lookup,
				project,
			})
			if err != nil {
				return nil, err
			}
			if err := cursor.All(db.Ctx, &grade); err != nil {
				return nil, err
			}
			response["grade"] = grade[0]
		}
	}
	// Get form
	if work.Type == "form" {
		form, err := formService.GetFormById(work.Form)
		if err != nil {
			return nil, err
		}
		response["form_has_points"] = form.HasPoints
	}
	// Student
	if claims.UserType == models.STUDENT || claims.UserType == models.STUDENT_DIRECTIVE {
		if work.Type == "form" {
			// Student access
			idObjStudent, err := primitive.ObjectIDFromHex(claims.ID)
			if err != nil {
				return nil, err
			}
			formAccess, err := w.getAccessFromIdStudentNIdWork(
				idObjStudent,
				idObjWork,
			)
			if err != nil && err.Error() != "mongo: no documents in result" {
				return nil, err
			}
			response["form_access"] = formAccess
		} else if work.Type == "files" {
			// Get files uploaded
			fUC, err := w.getFilesUploadedStudent(idObjUser, idObjWork)
			if err != nil {
				return nil, err
			}
			if len(fUC) > 0 {
				response["files_uploaded"] = fUC[0]
			} else {
				response["files_uploaded"] = nil
			}
		}
	}
	return response, nil
}

func (w *WorkSerice) getAnswerStudent(student, work, question primitive.ObjectID) (*models.Answer, error) {
	var answer *models.Answer
	cursor := answerModel.GetOne(bson.D{
		{
			Key:   "student",
			Value: student,
		},
		{
			Key:   "work",
			Value: work,
		},
		{
			Key:   "question",
			Value: question,
		},
	})
	if err := cursor.Decode(&answer); err != nil {
		return nil, err
	}
	return answer, nil
}

func (w *WorkSerice) getAccessFromIdStudentNIdWork(idStudent, idWork primitive.ObjectID) (*models.FormAccess, error) {
	var formAccess *models.FormAccess
	cursor := formAccessModel.GetOne(bson.D{
		{
			Key:   "student",
			Value: idStudent,
		},
		{
			Key:   "work",
			Value: idWork,
		},
	})
	if err := cursor.Decode(&formAccess); err != nil {
		return nil, err
	}
	return formAccess, nil
}

func (w *WorkSerice) GetForm(idWork, userId string) (map[string]interface{}, error) {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return nil, err
	}
	idObjUser, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return nil, err
	}
	// Get work
	work, err := w.getWorkFromId(idObjWork)

	if work.Type != "form" {
		return nil, fmt.Errorf("Este trabajo no es de tipo formulario")
	}
	if time.Now().Before(work.DateStart.Time()) {
		return nil, fmt.Errorf("No se puede acceder a este trabajo todavía")
	}
	// Get form
	form, err := formService.GetForm(work.Form.Hex(), userId, false)
	// Get form access
	formAccess, err := w.getAccessFromIdStudentNIdWork(idObjUser, idObjWork)
	if err != nil && err.Error() != "mongo: no documents in result" {
		return nil, err
	}

	if formAccess == nil && time.Now().Before(work.DateLimit.Time()) {
		modelFormAccess := models.NewModelFormAccess(
			idObjUser,
			idObjWork,
		)
		inserted, err := formAccessModel.NewDocument(modelFormAccess)
		if err != nil {
			return nil, err
		}
		var diff time.Duration
		if work.FormAccess == "default" {
			diff = work.DateLimit.Time().Sub(time.Now())
		} else {
			diff = time.Duration(work.TimeFormAccess * int(time.Second))
		}
		err = nats.PublishEncode("close_student_form", &CloseForm{
			Work:    idWork,
			Student: userId,
			Diff:    diff.Hours(),
		})
		if err != nil {
			return nil, err
		}
		formAccess = &models.FormAccess{
			ID:      inserted.InsertedID.(primitive.ObjectID),
			Date:    primitive.NewDateTimeFromTime(time.Now()),
			Student: idObjUser,
			Work:    idObjWork,
			Status:  "opened",
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

	for j, item := range form[0].Items {
		var questions = make([]models.ItemQuestion, len(item.Questions))
		var err error

		for i, question := range item.Questions {
			wg.Add(1)

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
				if err != nil && err.Error() != "mongo: no documents in result" {
					*returnErr = err
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
						return
					}
					if err := cursor.All(db.Ctx, &evaluate); err != nil {
						*returnErr = err
						return
					}
					answers[iAnswer].Evaluate = evaluate[0]
				}
				// Add question
				questions[iQuestion] = questionData
			}(&wg, question, questions, i, answerIndex, &err)
		}
		wg.Wait()
		if err != nil {
			return nil, err
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
		// Points response
		pointsRes := make(map[string]interface{})
		pointsRes["max_points"] = maxPoints
		pointsRes["total_points"] = totalPoints

		response["points"] = pointsRes
		if err != nil {
			return nil, err
		}
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
) (*models.FormWLookup, []AnswerRes, error) {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return nil, nil, err
	}
	idObjStudent, err := primitive.ObjectIDFromHex(idStudent)
	if err != nil {
		return nil, nil, err
	}
	// Get work
	work, err := w.getWorkFromId(idObjWork)
	if err != nil {
		return nil, nil, err
	}
	if work.Type != "form" {
		return nil, nil, fmt.Errorf("El trabajo no es un formulario")
	}
	if time.Now().Before(work.DateLimit.Time()) {
		return nil, nil, fmt.Errorf("Este formulario todavía no se puede evaluar")
	}
	// Get form
	form, err := formService.GetForm(work.Form.Hex(), primitive.NilObjectID.Hex(), false)
	// Get access student
	formAccess, err := w.getAccessFromIdStudentNIdWork(idObjStudent, idObjWork)
	if err != nil && err.Error() != "mongo: no documents in result" {
		return nil, nil, err
	}
	if formAccess == nil {
		return nil, nil, fmt.Errorf("Este alumno no ha tiene respuestas, ya que no abrió el formulario")
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

	for i, item := range form[0].Items {
		for j, question := range item.Questions {
			wg.Add(1)

			iAnswer := j
			for i != 0 {
				iAnswer += len(form[0].Items[i].Questions)
				i -= 1
			}
			go func(question models.ItemQuestion, iAnswer int, wg *sync.WaitGroup, errRet *error) {
				defer wg.Done()

				answer, err := w.getAnswerStudent(idObjStudent, idObjWork, question.ID)
				if err != nil {
					if err.Error() != "mongo: no documents in result" {
						*errRet = err
					}
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
					if err := cursor.Decode(&evaluatedAnswer); err != nil && err.Error() != "mongo: no documents in result" {
						*errRet = err
						return
					}
				}
				answers[iAnswer] = AnswerRes{
					Answer:   *answer,
					Evaluate: evaluatedAnswer,
				}
			}(question, iAnswer, &wg, &err)
		}
	}
	wg.Wait()
	if err != nil {
		return nil, nil, err
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

func (w *WorkSerice) getStudentEvaluate(
	questions []models.ItemQuestion,
	idStudent,
	idWork primitive.ObjectID,
) (int, int, error) {
	var err error
	var wg sync.WaitGroup
	var lock sync.Mutex
	totalPoints := 0
	evaluatedSum := 0
	for _, question := range questions {
		wg.Add(1)
		go func(question models.ItemQuestion, wg *sync.WaitGroup, lock *sync.Mutex, errRet *error) {
			defer wg.Done()
			if question.Type == "alternatives_correct" {
				answer, err := w.getAnswerStudent(idStudent, idWork, question.ID)
				lock.Lock()
				evaluatedSum += 1
				lock.Unlock()
				if err != nil {
					if err.Error() != "mongo: no documents in result" {
						*errRet = err
					}
					return
				}
				if question.Correct == answer.Answer {
					lock.Lock()
					totalPoints += question.Points
					lock.Unlock()
				}
			} else if question.Type == "written" {
				var evaluateAnswer *models.EvaluatedAnswers
				cursor := evaluatedAnswersModel.GetOne(bson.D{
					{
						Key:   "student",
						Value: idStudent,
					},
					{
						Key:   "question",
						Value: question.ID,
					},
					{
						Key:   "work",
						Value: idWork,
					},
				})
				if err := cursor.Decode(&evaluateAnswer); err != nil && err.Error() != "mongo: no documents in result" {
					*errRet = err
					return
				}
				if evaluateAnswer != nil {
					lock.Lock()
					totalPoints += evaluateAnswer.Points
					evaluatedSum += 1
					lock.Unlock()
				}
			}
		}(question, &wg, &lock, &err)
	}
	wg.Wait()
	if err != nil {
		return 0, 0, err
	}
	percentage := (evaluatedSum * 100) / len(questions)
	return totalPoints, percentage, nil
}

func (w *WorkSerice) getStudentsFromIdModule(idModule string) ([]Student, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	module := map[string]string{
		"data": idModule,
		"id":   id.String(),
	}
	data, err := json.Marshal(module)
	if err != nil {
		return nil, err
	}
	var students []Student
	var response stack.NatsNestJSRes
	message, err := nats.Request(
		"get_students_from_module",
		data,
	)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(message.Data, &response)
	if err != nil {
		return nil, err
	}
	jsonString, err := json.Marshal(response.Response)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(jsonString, &students)
	return students, nil
}

func (w *WorkSerice) getFilesUploadedStudent(
	student,
	work primitive.ObjectID,
) ([]models.FileUploadedClassroomWLookup, error) {
	var fUC []models.FileUploadedClassroomWLookup
	match := bson.D{{
		Key: "$match",
		Value: bson.M{
			"work":    work,
			"student": student,
		},
	}}
	lookup := bson.D{{
		Key: "$lookup",
		Value: bson.M{
			"from":         models.FILES_COLLECTION,
			"localField":   "files_uploaded",
			"foreignField": "_id",
			"as":           "files_uploaded",
		},
	}}
	cursor, err := fileUCModel.Aggreagate(mongo.Pipeline{
		match,
		lookup,
	})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(db.Ctx, &fUC); err != nil {
		return nil, err
	}
	return fUC, nil
}

func (w *WorkSerice) GetStudentsStatus(idModule, idWork string) (interface{}, int, error) {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return nil, -1, err
	}
	// Get students
	students, err := w.getStudentsFromIdModule(idModule)
	if err != nil {
		return nil, -1, err
	}
	// Get work
	work, err := w.getWorkFromId(idObjWork)
	if err != nil {
		return nil, -1, err
	}
	if len(students) == 0 {
		return nil, -1, fmt.Errorf("Ningún estudiante pertenece a este trabajo")
	}
	// Get access of students
	var wg sync.WaitGroup
	var questionsWPoints []models.ItemQuestion
	if work.Type == "form" {
		questions, err := w.getQuestionsFromIdForm(work.Form)
		if err != nil {
			return nil, -1, err
		}
		for _, question := range questions {
			if question.Type != "alternatives" {
				questionsWPoints = append(questionsWPoints, question)
			}
		}
	}
	for i, student := range students {
		wg.Add(1)
		go func(student Student, index int, wg *sync.WaitGroup, errRet *error) {
			defer wg.Done()
			idObjStudent, _ := primitive.ObjectIDFromHex(student.User.ID)
			if work.Type == "form" {
				// Get access
				access, err := w.getAccessFromIdStudentNIdWork(
					idObjStudent,
					idObjWork,
				)
				if err != nil {
					if err.Error() != "mongo: no documents in result" {
						*errRet = err
					}
					return
				}
				students[index].AccessForm = access
				// Response evaluate
				pointsTotal, answereds, err := w.getStudentEvaluate(
					questionsWPoints,
					idObjStudent,
					idObjWork,
				)
				if err != nil {
					if err.Error() != "mongo: no documents in result" {
						*errRet = err
					}
					return
				}
				evaluate := map[string]int{
					"points_total": pointsTotal,
					"percentage":   answereds,
				}
				students[index].Evuluate = evaluate
			} else if work.Type == "files" {
				// Get files uploaded
				fUC, err := w.getFilesUploadedStudent(idObjStudent, idObjWork)
				if err != nil {
					*errRet = err
					return
				}
				if len(fUC) > 0 {
					students[index].FilesUploaded = &fUC[0]
				} else {
					students[index].FilesUploaded = nil
				}
			}
		}(student, i, &wg, &err)
	}
	wg.Wait()
	if err != nil {
		return nil, -1, err
	}
	// Total points
	var totalPoints int
	if work.Type == "form" {
		for _, question := range questionsWPoints {
			totalPoints += question.Points
		}
	} else if work.Type == "files" {
		for _, item := range work.Pattern {
			totalPoints += item.Points
		}
	}
	return students, totalPoints, nil
}

func (w *WorkSerice) getZipStudentWork() {

}

func (w *WorkSerice) DownloadFilesWorkStudent(idWork, idStudent string, writter io.Writer) (*zip.Writer, error) {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return nil, err
	}
	idObjStudent, err := primitive.ObjectIDFromHex(idStudent)
	if err != nil {
		return nil, err
	}
	// Get files
	fUC, err := w.getFilesUploadedStudent(idObjStudent, idObjWork)
	if err != nil {
		return nil, err
	}
	if len(fUC) == 0 {
		return nil, fmt.Errorf("No se pueden descargar archivos si no hay archivos subidos")
	}
	// Download files AWS
	type File struct {
		file io.ReadCloser
		name string
	}
	files := make([]File, len(fUC[0].FilesUploaded))
	var wg sync.WaitGroup
	for i, file := range fUC[0].FilesUploaded {
		wg.Add(1)
		go func(file models.File, i int, wg *sync.WaitGroup, errRet *error) {
			defer wg.Done()

			bytes, err := aws.GetFile(file.Key)
			if err != nil {
				*errRet = err
				return
			}
			files[i] = File{
				file: bytes,
				name: file.Filename,
			}
		}(file, i, &wg, &err)
	}
	wg.Wait()
	if err != nil {
		return nil, err
	}
	// Create zip archive
	zipWritter := zip.NewWriter(writter)
	for _, file := range files {
		zipFile, err := zipWritter.Create(file.name)
		if err != nil {
			return nil, err
		}
		body, err := ioutil.ReadAll(file.file)
		_, err = zipFile.Write(body)
		if err != nil {
			return nil, err
		}
	}
	return zipWritter, nil
}

func (w *WorkSerice) UploadWork(
	work *forms.WorkForm,
	idModule string,
	claims *Claims,
) error {
	idObjUser, err := primitive.ObjectIDFromHex(claims.ID)
	if err != nil {
		return err
	}
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return err
	}
	// Date
	tStart, err := time.Parse("2006-01-02T15:04:05Z", work.DateStart)
	if err != nil {
		return err
	}
	tLimit, err := time.Parse("2006-01-02T15:04:05Z", work.DateLimit)
	if err != nil {
		return err
	}
	if tStart.After(tLimit) {
		return fmt.Errorf("La fecha y hora de inicio es mayor a la limite")
	}
	// Grade
	if *work.IsQualified {
		idObjGrade, err := primitive.ObjectIDFromHex(work.Grade)
		if err != nil {
			return err
		}
		idObjModule, err := primitive.ObjectIDFromHex(idModule)
		if err != nil {
			return err
		}
		// Exists
		var grade *models.GradesProgram
		cursor := gradeProgramModel.GetByID(idObjGrade)
		if err := cursor.Decode(&grade); err != nil {
			if err.Error() == "mongo: no documents in result" {
				cursor = gradeProgramModel.GetOne(bson.D{{
					Key: "acumulative",
					Value: bson.M{
						"$elemMatch": bson.M{
							"_id": idObjGrade,
						},
					},
				}})
				if err := cursor.Decode(&grade); err != nil {
					return fmt.Errorf("No existe la calificación indicada")
				}
				work.Acumulative = idObjGrade
				work.Grade = grade.ID.Hex()
			} else {
				return err
			}
		}
		// Not used
		var work *models.Work
		cursor = workModel.GetOne(bson.D{
			{
				Key:   "module",
				Value: idObjModule,
			},
			{
				Key:   "grade",
				Value: idObjGrade,
			},
		})
		if err := cursor.Decode(&work); err != nil && err.Error() != "mongo: no documents in result" {
			return err
		}
		if work != nil {
			return fmt.Errorf("Esta calificación está registrada ya a un trabajo")
		}
	}
	// Form
	if work.Type == "form" {
		idObjForm, err := primitive.ObjectIDFromHex(work.Form)
		if err != nil {
			return err
		}
		var form *models.Form
		cursor := formModel.GetByID(idObjForm)
		if err := cursor.Decode(&form); err != nil {
			return fmt.Errorf("No existe el formulario indicado")
		}
		if form.Author != idObjUser {
			return fmt.Errorf("No tienes acceso a este formulario")
		}
		if *work.IsQualified && !form.HasPoints {
			return fmt.Errorf("Este formulario no tiene puntaje. Escoga uno con puntaje")
		}
	}
	// Insert
	modelWork, err := models.NewModelWork(work, tStart, tLimit, idObjModule, idObjUser)
	if err != nil {
		return err
	}
	insertedWork, err := workModel.NewDocument(modelWork)
	if err != nil {
		return err
	}
	// Insert Elasticsearch
	indexerWork := &models.WorkES{
		Title:     work.Title,
		DateStart: tStart,
		DateLimit: tLimit,
		Author:    claims.Name,
		IDModule:  idModule,
		Published: time.Now(),
	}
	data, err := json.Marshal(indexerWork)
	if err != nil {
		return err
	}
	// Add item to the BulkIndexer
	oid, _ := insertedWork.InsertedID.(primitive.ObjectID)
	bi, err := models.NewBulkWork()
	if err != nil {
		return err
	}
	err = bi.Add(
		context.Background(),
		esutil.BulkIndexerItem{
			Action:     "index",
			DocumentID: oid.Hex(),
			Body:       bytes.NewReader(data),
		},
	)
	if err != nil {
		return err
	}
	if err := bi.Close(context.Background()); err != nil {
		return err
	}
	return nil
}

func (w *WorkSerice) saveAnswer(
	answer *forms.AnswerForm,
	idObjWork,
	idObjQuestion,
	idObjStudent primitive.ObjectID,
) error {
	// If question belongs to form
	var form []models.Form

	match := bson.D{{
		Key: "$match",
		Value: bson.M{
			"_id": idObjWork,
			"items": bson.M{
				"$elemMatch": bson.M{
					"questions": bson.M{
						"$in": bson.A{idObjQuestion},
					},
				},
			},
		},
	}}
	formCursor, err := formModel.Aggreagate(mongo.Pipeline{match})
	if err != nil {
		return err
	}
	if err := formCursor.All(db.Ctx, &form); err != nil {
		return err
	}
	if formCursor == nil {
		return fmt.Errorf("La pregunta no pertenece al trabajo indicado")
	}
	// Get question
	var question *models.ItemQuestion
	cursor := formQuestionModel.GetByID(idObjQuestion)
	if err := cursor.Decode(&question); err != nil {
		return err
	}
	lenAnswers := len(question.Answers)
	if question.Type != "written" && (lenAnswers <= *answer.Answer || lenAnswers < 0) {
		return fmt.Errorf("Indique una respuesta válida")
	}
	// Get answer
	var answerData *models.Answer
	cursor = answerModel.GetOne(bson.D{
		{
			Key:   "student",
			Value: idObjStudent,
		},
		{
			Key:   "work",
			Value: idObjWork,
		},
		{
			Key:   "question",
			Value: idObjQuestion,
		},
	})
	if err := cursor.Decode(&answerData); err != nil && err.Error() != "mongo: no documents in result" {
		return err
	}
	// Save
	if answer.Answer == nil && answer.Response == "" {
		if answerData == nil {
			return fmt.Errorf("La respuesta no existe para ser eliminada")
		}
		_, err = answerModel.Use().DeleteOne(db.Ctx, bson.D{{
			Key:   "_id",
			Value: answerData.ID,
		}})
		if err != nil {
			return err
		}
	} else {
		if question.Type == "written" && answer.Response == "" {
			return fmt.Errorf("No se puede insertar una respuesta de alternativa a una pregunta de escritura")
		}
		if question.Type != "written" && answer.Answer == nil {
			return fmt.Errorf("No se puede insertar una respuesta escrita a una pregunta de alternativas")
		}
		if answerData == nil {
			modelAnswer := models.NewModelAnswer(
				answer,
				idObjStudent,
				idObjWork,
				idObjQuestion,
			)
			_, err = answerModel.NewDocument(modelAnswer)
			if err != nil {
				return err
			}
		} else {
			setBson := bson.M{}
			if answer.Answer != nil {
				setBson["answer"] = answer.Answer
			} else {
				setBson["response"] = answer.Response
			}
			_, err = answerModel.Use().UpdateByID(db.Ctx, answerData.ID, bson.D{{
				Key:   "$set",
				Value: setBson,
			}})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *WorkSerice) SaveAnswer(answer *forms.AnswerForm, idWork, idQuestion, idStudent string) error {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return err
	}
	idObjQuestion, err := primitive.ObjectIDFromHex(idQuestion)
	if err != nil {
		return err
	}
	idObjStudent, err := primitive.ObjectIDFromHex(idStudent)
	if err != nil {
		return err
	}
	// Get work
	work, err := w.getWorkFromId(idObjWork)
	if err != nil {
		return err
	}
	if work.Type != "form" {
		return fmt.Errorf("El trabajo no es de tipo formulario")
	}
	if time.Now().After(work.DateLimit.Time()) {
		return fmt.Errorf("Ya no se puede acceder al formulario")
	}
	// Get access
	formAcess, err := w.getAccessFromIdStudentNIdWork(
		idObjStudent,
		idObjWork,
	)
	if err != nil {
		return err
	}
	if formAcess.Status != "opened" {
		return fmt.Errorf("Ya no puedes acceder al formulario")
	}
	limitDate := formAcess.Date.Time().Add(time.Duration(work.TimeFormAccess * int(time.Second)))
	if work.FormAccess == "wtime" && time.Now().After(limitDate) {
		return fmt.Errorf("Ya no puedes acceder al formulario")
	}
	w.saveAnswer(answer, idObjWork, idObjQuestion, idObjStudent)
	return nil
}

func (w *WorkSerice) UploadFiles(files []*multipart.FileHeader, idWork, idUser string) error {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return err
	}
	idObjUser, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return err
	}
	if len(files) > 3 {
		return fmt.Errorf("Solo se puede subir hasta 3 archivos por trabajo")
	}
	// Get work
	work, err := w.getWorkFromId(idObjWork)
	if err != nil {
		return err
	}
	now := time.Now()
	if now.Before(work.DateStart.Time()) {
		return fmt.Errorf("Todavía no se puede acceder a este trabajo")
	}
	if now.After(work.DateLimit.Time().Add(7*24*time.Hour)) || work.IsRevised {
		return fmt.Errorf("Ya no se pueden subir archivos a este trabajo")
	}
	// Get files uploaded
	var fUC *models.FileUploadedClassroom
	cursor := fileUCModel.GetOne(bson.D{
		{
			Key:   "work",
			Value: idObjWork,
		},
		{
			Key:   "student",
			Value: idObjUser,
		},
	})
	if err := cursor.Decode(&fUC); err != nil && err.Error() != "mongo: no documents in result" {
		return err
	}
	if fUC != nil && len(fUC.FilesUploaded)+len(files) > 3 {
		return fmt.Errorf("Solo se puede subir hasta 3 archivos por trabajo")
	}
	// UploadFiles
	filesIds := make([]primitive.ObjectID, len(files))
	var wg sync.WaitGroup

	type FileNats struct {
		Location string `json:"location"`
		Filename string `json:"filename"`
		Mimetype string `json:"mime-type"`
		Key      string `json:"key"`
	}
	for i, file := range files {
		wg.Add(1)
		go func(file multipart.FileHeader, i int, wg *sync.WaitGroup, errRet *error) {
			defer wg.Done()
			// Upload files to S3
			result, err := aws.UploadFile(&file)
			if err != nil {
				*errRet = err
				return
			}
			// Request NATS (Get id file insert)
			locationSplit := strings.Split(result.Location, "/")
			key := fmt.Sprintf(
				"%s/%s",
				locationSplit[len(locationSplit)-2],
				locationSplit[len(locationSplit)-1],
			)
			data, err := json.Marshal(FileNats{
				Location: result.Location,
				Mimetype: file.Header.Get("Content-Type"),
				Filename: file.Filename,
				Key:      key,
			})
			if err != nil {
				*errRet = err
				return
			}
			msg, err := nats.Request("upload_files_classroom", data)
			if err != nil {
				*errRet = err
				return
			}
			// Process response NATS
			var fileDb *models.FileDB
			err = json.Unmarshal(msg.Data, &fileDb)
			if err != nil {
				*errRet = err
				return
			}
			idObjFile, err := primitive.ObjectIDFromHex(fileDb.ID.OID)
			if err != nil {
				*errRet = err
				return
			}
			filesIds[i] = idObjFile
		}(*file, i, &wg, &err)
	}
	wg.Wait()
	if err != nil {
		return err
	}
	if fUC == nil {
		modelFileUC := models.NewModelFileUC(
			idObjWork,
			idObjUser,
			filesIds,
		)
		_, err = fileUCModel.NewDocument(modelFileUC)
		if err != nil {
			return err
		}
	} else {
		_, err = fileUCModel.Use().UpdateByID(db.Ctx, fUC.ID, bson.D{{
			Key: "$push",
			Value: bson.M{
				"files_uploaded": bson.M{
					"$each": filesIds,
				},
			},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *WorkSerice) FinishForm(answers *forms.AnswersForm, idWork, idStudent string) error {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return err
	}
	idObjStudent, err := primitive.ObjectIDFromHex(idStudent)
	if err != nil {
		return err
	}
	// Get work
	work, err := w.getWorkFromId(idObjWork)
	if err != nil {
		return err
	}
	now := time.Now()
	if work.DateLimit.Time().Add(time.Minute * 5).Before(now) {
		return fmt.Errorf("Ya no se pueden modificar las respuestas de este formulario")
	}
	// Save answers
	var wg sync.WaitGroup
	for _, answer := range answers.Answers {
		idObjQuestion, err := primitive.ObjectIDFromHex(answer.Question)
		if err != nil {
			return err
		}
		if answer.Answer != nil && answer.Response != "" {
			wg.Add(1)
			answer := &forms.AnswerForm{
				Answer:   answer.Answer,
				Response: answer.Response,
			}
			go func(
				answer *forms.AnswerForm,
				idObjWork,
				idObjStudent,
				idObjQuestion primitive.ObjectID,
				wg *sync.WaitGroup,
				errRes *error,
			) {
				defer wg.Done()
				err := w.saveAnswer(answer, idObjWork, idObjQuestion, idObjStudent)
				if err != nil {
					*errRes = err
				}
			}(answer, idObjWork, idObjStudent, idObjQuestion, &wg, &err)
		}
	}
	wg.Wait()
	if err != nil {
		return err
	}
	// Update student access status
	_, err = formAccessModel.Use().UpdateOne(
		db.Ctx,
		bson.D{
			{
				Key:   "student",
				Value: idObjStudent,
			},
			{
				Key:   "work",
				Value: idObjWork,
			},
		},
		bson.D{{
			Key: "$set",
			Value: bson.M{
				"status": "finished",
			},
		}},
	)
	if err != nil {
		return err
	}
	return nil
}

func (w *WorkSerice) UploadPointsStudent(
	points int,
	idEvaluator,
	idWork,
	idQuestion,
	idStudent string,
) error {
	idObjEvaluator, err := primitive.ObjectIDFromHex(idEvaluator)
	if err != nil {
		return err
	}
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return err
	}
	idObjQuestion, err := primitive.ObjectIDFromHex(idQuestion)
	if err != nil {
		return err
	}
	idObjStudent, err := primitive.ObjectIDFromHex(idStudent)
	if err != nil {
		return err
	}
	// Get work
	work, err := w.getWorkFromId(idObjWork)
	if err != nil {
		return err
	}
	if time.Now().Before(work.DateLimit.Time()) {
		return fmt.Errorf("Todavía no se pueden evaluar preguntas en este formulario")
	}
	// Get form
	form, err := formService.GetFormById(work.Form)
	if err != nil {
		return err
	}
	if !form.HasPoints {
		return fmt.Errorf("No se puede evaluar un formulario sin puntos")
	}
	// Get question
	var question *models.ItemQuestion
	cursor := formQuestionModel.GetByID(idObjQuestion)
	if err := cursor.Decode(&question); err != nil {
		return err
	}
	if points < 0 || points > question.Points {
		return fmt.Errorf("Puntaje fuera de rango. Debe ser entre cero y máx %v", question.Points)
	}
	// Form Access
	_, err = w.getAccessFromIdStudentNIdWork(
		idObjStudent,
		idObjWork,
	)
	if err != nil {
		return err
	}
	// Get evaluate
	var evaluatedAnswer *models.EvaluatedAnswers
	cursor = evaluatedAnswersModel.GetOne(bson.D{
		{
			Key:   "work",
			Value: idObjWork,
		},
		{
			Key:   "question",
			Value: idObjQuestion,
		},
		{
			Key:   "student",
			Value: idObjStudent,
		},
	})
	if err := cursor.Decode(&evaluatedAnswer); err != nil && err.Error() != "mongo: no documents in result" {
		return err
	}
	// Upload points
	if evaluatedAnswer != nil {
		_, err = evaluatedAnswersModel.Use().UpdateByID(db.Ctx, evaluatedAnswer.ID, bson.D{{
			Key: "$set",
			Value: bson.M{
				"points": points,
				"date":   primitive.NewDateTimeFromTime(time.Now()),
			},
		}})
		if err != nil {
			return err
		}
	} else {
		modelEvaluatedAnswer := models.NewModelEvaluatedAnswers(
			points,
			idObjStudent,
			idObjQuestion,
			idObjWork,
			idObjEvaluator,
		)
		_, err = evaluatedAnswersModel.NewDocument(modelEvaluatedAnswer)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *WorkSerice) UploadEvaluateFiles(evalute []forms.EvaluateFilesForm, idWork, idStudent string) error {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return err
	}
	idObjStudent, err := primitive.ObjectIDFromHex(idStudent)
	if err != nil {
		return err
	}
	// Get work
	work, err := w.getWorkFromId(idObjWork)
	if err != nil {
		return err
	}
	if work.Type != "files" {
		return fmt.Errorf("Este trabajo no es de tipo archivos")
	}
	now := time.Now()
	if now.Before(work.DateLimit.Time()) {
		return fmt.Errorf("Todavía no se puede evaluar el trabajo")
	}
	if work.IsRevised {
		return fmt.Errorf("Ya no se puede actualizar el puntaje del alumno")
	}
	// Get evaluate student
	var fUC *models.FileUploadedClassroom
	cursor := fileUCModel.GetOne(bson.D{
		{
			Key:   "student",
			Value: idObjStudent,
		},
		{
			Key:   "work",
			Value: idObjWork,
		},
	})
	if err := cursor.Decode(&fUC); err != nil {
		return fmt.Errorf("No se encontraron archivos subidos por parte del alumno")
	}
	// Build model
	var evaluateFiles []interface{}
	for _, ev := range evalute {
		idObjPattern, err := primitive.ObjectIDFromHex(ev.Pattern)
		if err != nil {
			return err
		}
		exists := false
		for _, item := range work.Pattern {
			if item.ID == idObjPattern {
				if item.Points < *ev.Points {
					return fmt.Errorf("Los puntos evaluados superan el máx. del item")
				}
				exists = true
				break
			}
		}
		if !exists {
			return fmt.Errorf("No existe el item #%s en este trabajo", ev.Pattern)
		}
		// Get evaluate files
		var idEvaluate primitive.ObjectID
		uploaded := false
		for _, item := range fUC.Evaluate {
			if item.Pattern == idObjPattern {
				idEvaluate = item.ID
				uploaded = true
				break
			}
		}
		// Upload
		if !uploaded {
			evaluateFiles = append(evaluateFiles, models.EvaluatedFiles{
				ID:      primitive.NewObjectID(),
				Pattern: idObjPattern,
				Points:  *ev.Points,
			})
		} else {
			_, err = fileUCModel.Use().UpdateOne(
				db.Ctx,
				bson.D{
					{
						Key:   "_id",
						Value: fUC.ID,
					},
					{
						Key: "evaluate",
						Value: bson.M{
							"$elemMatch": bson.M{
								"_id": idEvaluate,
							},
						},
					},
				},
				bson.D{{
					Key: "$set",
					Value: bson.M{
						"evaluate.$.points": *ev.Points,
					},
				}},
			)
			if err != nil {
				return err
			}
		}
	}
	if len(evaluateFiles) > 0 {
		_, err = fileUCModel.Use().UpdateByID(db.Ctx, fUC.ID, bson.D{{
			Key: "$push",
			Value: bson.M{
				"evaluate": bson.M{
					"$each": evaluateFiles,
				},
			},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *WorkSerice) TransformPointsToGrade(
	scale float32,
	minGrade,
	points int,
) float64 {
	grade := (scale * float32(points)) + float32(minGrade)
	finalGrade := math.Round(float64(grade)*10) / 10
	return finalGrade
}

func (w *WorkSerice) GradeForm(idWork, idUser string) error {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return err
	}
	idObjUser, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return err
	}
	// Get work
	work, err := w.getWorkFromId(idObjWork)
	if err != nil {
		return err
	}
	if work.Type != "form" {
		return fmt.Errorf("El trabajo no es de tipo formulario")
	}
	if time.Now().Before(work.DateLimit.Time()) {
		return fmt.Errorf("Este formulario todavía no se puede calificar")
	}
	if work.IsRevised {
		return fmt.Errorf("Este trabajo ya está evaluado")
	}
	// Get form
	var form *models.Form
	cursor := formModel.GetByID(work.Form)
	if err := cursor.Decode(&form); err != nil {
		return err
	}
	if !form.HasPoints {
		return fmt.Errorf("Este formulario no puede ser calificado, el formulario no tiene puntos")
	}
	// Get student
	students, err := w.getStudentsFromIdModule(work.Module.Hex())
	if err != nil {
		return err
	}
	if len(students) == 0 {
		return fmt.Errorf("No existen alumnos a evaluar en este trabajo")
	}
	// Get questions form
	var questionsWPoints []models.ItemQuestion
	questions, err := w.getQuestionsFromIdForm(work.Form)
	if err != nil {
		return err
	}
	for _, question := range questions {
		if question.Type != "alternatives" {
			questionsWPoints = append(questionsWPoints, question)
		}
	}
	// Get evaluted students
	type StudentPoints struct {
		ID     primitive.ObjectID
		Points int
	}
	var studentsPoints []StudentPoints
	// To add evaluated status
	var studentsWithoutAccess []primitive.ObjectID
	// /To add evaluated status
	var wg sync.WaitGroup
	var lock sync.Mutex
	for _, student := range students {
		wg.Add(1)
		go func(student Student, wg *sync.WaitGroup, lock *sync.Mutex, errRet *error) {
			defer wg.Done()

			idObjStudent, err := primitive.ObjectIDFromHex(student.User.ID)
			if err != nil {
				*errRet = err
				return
			}
			// Get form access
			_, err = w.getAccessFromIdStudentNIdWork(
				idObjStudent,
				idObjWork,
			)
			if err != nil {
				if err.Error() != "mongo: no documents in result" {
					*errRet = err
					return
				}
				lock.Lock()
				studentsWithoutAccess = append(studentsWithoutAccess, idObjStudent)
				studentsPoints = append(studentsPoints, StudentPoints{
					ID:     idObjStudent,
					Points: 0,
				})
				lock.Unlock()
				return
			}
			// Get evaluate
			points, prom, err := w.getStudentEvaluate(
				questionsWPoints,
				idObjStudent,
				idObjWork,
			)
			if err != nil {
				*errRet = err
				return
			}
			if prom != 100 {
				*errRet = fmt.Errorf("No todos los alumnos están completamente evaluados")
				return
			}
			lock.Lock()
			studentsPoints = append(studentsPoints, StudentPoints{
				ID:     idObjStudent,
				Points: points,
			})
			lock.Unlock()
		}(student, &wg, &lock, &err)
	}
	wg.Wait()
	if err != nil {
		return err
	}
	// Get min max grade
	minGrade, maxGrade, err := GetMinNMaxGrade()
	if err != nil {
		return err
	}
	// Transform grades
	type StudentGrades struct {
		ID    primitive.ObjectID
		Grade float64
	}
	var studentsGrade []StudentGrades
	var maxPoints int
	for _, question := range questionsWPoints {
		maxPoints += question.Points
	}

	var scale float32 = float32(maxGrade-minGrade) / float32(maxPoints)
	for _, student := range studentsPoints {
		grade := w.TransformPointsToGrade(
			scale,
			minGrade,
			student.Points,
		)
		studentsGrade = append(studentsGrade, StudentGrades{
			ID:    student.ID,
			Grade: grade,
		})
	}
	// Update and insert status
	if studentsWithoutAccess != nil {
		var insertStudents []interface{}
		for _, idStudent := range studentsWithoutAccess {
			modelAccess := models.NewModelFormAccess(
				idStudent,
				idObjWork,
			)
			insertStudents = append(insertStudents, modelAccess)
		}
		_, err = formAccessModel.Use().InsertMany(db.Ctx, insertStudents)
		if err != nil {
			return err
		}
	}
	_, err = formAccessModel.Use().UpdateMany(
		db.Ctx,
		bson.D{{
			Key:   "work",
			Value: idObjWork,
		}},
		bson.D{{
			Key: "$set",
			Value: bson.M{
				"status": "revised",
			},
		}},
	)
	if err != nil {
		return err
	}
	// Update work status
	_, err = workModel.Use().UpdateByID(db.Ctx, idObjWork, bson.D{{
		Key: "$set",
		Value: bson.M{
			"is_revised": true,
		},
	}})
	if err != nil {
		return err
	}
	// Insert grades
	if work.IsQualified {
		// Get grade
		isAcumulative := false
		var grade *models.Grade
		cursor := gradeModel.GetByID(work.Grade)
		if err := cursor.Decode(&grade); err != nil {
			if err.Error() != "mongo: no documents in result" {
				return nil
			}
			isAcumulative = true
		}
		// Generate models
		var modelsGrades []interface{}
		for _, student := range studentsGrade {
			modelGrade := models.NewModelGrade(
				work.Module,
				student.ID,
				work.Grade,
				idObjUser,
				student.Grade,
				isAcumulative,
			)
			modelsGrades = append(modelsGrades, modelGrade)
		}
		_, err = gradeModel.Use().InsertMany(db.Ctx, modelsGrades)
		if err != nil {
			return err
		}
	} else {
		// Generate models
		var modelsGrades []interface{}
		for _, student := range studentsGrade {
			modelWorkGrade := models.NewModelWorkGrade(
				work.Module,
				student.ID,
				idObjUser,
				student.Grade,
			)
			modelsGrades = append(modelsGrades, modelWorkGrade)
		}
		_, err := workGradeModel.Use().InsertMany(db.Ctx, modelsGrades)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *WorkSerice) GradeFiles(idWork, idUser string) error {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return err
	}
	idObjUser, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return err
	}
	// Get work
	work, err := w.getWorkFromId(idObjWork)
	if err != nil {
		return err
	}
	if work.Type != "files" {
		return fmt.Errorf("El trabajo no es de tipo archivos")
	}
	if time.Now().Before(work.DateLimit.Time()) {
		return fmt.Errorf("Este trabajo todavía no se puede calificar")
	}
	if work.IsRevised {
		return fmt.Errorf("Este trabajo ya está evaluado")
	}
	// Get student
	students, err := w.getStudentsFromIdModule(work.Module.Hex())
	if err != nil {
		return err
	}
	if len(students) == 0 {
		return fmt.Errorf("No existen alumnos a evaluar en este trabajo")
	}
	// Get points student
	type StudentPoints struct {
		Student primitive.ObjectID
		Points  int
	}

	var studentsPoints []StudentPoints
	var wg sync.WaitGroup
	var lock sync.Mutex
	for _, student := range students {
		wg.Add(1)
		go func(student Student, wg *sync.WaitGroup, lock *sync.Mutex, errRet *error) {
			defer wg.Done()

			idObjStudent, err := primitive.ObjectIDFromHex(student.User.ID)
			if err != nil {
				*errRet = err
				return
			}
			// Get files uploaded W Points
			var fUC *models.FileUploadedClassroom
			cursor := fileUCModel.GetOne(bson.D{
				{
					Key:   "work",
					Value: idObjWork,
				},
				{
					Key:   "student",
					Value: idObjStudent,
				},
			})
			if err := cursor.Decode(&fUC); err != nil {
				if err.Error() != "mongo: no documents in result" {
					lock.Lock()
					studentsPoints = append(studentsPoints, StudentPoints{
						Student: idObjStudent,
						Points:  0,
					})
					lock.Unlock()
					*errRet = err
				}
				return
			}
			// Evaluate
			if len(fUC.Evaluate) != len(work.Pattern) {
				*errRet = fmt.Errorf("No todos los alumnos están completamente evaluados con todos los items")
				return
			}
			// Push points
			var points int
			for _, item := range fUC.Evaluate {
				points += item.Points
			}

			lock.Lock()
			studentsPoints = append(studentsPoints, StudentPoints{
				Student: idObjStudent,
				Points:  points,
			})
			lock.Unlock()
		}(student, &wg, &lock, &err)
	}
	wg.Wait()
	if err != nil {
		return err
	}
	// Get min max grade
	minGrade, maxGrade, err := GetMinNMaxGrade()
	if err != nil {
		return err
	}
	// Transform grades
	type StudentGrades struct {
		ID    primitive.ObjectID
		Grade float64
	}
	var studentsGrade []StudentGrades
	var maxPoints int
	for _, item := range work.Pattern {
		maxPoints += item.Points
	}

	var scale float32 = float32(maxGrade-minGrade) / float32(maxPoints)
	for _, student := range studentsPoints {
		grade := w.TransformPointsToGrade(
			scale,
			minGrade,
			student.Points,
		)
		studentsGrade = append(studentsGrade, StudentGrades{
			ID:    student.Student,
			Grade: grade,
		})
	}
	// Update work status
	_, err = workModel.Use().UpdateByID(db.Ctx, idObjWork, bson.D{{
		Key: "$set",
		Value: bson.M{
			"is_revised": true,
		},
	}})
	if err != nil {
		return err
	}
	// Insert grades
	if work.IsQualified {
		// Get grade
		isAcumulative := false
		var grade *models.Grade
		cursor := gradeModel.GetByID(work.Grade)
		if err := cursor.Decode(&grade); err != nil {
			if err.Error() != "mongo: no documents in result" {
				return nil
			}
			isAcumulative = true
		}
		// Generate models
		var modelsGrades []interface{}
		for _, student := range studentsGrade {
			modelGrade := models.NewModelGrade(
				work.Module,
				student.ID,
				work.Grade,
				idObjUser,
				student.Grade,
				isAcumulative,
			)
			modelsGrades = append(modelsGrades, modelGrade)
		}
		_, err = gradeModel.Use().InsertMany(db.Ctx, modelsGrades)
		if err != nil {
			return err
		}
	} else {
		// Generate models
		var modelsGrades []interface{}
		for _, student := range studentsGrade {
			modelWorkGrade := models.NewModelWorkGrade(
				work.Module,
				student.ID,
				idObjUser,
				student.Grade,
			)
			modelsGrades = append(modelsGrades, modelWorkGrade)
		}
		_, err := workGradeModel.Use().InsertMany(db.Ctx, modelsGrades)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *WorkSerice) DeleteFileClassroom(idWork, idFile, idUser string) error {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return err
	}
	idObjFile, err := primitive.ObjectIDFromHex(idFile)
	if err != nil {
		fmt.Printf("idFile: %v\n", idFile)
		return err
	}
	idObjUser, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return err
	}
	// Get files uploaded
	var fUC *models.FileUploadedClassroom
	cursor := fileUCModel.GetOne(bson.D{
		{
			Key:   "work",
			Value: idObjWork,
		},
		{
			Key:   "student",
			Value: idObjUser,
		},
	})
	if err := cursor.Decode(&fUC); err != nil {
		return err
	}
	updated := false
	for _, file := range fUC.FilesUploaded {
		if file == idObjFile {
			// Delete file AWS
			msg, err := nats.Request("get_key_from_id_file", []byte(idFile))
			if err != nil {
				return err
			}

			keyFile := string(msg.Data[:])
			err = aws.DeleteFile(keyFile)
			if err != nil {
				return err
			}
			// Delete file classroom
			if len(fUC.FilesUploaded) == 1 {
				_, err = fileUCModel.Use().DeleteOne(db.Ctx, bson.D{{
					Key:   "_id",
					Value: fUC.ID,
				}})
				if err != nil {
					return err
				}
			} else {
				_, err = fileUCModel.Use().UpdateByID(db.Ctx, fUC.ID, bson.D{{
					Key: "$pull",
					Value: bson.M{
						"files_uploaded": idObjFile,
					},
				}})
				if err != nil {
					return err
				}
			}
			updated = true
			break
		}
	}
	if !updated {
		return fmt.Errorf("No se encontró el archivo a eliminar en este trabajo")
	}
	return nil
}

func NewWorksService() *WorkSerice {
	if publicationService == nil {
		workService = &WorkSerice{}
	}
	return workService
}
