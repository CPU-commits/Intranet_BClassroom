package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
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

func (w *WorkSerice) GetModulesWorks(claims Claims) ([]WorkStatus, *res.ErrorRes) {
	// Recovery if close channel
	defer func() {
		recovery := recover()
		if recovery != nil {
			fmt.Printf("A channel closed")
		}
	}()

	idObjUser, err := primitive.ObjectIDFromHex(claims.ID)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}

	// Get modules
	courses, errRes := FindCourses(&claims)
	if errRes != nil {
		return nil, errRes
	}
	modules, errRes := moduleService.GetModules(courses, claims.UserType, true)
	if errRes != nil {
		return nil, errRes
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
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if err := cursor.All(db.Ctx, &works); err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Get status
	workStatus := make([]WorkStatus, len(works))
	var wg sync.WaitGroup
	c := make(chan (int), 5)

	for i, work := range works {
		wg.Add(1)
		c <- 1

		go func(work models.Work, i int, wg *sync.WaitGroup, errRet *res.ErrorRes) {
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
					*errRet = res.ErrorRes{
						Err:        err,
						StatusCode: http.StatusServiceUnavailable,
					}
					close(c)
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
					errRet = &res.ErrorRes{
						Err:        err,
						StatusCode: http.StatusServiceUnavailable,
					}
					close(c)
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
			<-c
		}(work, i, &wg, errRes)
	}
	wg.Wait()
	if errRes != nil {
		return nil, errRes
	}
	// Order by status asc
	sort.Slice(workStatus, func(i, j int) bool {
		return workStatus[i].Status < workStatus[j].Status
	})
	return workStatus, nil
}

func (w *WorkSerice) GetWorks(idModule string) ([]models.WorkWLookup, *res.ErrorRes) {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get
	works, errRes := workRepository.GetWorks(idObjModule)
	if errRes != nil {
		return nil, errRes
	}
	return works, nil
}

