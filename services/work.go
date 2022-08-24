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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/stack"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/klauspost/compress/zip"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var workService *WorkSerice

type WorkSerice struct{}

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

func (w *WorkSerice) GetModulesWorks(claims Claims) (interface{}, error) {
	idObjUser, err := primitive.ObjectIDFromHex(claims.ID)
	if err != nil {
		return nil, err
	}

	// Get modules
	courses, err := FindCourses(&claims)
	if err != nil {
		return nil, err
	}
	modules, err := moduleService.GetModules(courses, claims.UserType, true)
	if err != nil {
		return nil, err
	}
	var modulesOr bson.A
	for _, module := range modules {
		modulesOr = append(modulesOr, bson.M{
			"module": module.ID,
		})
	}
	// Get works
	var works []models.Work
	match := bson.D{{
		Key: "$match",
		Value: bson.M{
			"$or":        modulesOr,
			"is_revised": false,
			"date_limit": bson.M{
				"$gte": primitive.NewDateTimeFromTime(time.Now()),
			},
		},
	}}
	sortA := bson.D{{
		Key: "$sort",
		Value: bson.M{
			"date_limit": 1,
		},
	}}
	project := bson.D{{
		Key: "$project",
		Value: bson.M{
			"title":        1,
			"is_qualified": 1,
			"type":         1,
			"date_start":   1,
			"date_limit":   1,
			"date_upload":  1,
			"module":       1,
			"_id":          1,
		},
	}}
	cursor, err := workModel.Aggreagate(mongo.Pipeline{
		match,
		sortA,
		project,
	})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(db.Ctx, &works); err != nil {
		return nil, err
	}
	// Get status
	type WorkStatus struct {
		Title       string    `json:"title"`
		IsQualified bool      `json:"is_qualified"`
		Type        string    `json:"type"`
		Module      string    `json:"module"`
		ID          string    `json:"_id"`
		DateStart   time.Time `json:"date_start"`
		DateLimit   time.Time `json:"date_limit"`
		DateUpload  time.Time `json:"date_upload"`
		Status      int       `json:"status"`
	}
	workStatus := make([]WorkStatus, len(works))
	var wg sync.WaitGroup

	for i, work := range works {
		wg.Add(1)

		go func(work models.Work, i int, wg *sync.WaitGroup, errRet *error) {
			defer wg.Done()

			workStatus[i] = WorkStatus{
				Title:       work.Title,
				Module:      work.Module.Hex(),
				ID:          work.ID.Hex(),
				IsQualified: work.IsQualified,
				Type:        work.Type,
				DateStart:   work.DateStart.Time(),
				DateLimit:   work.DateLimit.Time(),
				DateUpload:  work.DateUpload.Time(),
			}
			if work.Type == "files" {
				var fUC *models.FileUploadedClassroom

				cursor := fileUCModel.GetOne(bson.D{
					{
						Key:   "work",
						Value: work.ID,
					},
					{
						Key:   "student",
						Value: idObjUser,
					},
				})
				if err := cursor.Decode(&fUC); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
					*errRet = err
					return
				}
				if fUC != nil {
					workStatus[i].Status = 2
				}
			} else if work.Type == "form" {
				var formAccess *models.FormAccess

				cursor := formAccessModel.GetOne(bson.D{
					{
						Key:   "work",
						Value: work.ID,
					},
					{
						Key:   "student",
						Value: idObjUser,
					},
				})
				if err := cursor.Decode(&formAccess); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
					*errRet = err
					return
				}
				if formAccess != nil {
					if formAccess.Status == "finished" {
						workStatus[i].Status = 2
					} else if formAccess.Status == "opened" {
						workStatus[i].Status = 1
					}
				}
			}
		}(work, i, &wg, &err)
	}
	wg.Wait()
	if err != nil {
		return nil, err
	}
	// Order by status asc
	sort.Slice(workStatus, func(i, j int) bool {
		return workStatus[i].Status < workStatus[j].Status
	})
	return workStatus, nil
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

			matchValue := bson.M{
				"module":  work.Module,
				"program": work.Grade.ID,
				"student": idObjUser,
			}
			if work.Grade.IsAcumulative {
				matchValue["acumulative"] = work.Acumulative
			}
			match := bson.D{{
				Key:   "$match",
				Value: matchValue,
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
			if err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
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
	if err != nil {
		return nil, err
	}

	if work.Type != "form" {
		return nil, fmt.Errorf("Este trabajo no es de tipo formulario")
	}
	if time.Now().Before(work.DateStart.Time()) {
		return nil, fmt.Errorf("No se puede acceder a este trabajo todavía")
	}
	// Get form
	form, err := formService.GetForm(work.Form.Hex(), userId, false)
	if err != nil {
		return nil, err
	}
	// Get form access
	formAccess, err := w.getAccessFromIdStudentNIdWork(idObjUser, idObjWork)
	if err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
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
	if formAccess == nil && time.Now().After(work.DateLimit.Time()) {
		return nil, fmt.Errorf("No accediste al formulario, no hay respuestas a revisar")
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
				if err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
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
	if err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
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
					if err.Error() != db.NO_SINGLE_DOCUMENT {
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
					if err := cursor.Decode(&evaluatedAnswer); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
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
					if err.Error() != db.NO_SINGLE_DOCUMENT {
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
				if err := cursor.Decode(&evaluateAnswer); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
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
	data, err := formatRequestToNestjsNats(idModule)
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
					if err.Error() != db.NO_SINGLE_DOCUMENT {
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
					if err.Error() != db.NO_SINGLE_DOCUMENT {
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
		if err != nil {
			return nil, err
		}
		_, err = zipFile.Write(body)
		if err != nil {
			return nil, err
		}
	}
	return zipWritter, nil
}

type Grade struct {
	Acumulative   primitive.ObjectID
	Grade         primitive.ObjectID
	IsAcumulative bool
}

func (w *WorkSerice) verifyGradeWork(idObjModule, idObjGrade primitive.ObjectID) (*Grade, error) {
	gradeRet := Grade{}
	// Exists
	var grade *models.GradesProgram
	cursor := gradeProgramModel.GetByID(idObjGrade)
	if err := cursor.Decode(&grade); err != nil {
		if err.Error() == db.NO_SINGLE_DOCUMENT {
			cursor = gradeProgramModel.GetOne(bson.D{{
				Key: "acumulative",
				Value: bson.M{
					"$elemMatch": bson.M{
						"_id": idObjGrade,
					},
				},
			}})
			if err := cursor.Decode(&grade); err != nil {
				return nil, fmt.Errorf("No existe la calificación indicada")
			}
			gradeRet.Acumulative = idObjGrade
			gradeRet.Grade = grade.ID
			gradeRet.IsAcumulative = true
		} else {
			return nil, err
		}
	}
	gradeRet.Grade = grade.ID
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
	if err := cursor.Decode(&work); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
		return nil, err
	}
	if work != nil {
		return nil, fmt.Errorf("Esta calificación está registrada ya a un trabajo")
	}
	return &gradeRet, nil
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
	tStart, err := time.Parse("2006-01-02 15:04", work.DateStart)
	if err != nil {
		return err
	}
	tLimit, err := time.Parse("2006-01-02 15:04", work.DateLimit)
	if err != nil {
		return err
	}
	if tStart.After(tLimit) {
		return fmt.Errorf("La fecha y hora de inicio es mayor a la limite")
	}
	// Get module
	module, err := moduleService.GetModuleFromID(idModule)
	if err != nil {
		return err
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
		// Grade
		grade, err := w.verifyGradeWork(idObjModule, idObjGrade)
		if err != nil {
			return err
		}
		if grade.IsAcumulative {
			work.Acumulative = idObjGrade
			work.Grade = grade.Grade.Hex()
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
		if !form.Status {
			return fmt.Errorf("Este formulario está eliminado")
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
		Title:       work.Title,
		Description: work.Description,
		DateStart:   tStart,
		DateLimit:   tLimit,
		Author:      claims.Name,
		IDModule:    idModule,
		Published:   time.Now(),
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
	// Notification
	nats.PublishEncode("notify/classroom", res.NotifyClassroom{
		Title: work.Title,
		Link: fmt.Sprintf(
			"/aula_virtual/clase/%s/trabajos/%s",
			idModule,
			insertedWork.InsertedID.(primitive.ObjectID).Hex(),
		),
		Where: module.Subject.Hex(),
		Room:  module.Section.Hex(),
		Type:  res.WORK,
	})
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
	if err := cursor.Decode(&answerData); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
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
	if err := cursor.Decode(&fUC); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
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

func (w *WorkSerice) updateGrade(
	work *models.Work,
	idObjStudent,
	idObjEvaluator primitive.ObjectID,
	points int,
) error {
	var maxPoints int
	if work.Type == "form" {
		questions, err := w.getQuestionsFromIdForm(work.Form)
		if err != nil {
			return err
		}
		// MAX Points
		var questionsWPoints []models.ItemQuestion
		for _, question := range questions {
			if question.Type != "alternatives" {
				maxPoints += question.Points
				questionsWPoints = append(questionsWPoints, question)
			}
		}
		// Points
		points, _, err = w.getStudentEvaluate(
			questionsWPoints,
			idObjStudent,
			work.ID,
		)
		if err != nil {
			return err
		}
	} else if work.Type == "files" {
		for _, item := range work.Pattern {
			maxPoints += item.Points
		}
	}
	// Update grade
	min, max, err := GetMinNMaxGrade()
	if err != nil {
		return err
	}
	var scale float32 = float32(max-min) / float32(maxPoints)
	grade := w.TransformPointsToGrade(
		scale,
		min,
		points,
	)
	if work.IsQualified {
		// Get grade
		var gradeD *models.GradesProgram
		cursor := gradeProgramModel.GetByID(work.Grade)
		if err := cursor.Decode(&gradeD); err != nil {
			return nil
		}

		// Generate models
		match := bson.D{
			{
				Key:   "module",
				Value: work.Module,
			},
			{
				Key:   "student",
				Value: idObjStudent,
			},
			{
				Key:   "program",
				Value: work.Grade,
			},
		}
		if gradeD.IsAcumulative {
			match = append(match, bson.E{
				Key:   "acumulative",
				Value: work.Acumulative,
			})
		}

		_, err = gradeModel.Use().UpdateOne(
			db.Ctx,
			match,
			bson.D{{
				Key: "$set",
				Value: bson.M{
					"grade":     grade,
					"date":      primitive.NewDateTimeFromTime(time.Now()),
					"evaluator": idObjEvaluator,
				},
			}},
		)
		if err != nil {
			return err
		}
	} else {
		_, err = workGradeModel.Use().UpdateOne(
			db.Ctx,
			bson.D{
				{
					Key:   "module",
					Value: work.Module,
				},
				{
					Key:   "student",
					Value: idObjStudent,
				},
				{
					Key:   "work",
					Value: work.ID,
				},
			},
			bson.D{{
				Key: "$set",
				Value: bson.M{
					"grade":     grade,
					"date":      primitive.NewDateTimeFromTime(time.Now()),
					"evaluator": idObjEvaluator,
				},
			}},
		)
		if err != nil {
			return err
		}
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
	// Get module
	module, err := moduleService.GetModuleFromID(work.Module.Hex())
	if err != nil {
		return err
	}
	// Get grade program
	var gradeProgram models.GradesProgram
	cursor := gradeProgramModel.GetByID(work.Grade)
	if err := cursor.Decode(&gradeProgram); err != nil {
		if err.Error() == db.NO_SINGLE_DOCUMENT {
			return fmt.Errorf("No existe la programación de calificación")
		}
		return err
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
	cursor = formQuestionModel.GetByID(idObjQuestion)
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
	if err := cursor.Decode(&evaluatedAnswer); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
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
	// Grade
	if work.IsRevised {
		err = w.updateGrade(work, idObjStudent, idObjEvaluator, 0)
		if err != nil {
			return err
		}
		// Send notifications
		nats.PublishEncode("notify/classroom", res.NotifyClassroom{
			Title: fmt.Sprintf("Calificación N%d° actualizada", gradeProgram.Number),
			Link: fmt.Sprintf(
				"/aula_virtual/clase/%s/calificaciones",
				work.Module.Hex(),
			),
			Where:  module.Subject.Hex(),
			Room:   module.Section.Hex(),
			Type:   res.GRADE,
			IDUser: idStudent,
		})
	}
	return nil
}

func (w *WorkSerice) UploadEvaluateFiles(
	evalute []forms.EvaluateFilesForm,
	idWork,
	idEvaluator,
	idStudent string,
	reavaluate bool,
) error {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return err
	}
	idObjStudent, err := primitive.ObjectIDFromHex(idStudent)
	if err != nil {
		return err
	}
	idObjEvaluator, err := primitive.ObjectIDFromHex(idEvaluator)
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
	if !reavaluate && work.IsRevised {
		return fmt.Errorf("Ya no se puede actualizar el puntaje del alumno")
	}
	// Get module
	module, err := moduleService.GetModuleFromID(work.Module.Hex())
	if err != nil {
		return err
	}
	// Get grade program
	var gradeProgram models.GradesProgram
	cursor := gradeProgramModel.GetByID(work.Grade)
	if err := cursor.Decode(&gradeProgram); err != nil {
		if err.Error() == db.NO_SINGLE_DOCUMENT {
			return fmt.Errorf("No existe la programación de calificación")
		}
		return err
	}
	// Get evaluate student
	var fUC *models.FileUploadedClassroom
	cursor = fileUCModel.GetOne(bson.D{
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
	if !reavaluate && len(evaluateFiles) > 0 {
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
	} else if reavaluate {
		points := 0
		for _, eva := range evalute {
			points += *eva.Points
		}
		err = w.updateGrade(work, idObjStudent, idObjEvaluator, points)
		if err != nil {
			return err
		}
	}
	// Send notifications
	nats.PublishEncode("notify/classroom", res.NotifyClassroom{
		Title: fmt.Sprintf("Calificación N%d° actualizada", gradeProgram.Number),
		Link: fmt.Sprintf(
			"/aula_virtual/clase/%s/calificaciones",
			work.Module.Hex(),
		),
		Where:  module.Subject.Hex(),
		Room:   module.Section.Hex(),
		Type:   res.GRADE,
		IDUser: idStudent,
	})
	return nil
}

func (w *WorkSerice) TransformPointsToGrade(
	scale float32,
	minGrade,
	points int,
) float64 {
	if points == 0 {
		return float64(minGrade)
	}
	grade := (scale * float32(points)) + float32(minGrade)
	finalGrade := math.Round(float64(grade)*10) / 10
	return finalGrade
}

type StudentGrades struct {
	ID          primitive.ObjectID
	Grade       float64
	ExistsGrade bool
}

func (w *WorkSerice) gradeEvaluatedWork(
	studentsGrade []StudentGrades,
	work *models.Work,
	idObjUser primitive.ObjectID,
	program *models.GradesProgram,
) error {
	type UpdateGrade struct {
		Student primitive.ObjectID
		Grade   float64
	}
	// Generate models
	var modelsGrades []interface{}
	var updates []UpdateGrade
	for _, student := range studentsGrade {
		if !student.ExistsGrade {
			modelGrade := models.NewModelGrade(
				work.Module,
				student.ID,
				work.Acumulative,
				work.Grade,
				idObjUser,
				student.Grade,
				program.IsAcumulative,
			)
			modelsGrades = append(modelsGrades, modelGrade)
		} else {
			updates = append(updates, UpdateGrade{
				Student: student.ID,
				Grade:   student.Grade,
			})
		}
	}
	// Insert grades
	_, err := gradeModel.Use().InsertMany(db.Ctx, modelsGrades)
	if err != nil {
		return err
	}
	// Update grades
	for _, update := range updates {
		filter := bson.D{
			{
				Key:   "module",
				Value: work.Module,
			},
			{
				Key:   "student",
				Value: update.Student,
			},
			{
				Key:   "program",
				Value: program.ID,
			},
		}
		if program.IsAcumulative {
			filter = append(filter, bson.E{
				Key:   "acumulative",
				Value: program.Acumulative,
			})
		}
		_, err = gradeModel.Use().UpdateOne(db.Ctx, filter, bson.D{{
			Key: "$set",
			Value: bson.M{
				"grade": update.Grade,
			},
		}})
		if err != nil {
			return err
		}
	}
	return nil
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
	// Get grade program
	var program *models.GradesProgram
	if work.IsQualified {
		cursor = gradeProgramModel.GetByID(work.Grade)
		if err := cursor.Decode(&program); err != nil {
			return nil
		}
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
		ID          primitive.ObjectID
		Points      int
		ExistsGrade bool
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
			// Get grade if exists
			var grade *models.Grade
			match := bson.D{
				{
					Key:   "module",
					Value: work.Module,
				},
				{
					Key:   "student",
					Value: idObjStudent,
				},
				{
					Key:   "program",
					Value: work.Grade,
				},
			}
			if work.IsQualified && program.IsAcumulative {
				match = append(match, bson.E{
					Key:   "acumulative",
					Value: work.Acumulative,
				})
			}
			cursor = gradeModel.GetOne(match)
			if err := cursor.Decode(&grade); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
				*errRet = err
				return
			}
			// Get form access
			_, err = w.getAccessFromIdStudentNIdWork(
				idObjStudent,
				idObjWork,
			)
			if err != nil {
				if err.Error() != db.NO_SINGLE_DOCUMENT {
					*errRet = err
					return
				}
				lock.Lock()
				studentsWithoutAccess = append(studentsWithoutAccess, idObjStudent)
				studentsPoints = append(studentsPoints, StudentPoints{
					ID:          idObjStudent,
					Points:      0,
					ExistsGrade: grade != nil,
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
				ID:          idObjStudent,
				Points:      points,
				ExistsGrade: grade != nil,
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
			ID:          student.ID,
			Grade:       grade,
			ExistsGrade: student.ExistsGrade,
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
	// Insert and update grades
	if work.IsQualified {
		err = w.gradeEvaluatedWork(
			studentsGrade,
			work,
			idObjUser,
			program,
		)
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
				idObjWork,
				student.Grade,
			)
			modelsGrades = append(modelsGrades, modelWorkGrade)
		}
		_, err := workGradeModel.Use().InsertMany(db.Ctx, modelsGrades)
		if err != nil {
			return err
		}
	}
	// Send notifications
	module, err := moduleService.GetModuleFromID(work.Module.Hex())
	if err != nil {
		return err
	}
	nats.PublishEncode("notify/classroom", res.NotifyClassroom{
		Title: fmt.Sprintf("Trabajo evaluado %v", work.Title),
		Link: fmt.Sprintf(
			"/aula_virtual/clase/%s/trabajos/%s",
			work.Module.Hex(),
			work.ID.Hex(),
		),
		Where: module.Subject.Hex(),
		Room:  module.Section.Hex(),
		Type:  res.GRADE,
	})
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
	// Get module
	module, err := moduleService.GetModuleFromID(work.Module.Hex())
	if err != nil {
		return err
	}
	// Get grade program
	var program *models.GradesProgram
	cursor := gradeProgramModel.GetByID(work.Grade)
	if err := cursor.Decode(&program); err != nil {
		return nil
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
		Student     primitive.ObjectID
		Points      int
		ExistsGrade bool
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
			// Get grade if exists
			var grade *models.Grade
			match := bson.D{
				{
					Key:   "module",
					Value: work.Module,
				},
				{
					Key:   "student",
					Value: idObjStudent,
				},
				{
					Key:   "program",
					Value: work.Grade,
				},
			}
			if program.IsAcumulative {
				match = append(match, bson.E{
					Key:   "acumulative",
					Value: work.Acumulative,
				})
			}
			cursor = gradeModel.GetOne(match)
			if err := cursor.Decode(&grade); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
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
				if err.Error() != db.NO_SINGLE_DOCUMENT {
					lock.Lock()
					studentsPoints = append(studentsPoints, StudentPoints{
						Student:     idObjStudent,
						Points:      0,
						ExistsGrade: grade != nil,
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
				Student:     idObjStudent,
				Points:      points,
				ExistsGrade: grade != nil,
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
		err = w.gradeEvaluatedWork(
			studentsGrade,
			work,
			idObjUser,
			program,
		)
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
				idObjWork,
				student.Grade,
			)
			modelsGrades = append(modelsGrades, modelWorkGrade)
		}
		_, err := workGradeModel.Use().InsertMany(db.Ctx, modelsGrades)
		if err != nil {
			return err
		}
	}
	// Send notifications
	nats.PublishEncode("notify/classroom", res.NotifyClassroom{
		Title: fmt.Sprintf("Trabajo evaluado %v", work.Title),
		Link: fmt.Sprintf(
			"/aula_virtual/clase/%s/trabajos/%s",
			work.Module.Hex(),
			work.ID.Hex(),
		),
		Where: module.Subject.Hex(),
		Room:  module.Section.Hex(),
		Type:  res.GRADE,
	})
	return nil
}

func (w *WorkSerice) UpdateWork(work *forms.UpdateWorkForm, idWork, idUser string) error {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return err
	}
	idObjUser, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return err
	}
	// Get work
	workData, err := w.getWorkFromId(idObjWork)
	if err != nil {
		return err
	}
	if workData.IsRevised {
		return fmt.Errorf("Ya no se puede editar este trabajo")
	}
	// Update work
	update := bson.M{}
	updateEs := make(map[string]interface{})
	unset := bson.M{}
	if work.Title != "" {
		update["title"] = work.Title
		updateEs["title"] = work.Title
	}
	if work.Description != "" {
		update["description"] = work.Description
		updateEs["description"] = work.Description
	}
	if workData.IsQualified && work.Grade != "" {
		idObjGrade, err := primitive.ObjectIDFromHex(work.Grade)
		if err != nil {
			return err
		}
		// Grade
		grade, err := w.verifyGradeWork(workData.Module, idObjGrade)
		if err != nil {
			return err
		}
		if grade.IsAcumulative {
			update["grade"] = grade.Grade
			update["acumulative"] = grade.Acumulative
		} else {
			update["grade"] = grade.Grade
			unset["acumulative"] = ""
		}
	}
	now := time.Now()
	if workData.Type == "files" && now.Before(workData.DateStart.Time()) {
		var pattern []models.WorkPattern

		for _, item := range work.Pattern {
			var itemAdd models.WorkPattern
			if item.ID != "" {
				idObjItem, err := primitive.ObjectIDFromHex(item.ID)
				if err != nil {
					return err
				}
				var find bool
				for _, itemData := range workData.Pattern {
					if itemData.ID == idObjItem {
						find = true
						break
					}
				}
				if !find {
					return fmt.Errorf("No se puede actualizar un item que no está registrado")
				}
				itemAdd.ID = idObjItem
			}
			itemAdd.Title = item.Title
			itemAdd.Description = item.Description
			itemAdd.Points = item.Points
			pattern = append(pattern, itemAdd)
		}
		update["pattern"] = pattern
	} else if workData.Type == "form" && now.Before(workData.DateStart.Time()) {
		if work.Form != "" {
			idObjForm, err := primitive.ObjectIDFromHex(work.Form)
			if err != nil {
				return err
			}
			form, err := formService.GetFormById(idObjForm)
			if err != nil {
				return err
			}
			if !form.Status {
				return fmt.Errorf("Este formulario está eliminado")
			}
			if form.Author != idObjUser {
				return fmt.Errorf("Este formulario no te pertenece")
			}
			if !form.HasPoints && workData.IsQualified {
				return fmt.Errorf("Un trabajo evaluado no puede tener un formulario sin puntaje")
			}
			update["form"] = idObjForm
		}
		if work.FormAccess != "" {
			update["form_access"] = work.FormAccess
		}
		if work.TimeFormAccess != 0 {
			update["time_access"] = work.TimeFormAccess
		}
	}
	// Attached
	var attached []models.Attached
	for _, att := range work.Attached {
		attachedModel := models.Attached{
			ID:   primitive.NewObjectID(),
			Type: att.Type,
		}
		if att.Type == "link" {
			attachedModel.Link = att.Link
			attachedModel.Title = att.Title
		} else {
			idObjFile, err := primitive.ObjectIDFromHex(att.File)
			if err != nil {
				return err
			}
			attachedModel.File = idObjFile
		}
		attached = append(attached, attachedModel)
	}
	update["attached"] = attached
	// Date
	var tStart time.Time
	var tLimit time.Time
	if now.Before(workData.DateStart.Time()) && work.DateStart != "" {
		tStart, err := time.Parse("2006-01-02 15:04", work.DateStart)
		if err != nil {
			return err
		}
		toDateTime := primitive.NewDateTimeFromTime(tStart)
		update["date_start"] = toDateTime
		updateEs["date_start"] = toDateTime
	}
	if work.DateLimit != "" {
		tLimit, err := time.Parse("2006-01-02 15:04", work.DateLimit)
		if err != nil {
			return err
		}
		toDateTime := primitive.NewDateTimeFromTime(tLimit)
		update["date_limit"] = toDateTime
		updateEs["date_limit"] = toDateTime
	}
	if !tStart.IsZero() && !tLimit.IsZero() && tStart.After(tLimit) {
		return fmt.Errorf("La fecha y hora de inicio es mayor a la limite")
	}
	if !tStart.IsZero() && tLimit.IsZero() && tStart.After(workData.DateLimit.Time()) {
		return fmt.Errorf("La fecha y hora de inicio es mayor a la limite registrada")
	}
	if !tLimit.IsZero() && tStart.IsZero() && workData.DateStart.Time().After(tLimit) {
		return fmt.Errorf("La fecha y hora de inicio registrada es mayor a la limite")
	}
	update["date_update"] = primitive.NewDateTimeFromTime(now)
	// Update work
	// Update ES
	data, err := json.Marshal(updateEs)
	if err != nil {
		return err
	}
	bi, err := models.NewBulkWork()
	if err != nil {
		return err
	}
	err = bi.Add(
		context.Background(),
		esutil.BulkIndexerItem{
			Action:     "update",
			DocumentID: idWork,
			Body:       bytes.NewReader([]byte(fmt.Sprintf(`{"doc":%s}`, data))),
		},
	)
	if err != nil {
		return err
	}
	if err := bi.Close(context.Background()); err != nil {
		return err
	}
	// Update DB
	_, err = workModel.Use().UpdateByID(db.Ctx, idObjWork, bson.D{
		{
			Key:   "$set",
			Value: update,
		},
		{
			Key:   "$unset",
			Value: unset,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (w *WorkSerice) DeleteWork(idWork string) error {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return err
	}
	// Get work
	work, err := w.getWorkFromId(idObjWork)
	if err != nil {
		return err
	}
	if work.IsRevised {
		return fmt.Errorf("No se puede eliminar un trabajo calificado")
	}
	// Delete references
	filter := bson.D{{
		Key:   "work",
		Value: idObjWork,
	}}
	if work.Type == "files" {
		var fUCs []models.FileUploadedClassroom
		cursor, err := fileUCModel.GetAll(filter, &options.FindOptions{})
		if err != nil {
			return err
		}
		if err := cursor.All(db.Ctx, &fUCs); err != nil {
			return err
		}
		// Files to delete
		var files []string
		for _, fUC := range fUCs {
			for _, file := range fUC.FilesUploaded {
				files = append(files, file.Hex())
			}
		}
		if len(files) > 0 {
			err = nats.PublishEncode("delete_files", files)
			if err != nil {
				return err
			}
		}
		_, err = fileUCModel.Use().DeleteMany(db.Ctx, filter)
		if err != nil {
			return err
		}
	} else if work.Type == "form" {
		_, err = answerModel.Use().DeleteMany(db.Ctx, filter)
		if err != nil {
			return err
		}
		_, err = evaluatedAnswersModel.Use().DeleteMany(db.Ctx, filter)
		if err != nil {
			return err
		}
		_, err = formAccessModel.Use().DeleteMany(db.Ctx, filter)
		if err != nil {
			return err
		}
	}
	// Delete work ElasticSearch
	bi, err := models.NewBulkWork()
	if err != nil {
		return err
	}
	err = bi.Add(
		context.Background(),
		esutil.BulkIndexerItem{
			Action:     "delete",
			DocumentID: idWork,
		},
	)
	if err != nil {
		return err
	}
	if err := bi.Close(context.Background()); err != nil {
		return err
	}
	// Delete work
	_, err = workModel.Use().DeleteOne(db.Ctx, bson.D{{
		Key:   "_id",
		Value: idObjWork,
	}})
	if err != nil {
		return err
	}
	// Delete notifications
	nats.Publish("delete_notification", []byte(idWork))
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

func (w *WorkSerice) DeleteAttached(idWork, idAttached string) error {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return err
	}
	idObjAttached, err := primitive.ObjectIDFromHex(idAttached)
	if err != nil {
		return err
	}
	// Get work
	work, err := w.getWorkFromId(idObjWork)
	if err != nil {
		return err
	}
	if work.IsRevised {
		return fmt.Errorf("Ya no se puede editar este trabajo")
	}
	// Delete attached
	for _, attached := range work.Attached {
		if attached.ID == idObjAttached {
			_, err = workModel.Use().UpdateByID(
				db.Ctx,
				idObjWork,
				bson.D{{
					Key: "$pull",
					Value: bson.M{
						"attached": bson.M{
							"_id": idObjAttached,
						},
					},
				}},
			)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("No existe este elemento adjunto al trabajo")
}

func (w *WorkSerice) DeleteItemPattern(idWork, idItem string) error {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return err
	}
	idObjItem, err := primitive.ObjectIDFromHex(idItem)
	if err != nil {
		return err
	}
	// Get work
	work, err := w.getWorkFromId(idObjWork)
	if err != nil {
		return err
	}
	if work.Type != "files" {
		return fmt.Errorf("Este no es un trabajo de archivos")
	}
	if work.IsRevised {
		return fmt.Errorf("Este trabajo ya no se puede editar")
	}
	// Delete item
	for _, item := range work.Pattern {
		if item.ID == idObjItem {
			_, err := workModel.Use().UpdateByID(db.Ctx, idObjWork, bson.D{{
				Key: "$pull",
				Value: bson.M{
					"pattern": bson.M{
						"_id": idObjItem,
					},
				},
			}})
			if err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("No existe el item a eliminar")
}

func NewWorksService() *WorkSerice {
	if workService == nil {
		workService = &WorkSerice{}
	}
	return workService
}