func (w *WorkSerice) GetWork(idWork string, claims *Claims) (map[string]interface{}, *res.ErrorRes) {
	idObjUser, err := primitive.ObjectIDFromHex(claims.ID)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get work
	work, errRes := workRepository.GetWork(idObjWork)
	if errRes != nil {
		return nil, errRes
	}
	if time.Now().Before(work.DateStart.Time()) && (claims.UserType == models.STUDENT || claims.UserType == models.STUDENT_DIRECTIVE) {
		return nil, &res.ErrorRes{
			Err:        fmt.Errorf("no se puede acceder a este trabajo todav??a"),
			StatusCode: http.StatusUnauthorized,
		}
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
				return nil, &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
			}
			if err := cursor.All(db.Ctx, &grade); err != nil {
				return nil, &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
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
				return nil, &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
			}
			if err := cursor.All(db.Ctx, &grade); err != nil {
				return nil, &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
			}
			response["grade"] = grade[0]
		}
	}
	// Get form
	if work.Type == "form" {
		form, err := formService.GetFormById(work.Form)
		if err != nil {
			return nil, &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
		response["form_has_points"] = form.HasPoints
	}
	// Student
	if claims.UserType == models.STUDENT || claims.UserType == models.STUDENT_DIRECTIVE {
		if work.Type == "form" {
			// Student access
			idObjStudent, err := primitive.ObjectIDFromHex(claims.ID)
			if err != nil {
				return nil, &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
			}
			formAccess, err := w.getAccessFromIdStudentNIdWork(
				idObjStudent,
				idObjWork,
			)
			if err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
				return nil, &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
			}
			response["form_access"] = formAccess
		} else if work.Type == "files" {
			// Get files uploaded
			fUC, err := w.getFilesUploadedStudent(idObjUser, idObjWork)
			if err != nil {
				return nil, &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
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

func (w *WorkSerice) getStudentEvaluate(
	questions []models.ItemQuestion,
	idStudent,
	idWork primitive.ObjectID,
) (int, int, error) {
	// Recovery if close channel
	defer func() {
		recovery := recover()
		if recovery != nil {
			fmt.Printf("A channel closed")
		}
	}()

	var err error
	var wg sync.WaitGroup
	var lock sync.Mutex
	c := make(chan (int), 5)
	totalPoints := 0
	evaluatedSum := 0
	for _, question := range questions {
		wg.Add(1)
		c <- 1
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
					close(c)
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
					close(c)
					return
				}
				if evaluateAnswer != nil {
					lock.Lock()
					totalPoints += evaluateAnswer.Points
					evaluatedSum += 1
					lock.Unlock()
				}
			}
			<-c
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

func (w *WorkSerice) GetStudentsStatus(idModule, idWork string) ([]Student, int, *res.ErrorRes) {
	// Recovery if close channel
	defer func() {
		recovery := recover()
		if recovery != nil {
			fmt.Printf("A channel closed")
		}
	}()

	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return nil, -1, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get students
	students, err := w.getStudentsFromIdModule(idModule)
	if err != nil {
		return nil, -1, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Get work
	work, err := workRepository.GetWorkFromId(idObjWork)
	if err != nil {
		return nil, -1, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if len(students) == 0 {
		return nil, -1, &res.ErrorRes{
			Err:        fmt.Errorf("ning??n estudiante pertenece a este trabajo"),
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get access of students
	var wg sync.WaitGroup
	c := make(chan (int), 5)
	var questionsWPoints []models.ItemQuestion
	if work.Type == "form" {
		questions, err := w.getQuestionsFromIdForm(work.Form)
		if err != nil {
			return nil, -1, &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
		for _, question := range questions {
			if question.Type != "alternatives" {
				questionsWPoints = append(questionsWPoints, question)
			}
		}
	}
	for i, student := range students {
		wg.Add(1)
		c <- 1
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
			<-c
		}(student, i, &wg, &err)
	}
	wg.Wait()
	if err != nil {
		return nil, -1, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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

func (w *WorkSerice) DownloadFilesWorkStudent(idWork, idStudent string, writter io.Writer) (*zip.Writer, *res.ErrorRes) {
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
	idObjStudent, err := primitive.ObjectIDFromHex(idStudent)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get files
	fUC, err := w.getFilesUploadedStudent(idObjStudent, idObjWork)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if len(fUC) == 0 {
		return nil, &res.ErrorRes{
			Err:        fmt.Errorf("no se pueden descargar archivos si no hay archivos subidos"),
			StatusCode: http.StatusBadRequest,
		}
	}
	// Download files AWS
	type File struct {
		file io.ReadCloser
		name string
	}
	files := make([]File, len(fUC[0].FilesUploaded))
	var wg sync.WaitGroup
	c := make(chan (int), 5)
	for i, file := range fUC[0].FilesUploaded {
		wg.Add(1)
		c <- 1
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
			<-c
		}(file, i, &wg, &err)
	}
	wg.Wait()
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Create zip archive
	zipWritter := zip.NewWriter(writter)
	for _, file := range files {
		zipFile, err := zipWritter.Create(file.name)
		if err != nil {
			return nil, &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusInternalServerError,
			}
		}
		body, err := io.ReadAll(file.file)
		if err != nil {
			return nil, &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusInternalServerError,
			}
		}
		_, err = zipFile.Write(body)
		if err != nil {
			return nil, &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusInternalServerError,
			}
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
				return nil, fmt.Errorf("no existe la calificaci??n indicada")
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
	if !gradeRet.IsAcumulative {
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
	} else {
		cursor = workModel.GetOne(bson.D{
			{
				Key:   "module",
				Value: idObjModule,
			},
			{
				Key:   "grade",
				Value: gradeRet.Grade,
			},
			{
				Key:   "acumulative",
				Value: gradeRet.Acumulative,
			},
		})
	}

	if err := cursor.Decode(&work); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
		return nil, err
	}
	if work != nil {
		return nil, fmt.Errorf("esta calificaci??n est?? registrada ya a un trabajo")
	}
	return &gradeRet, nil
}

func (w *WorkSerice) UploadWork(
	work *forms.WorkForm,
	idModule string,
	claims *Claims,
) *res.ErrorRes {
	idObjUser, err := primitive.ObjectIDFromHex(claims.ID)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Date
	tStart, err := time.Parse("2006-01-02 15:04", work.DateStart)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	tLimit, err := time.Parse("2006-01-02 15:04", work.DateLimit)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	if tStart.After(tLimit) {
		return &res.ErrorRes{
			Err:        fmt.Errorf("la fecha y hora de inicio es mayor a la limite"),
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get module
	module, err := moduleService.GetModuleFromID(idModule)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Grade
	if *work.IsQualified {
		idObjGrade, err := primitive.ObjectIDFromHex(work.Grade)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusBadRequest,
			}
		}
		idObjModule, err := primitive.ObjectIDFromHex(idModule)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusBadRequest,
			}
		}
		// Grade
		grade, err := w.verifyGradeWork(idObjModule, idObjGrade)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusBadRequest,
			}
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
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusBadRequest,
			}
		}
		var form *models.Form
		cursor := formModel.GetByID(idObjForm)
		if err := cursor.Decode(&form); err != nil {
			return &res.ErrorRes{
				Err:        fmt.Errorf("no existe el formulario indicado"),
				StatusCode: http.StatusBadRequest,
			}
		}
		if !form.Status {
			return &res.ErrorRes{
				Err:        fmt.Errorf("este formulario est?? eliminado"),
				StatusCode: http.StatusBadRequest,
			}
		}
		if form.Author != idObjUser {
			return &res.ErrorRes{
				Err:        fmt.Errorf("no tienes acceso a este formulario"),
				StatusCode: http.StatusBadRequest,
			}
		}
		if *work.IsQualified && !form.HasPoints {
			return &res.ErrorRes{
				Err:        fmt.Errorf("este formulario no tiene puntaje. Escoga uno con puntaje"),
				StatusCode: http.StatusBadRequest,
			}
		}
	}
	// Insert
	modelWork, err := models.NewModelWork(work, tStart, tLimit, idObjModule, idObjUser)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	insertedWork, err := workModel.NewDocument(modelWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusInternalServerError,
		}
	}
	// Add item to the BulkIndexer
	oid, _ := insertedWork.InsertedID.(primitive.ObjectID)
	bi, err := models.NewBulkWork()
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if err := bi.Close(context.Background()); err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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
		return fmt.Errorf("la pregunta no pertenece al trabajo indicado")
	}
	// Get question
	var question *models.ItemQuestion
	cursor := formQuestionModel.GetByID(idObjQuestion)
	if err := cursor.Decode(&question); err != nil {
		return err
	}
	lenAnswers := len(question.Answers)

	if question.Type != "written" && answer.Answer != nil {
		if lenAnswers <= *answer.Answer || lenAnswers < 0 {
			return fmt.Errorf("indique una respuesta v??lida")
		}
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
			return fmt.Errorf("la respuesta no existe para ser eliminada")
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
			return fmt.Errorf("no se puede insertar una respuesta de alternativa a una pregunta de escritura")
		}
		if question.Type != "written" && answer.Answer == nil {
			return fmt.Errorf("no se puede insertar una respuesta escrita a una pregunta de alternativas")
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

func (w *WorkSerice) SaveAnswer(answer *forms.AnswerForm, idWork, idQuestion, idStudent string) *res.ErrorRes {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjQuestion, err := primitive.ObjectIDFromHex(idQuestion)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjStudent, err := primitive.ObjectIDFromHex(idStudent)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get work
	work, err := workRepository.GetWorkFromId(idObjWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	if work.Type != "form" {
		return &res.ErrorRes{
			Err:        fmt.Errorf("el trabajo no es de tipo formulario"),
			StatusCode: http.StatusBadRequest,
		}
	}
	if time.Now().After(work.DateLimit.Time()) {
		return &res.ErrorRes{
			Err:        fmt.Errorf("ya no se puede acceder al formulario"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	// Get access
	formAcess, err := w.getAccessFromIdStudentNIdWork(
		idObjStudent,
		idObjWork,
	)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if formAcess.Status != "opened" {
		return &res.ErrorRes{
			Err:        fmt.Errorf("ya no puedes acceder al formulario"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	limitDate := formAcess.Date.Time().Add(time.Duration(work.TimeFormAccess * int(time.Second)))
	if work.FormAccess == "wtime" && time.Now().After(limitDate) {
		return &res.ErrorRes{
			Err:        fmt.Errorf("ya no puedes acceder al formulario"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	w.saveAnswer(answer, idObjWork, idObjQuestion, idObjStudent)
	return nil
}

func (w *WorkSerice) UploadFiles(files []*multipart.FileHeader, idWork, idUser string) *res.ErrorRes {
	// Recovery if close channel
	defer func() {
		recovery := recover()
		if recovery != nil {
			fmt.Printf("A channel closed")
		}
	}()

	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjUser, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	if len(files) > 3 {
		return &res.ErrorRes{
			Err:        fmt.Errorf("solo se puede subir hasta 3 archivos por trabajo"),
			StatusCode: http.StatusRequestEntityTooLarge,
		}
	}
	// Get work
	work, err := workRepository.GetWorkFromId(idObjWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	now := time.Now()
	if now.Before(work.DateStart.Time()) {
		return &res.ErrorRes{
			Err:        fmt.Errorf("todav??a no se puede acceder a este trabajo"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	if now.After(work.DateLimit.Time().Add(7*24*time.Hour)) || work.IsRevised {
		return &res.ErrorRes{
			Err:        fmt.Errorf("ya no se pueden subir archivos a este trabajo"),
			StatusCode: http.StatusUnauthorized,
		}
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
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if fUC != nil && len(fUC.FilesUploaded)+len(files) > 3 {
		return &res.ErrorRes{
			Err:        fmt.Errorf("solo se puede subir hasta 3 archivos por trabajo"),
			StatusCode: http.StatusRequestEntityTooLarge,
		}
	}
	// UploadFiles
	filesIds := make([]primitive.ObjectID, len(files))
	var wg sync.WaitGroup
	c := make(chan (int), 5)

	type FileNats struct {
		Location string `json:"location"`
		Filename string `json:"filename"`
		Mimetype string `json:"mime-type"`
		Key      string `json:"key"`
	}
	for i, file := range files {
		wg.Add(1)
		c <- 1
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
			<-c
		}(*file, i, &wg, &err)
	}
	wg.Wait()
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if fUC == nil {
		modelFileUC := models.NewModelFileUC(
			idObjWork,
			idObjUser,
			filesIds,
		)
		_, err = fileUCModel.NewDocument(modelFileUC)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
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
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
	}
	return nil
}

func (w *WorkSerice) FinishForm(answers *forms.AnswersForm, idWork, idStudent string) *res.ErrorRes {
	// Recovery if close channel
	defer func() {
		recovery := recover()
		if recovery != nil {
			fmt.Printf("A channel closed")
		}
	}()

	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjStudent, err := primitive.ObjectIDFromHex(idStudent)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get work
	work, err := workRepository.GetWorkFromId(idObjWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	now := time.Now()
	if work.DateLimit.Time().Add(time.Minute * 5).Before(now) {
		return &res.ErrorRes{
			Err:        fmt.Errorf("ya no se pueden modificar las respuestas de este formulario"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	// Save answers
	var wg sync.WaitGroup
	c := make(chan (int), 5)
	for _, answer := range answers.Answers {
		idObjQuestion, err := primitive.ObjectIDFromHex(answer.Question)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusBadRequest,
			}
		}
		if answer.Answer != nil && answer.Response != "" {
			wg.Add(1)
			c <- 1
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
				<-c
			}(answer, idObjWork, idObjStudent, idObjQuestion, &wg, &err)
		}
	}
	wg.Wait()
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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
) *res.ErrorRes {
	idObjEvaluator, err := primitive.ObjectIDFromHex(idEvaluator)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjQuestion, err := primitive.ObjectIDFromHex(idQuestion)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjStudent, err := primitive.ObjectIDFromHex(idStudent)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get work
	work, err := workRepository.GetWorkFromId(idObjWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if time.Now().Before(work.DateLimit.Time()) {
		return &res.ErrorRes{
			Err:        fmt.Errorf("todav??a no se pueden evaluar preguntas en este formulario"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	// Get module
	module, err := moduleService.GetModuleFromID(work.Module.Hex())
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Get grade program
	var gradeProgram models.GradesProgram
	cursor := gradeProgramModel.GetByID(work.Grade)
	if err := cursor.Decode(&gradeProgram); err != nil {
		if err.Error() == db.NO_SINGLE_DOCUMENT {
			return &res.ErrorRes{
				Err:        fmt.Errorf("no existe la programaci??n de calificaci??n"),
				StatusCode: http.StatusNotFound,
			}
		}
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Get form
	form, err := formService.GetFormById(work.Form)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if !form.HasPoints {
		return &res.ErrorRes{
			Err:        fmt.Errorf("no se puede evaluar un formulario sin puntos"),
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get question
	var question *models.ItemQuestion
	cursor = formQuestionModel.GetByID(idObjQuestion)
	if err := cursor.Decode(&question); err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if points < 0 || points > question.Points {
		return &res.ErrorRes{
			Err:        fmt.Errorf("puntaje fuera de rango. Debe ser entre cero y m??x %v", question.Points),
			StatusCode: http.StatusBadRequest,
		}
	}
	// Form Access
	_, err = w.getAccessFromIdStudentNIdWork(
		idObjStudent,
		idObjWork,
	)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusUnauthorized,
		}
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
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
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
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
	}
	// Grade
	if work.IsRevised {
		err = w.updateGrade(work, idObjStudent, idObjEvaluator, 0)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
		// Send notifications
		nats.PublishEncode("notify/classroom", res.NotifyClassroom{
			Title: fmt.Sprintf("Calificaci??n N%d?? actualizada", gradeProgram.Number),
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
) *res.ErrorRes {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjStudent, err := primitive.ObjectIDFromHex(idStudent)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjEvaluator, err := primitive.ObjectIDFromHex(idEvaluator)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get work
	work, err := workRepository.GetWorkFromId(idObjWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if work.Type != "files" {
		return &res.ErrorRes{
			Err:        fmt.Errorf("este trabajo no es de tipo archivos"),
			StatusCode: http.StatusBadRequest,
		}
	}
	now := time.Now()
	if now.Before(work.DateLimit.Time()) {
		return &res.ErrorRes{
			Err:        fmt.Errorf("todav??a no se puede evaluar el trabajo"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	if !reavaluate && work.IsRevised {
		return &res.ErrorRes{
			Err:        fmt.Errorf("ya no se puede actualizar el puntaje del alumno"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	// Get module
	module, err := moduleService.GetModuleFromID(work.Module.Hex())
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Get grade program
	var gradeProgram models.GradesProgram
	cursor := gradeProgramModel.GetByID(work.Grade)
	if err := cursor.Decode(&gradeProgram); err != nil {
		if err.Error() == db.NO_SINGLE_DOCUMENT {
			return &res.ErrorRes{
				Err:        fmt.Errorf("no existe la programaci??n de calificaci??n"),
				StatusCode: http.StatusNotFound,
			}
		}
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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
		return &res.ErrorRes{
			Err:        fmt.Errorf("no se encontraron archivos subidos por parte del alumno"),
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Build model
	var evaluateFiles []interface{}
	for _, ev := range evalute {
		idObjPattern, err := primitive.ObjectIDFromHex(ev.Pattern)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusBadRequest,
			}
		}
		exists := false
		for _, item := range work.Pattern {
			if item.ID == idObjPattern {
				if item.Points < *ev.Points {
					return &res.ErrorRes{
						Err:        fmt.Errorf("los puntos evaluados superan el m??x. del item"),
						StatusCode: http.StatusBadRequest,
					}
				}
				exists = true
				break
			}
		}
		if !exists {
			return &res.ErrorRes{
				Err:        fmt.Errorf("no existe el item #%s en este trabajo", ev.Pattern),
				StatusCode: http.StatusNotFound,
			}
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
				return &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
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
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
	} else if reavaluate {
		points := 0
		for _, eva := range evalute {
			points += *eva.Points
		}
		err = w.updateGrade(work, idObjStudent, idObjEvaluator, points)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
	}
	// Send notifications
	nats.PublishEncode("notify/classroom", res.NotifyClassroom{
		Title: fmt.Sprintf("Calificaci??n N%d?? actualizada", gradeProgram.Number),
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
	if len(modelsGrades) > 0 {
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
	}
	return nil
}

func (w *WorkSerice) GradeForm(idWork, idUser string) *res.ErrorRes {
	// Recovery if close channel
	defer func() {
		recovery := recover()
		if recovery != nil {
			fmt.Printf("A channel closed")
		}
	}()

	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjUser, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get work
	work, err := workRepository.GetWorkFromId(idObjWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if work.Type != "form" {
		return &res.ErrorRes{
			Err:        fmt.Errorf("el trabajo no es de tipo formulario"),
			StatusCode: http.StatusBadRequest,
		}
	}
	if time.Now().Before(work.DateLimit.Time()) {
		return &res.ErrorRes{
			Err:        fmt.Errorf("este formulario todav??a no se puede calificar"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	if work.IsRevised {
		return &res.ErrorRes{
			Err:        fmt.Errorf("este trabajo ya est?? evaluado"),
			StatusCode: http.StatusForbidden,
		}
	}
	// Get form
	var form *models.Form
	cursor := formModel.GetByID(work.Form)
	if err := cursor.Decode(&form); err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if !form.HasPoints {
		return &res.ErrorRes{
			Err:        fmt.Errorf("este formulario no puede ser calificado, el formulario no tiene puntos"),
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get student
	students, err := w.getStudentsFromIdModule(work.Module.Hex())
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if len(students) == 0 {
		return &res.ErrorRes{
			Err:        fmt.Errorf("no existen alumnos a evaluar en este trabajo"),
			StatusCode: http.StatusBadRequest,
		}
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
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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
	c := make(chan (int), 5)
	for _, student := range students {
		wg.Add(1)
		c <- 1
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
				*errRet = fmt.Errorf("no todos los alumnos est??n completamente evaluados")
				return
			}
			lock.Lock()
			studentsPoints = append(studentsPoints, StudentPoints{
				ID:          idObjStudent,
				Points:      points,
				ExistsGrade: grade != nil,
			})
			lock.Unlock()
			<-c
		}(student, &wg, &lock, &err)
	}
	wg.Wait()
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Get min max grade
	minGrade, maxGrade, err := GetMinNMaxGrade()
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
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
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Update work status
	_, err = workModel.Use().UpdateByID(db.Ctx, idObjWork, bson.D{{
		Key: "$set",
		Value: bson.M{
			"is_revised": true,
		},
	}})
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
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
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
	}
	// Send notifications
	module, err := moduleService.GetModuleFromID(work.Module.Hex())
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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

func (w *WorkSerice) GradeFiles(idWork, idUser string) *res.ErrorRes {
	// Recovery if close channel
	defer func() {
		recovery := recover()
		if recovery != nil {
			fmt.Printf("A channel closed")
		}
	}()

	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjUser, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get work
	work, err := workRepository.GetWorkFromId(idObjWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if work.Type != "files" {
		return &res.ErrorRes{
			Err:        fmt.Errorf("el trabajo no es de tipo archivos"),
			StatusCode: http.StatusBadRequest,
		}
	}
	if time.Now().Before(work.DateLimit.Time()) {
		return &res.ErrorRes{
			Err:        fmt.Errorf("este trabajo todav??a no se puede calificar"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	if work.IsRevised {
		return &res.ErrorRes{
			Err:        fmt.Errorf("este trabajo ya est?? evaluado"),
			StatusCode: http.StatusForbidden,
		}
	}
	// Get module
	module, err := moduleService.GetModuleFromID(work.Module.Hex())
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if len(students) == 0 {
		return &res.ErrorRes{
			Err:        fmt.Errorf("no existen alumnos a evaluar en este trabajo"),
			StatusCode: http.StatusBadRequest,
		}
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
	c := make(chan (int), 1)
	for _, student := range students {
		wg.Add(1)
		c <- 1
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
				*errRet = fmt.Errorf("no todos los alumnos est??n completamente evaluados con todos los items")
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
			<-c
		}(student, &wg, &lock, &err)
	}
	wg.Wait()
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Get min max grade
	minGrade, maxGrade, err := GetMinNMaxGrade()
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
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
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
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

func (w *WorkSerice) UpdateWork(work *forms.UpdateWorkForm, idWork, idUser string) *res.ErrorRes {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjUser, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get work
	workData, err := workRepository.GetWorkFromId(idObjWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if workData.IsRevised {
		return &res.ErrorRes{
			Err:        fmt.Errorf("ya no se puede editar este trabajo"),
			StatusCode: http.StatusUnauthorized,
		}
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
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusBadRequest,
			}
		}
		if workData.Grade != idObjGrade && workData.Acumulative != idObjGrade {
			// Grade
			grade, err := w.verifyGradeWork(workData.Module, idObjGrade)
			if err != nil {
				return &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
			}
			if grade.IsAcumulative {
				update["grade"] = grade.Grade
				update["acumulative"] = grade.Acumulative
			} else {
				update["grade"] = grade.Grade
				unset["acumulative"] = ""
			}
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
					return &res.ErrorRes{
						Err:        err,
						StatusCode: http.StatusBadRequest,
					}
				}
				var find bool
				for _, itemData := range workData.Pattern {
					if itemData.ID == idObjItem {
						find = true
						break
					}
				}
				if !find {
					return &res.ErrorRes{
						Err:        fmt.Errorf("no se puede actualizar un item que no est?? registrado"),
						StatusCode: http.StatusNotFound,
					}
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
				return &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusBadRequest,
				}
			}
			form, err := formService.GetFormById(idObjForm)
			if err != nil {
				return &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
			}
			if !form.Status {
				return &res.ErrorRes{
					Err:        fmt.Errorf("este formulario est?? eliminado"),
					StatusCode: http.StatusBadRequest,
				}
			}
			if form.Author != idObjUser {
				return &res.ErrorRes{
					Err:        fmt.Errorf("este formulario no te pertenece"),
					StatusCode: http.StatusUnauthorized,
				}
			}
			if !form.HasPoints && workData.IsQualified {
				return &res.ErrorRes{
					Err:        fmt.Errorf("un trabajo evaluado no puede tener un formulario sin puntaje"),
					StatusCode: http.StatusBadRequest,
				}
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
				return &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusBadRequest,
				}
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
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusBadRequest,
			}
		}
		toDateTime := primitive.NewDateTimeFromTime(tStart)
		update["date_start"] = toDateTime
		updateEs["date_start"] = toDateTime
	}
	if work.DateLimit != "" {
		tLimit, err := time.Parse("2006-01-02 15:04", work.DateLimit)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusBadRequest,
			}
		}
		toDateTime := primitive.NewDateTimeFromTime(tLimit)
		update["date_limit"] = toDateTime
		updateEs["date_limit"] = toDateTime
	}
	if !tStart.IsZero() && !tLimit.IsZero() && tStart.After(tLimit) {
		return &res.ErrorRes{
			Err:        fmt.Errorf("la fecha y hora de inicio es mayor a la limite"),
			StatusCode: http.StatusBadRequest,
		}
	}
	if !tStart.IsZero() && tLimit.IsZero() && tStart.After(workData.DateLimit.Time()) {
		return &res.ErrorRes{
			Err:        fmt.Errorf("la fecha y hora de inicio es mayor a la limite registrada"),
			StatusCode: http.StatusBadRequest,
		}
	}
	if !tLimit.IsZero() && tStart.IsZero() && workData.DateStart.Time().After(tLimit) {
		return &res.ErrorRes{
			Err:        fmt.Errorf("la fecha y hora de inicio registrada es mayor a la limite"),
			StatusCode: http.StatusBadRequest,
		}
	}
	update["date_update"] = primitive.NewDateTimeFromTime(now)
	// Update work
	// Update ES
	data, err := json.Marshal(updateEs)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusInternalServerError,
		}
	}
	bi, err := models.NewBulkWork()
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if err := bi.Close(context.Background()); err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
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
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	return nil
}

func (w *WorkSerice) DeleteWork(idWork string) *res.ErrorRes {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get work
	work, err := workRepository.GetWorkFromId(idObjWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if work.IsRevised {
		return &res.ErrorRes{
			Err:        fmt.Errorf("no se puede eliminar un trabajo calificado"),
			StatusCode: http.StatusBadRequest,
		}
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
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
		if err := cursor.All(db.Ctx, &fUCs); err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
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
				return &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
			}
		}
		_, err = fileUCModel.Use().DeleteMany(db.Ctx, filter)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
	} else if work.Type == "form" {
		_, err = answerModel.Use().DeleteMany(db.Ctx, filter)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
		_, err = evaluatedAnswersModel.Use().DeleteMany(db.Ctx, filter)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
		_, err = formAccessModel.Use().DeleteMany(db.Ctx, filter)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
	}
	// Delete work ElasticSearch
	bi, err := models.NewBulkWork()
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	err = bi.Add(
		context.Background(),
		esutil.BulkIndexerItem{
			Action:     "delete",
			DocumentID: idWork,
		},
	)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if err := bi.Close(context.Background()); err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Delete work
	_, err = workModel.Use().DeleteOne(db.Ctx, bson.D{{
		Key:   "_id",
		Value: idObjWork,
	}})
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Delete notifications
	nats.Publish("delete_notification", []byte(idWork))
	return nil
}

func (w *WorkSerice) DeleteFileClassroom(idWork, idFile, idUser string) *res.ErrorRes {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjFile, err := primitive.ObjectIDFromHex(idFile)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjUser, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
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
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	updated := false
	for _, file := range fUC.FilesUploaded {
		if file == idObjFile {
			// Delete file AWS
			msg, err := nats.Request("get_key_from_id_file", []byte(idFile))
			if err != nil {
				return &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
			}

			keyFile := string(msg.Data[:])
			err = aws.DeleteFile(keyFile)
			if err != nil {
				return &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
			}
			// Delete file classroom
			if len(fUC.FilesUploaded) == 1 {
				_, err = fileUCModel.Use().DeleteOne(db.Ctx, bson.D{{
					Key:   "_id",
					Value: fUC.ID,
				}})
				if err != nil {
					return &res.ErrorRes{
						Err:        err,
						StatusCode: http.StatusServiceUnavailable,
					}
				}
			} else {
				_, err = fileUCModel.Use().UpdateByID(db.Ctx, fUC.ID, bson.D{{
					Key: "$pull",
					Value: bson.M{
						"files_uploaded": idObjFile,
					},
				}})
				if err != nil {
					return &res.ErrorRes{
						Err:        err,
						StatusCode: http.StatusServiceUnavailable,
					}
				}
			}
			updated = true
			break
		}
	}
	if !updated {
		return &res.ErrorRes{
			Err:        fmt.Errorf("no se encontr?? el archivo a eliminar en este trabajo"),
			StatusCode: http.StatusNotFound,
		}
	}
	return nil
}

func (w *WorkSerice) DeleteAttached(idWork, idAttached string) *res.ErrorRes {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjAttached, err := primitive.ObjectIDFromHex(idAttached)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get work
	work, err := workRepository.GetWorkFromId(idObjWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if work.IsRevised {
		return &res.ErrorRes{
			Err:        fmt.Errorf("ya no se puede editar este trabajo"),
			StatusCode: http.StatusUnauthorized,
		}
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
				return &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
			}
			return nil
		}
	}
	return &res.ErrorRes{
		Err:        fmt.Errorf("no existe este elemento adjunto al trabajo"),
		StatusCode: http.StatusNotFound,
	}
}

func (w *WorkSerice) DeleteItemPattern(idWork, idItem string) *res.ErrorRes {
	idObjWork, err := primitive.ObjectIDFromHex(idWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjItem, err := primitive.ObjectIDFromHex(idItem)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get work
	work, err := workRepository.GetWorkFromId(idObjWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if work.Type != "files" {
		return &res.ErrorRes{
			Err:        fmt.Errorf("este no es un trabajo de archivos"),
			StatusCode: http.StatusBadRequest,
		}
	}
	if work.IsRevised {
		return &res.ErrorRes{
			Err:        fmt.Errorf("este trabajo ya no se puede editar"),
			StatusCode: http.StatusUnauthorized,
		}
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
				return &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
			}
			return nil
		}
	}
	return &res.ErrorRes{
		Err:        fmt.Errorf("no existe el item a eliminar"),
		StatusCode: http.StatusNotFound,
	}
}

func NewWorksService() *WorkSerice {
	if workService == nil {
		workService = &WorkSerice{}
	}
	return workService
}
