package services

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

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

	var grade float64
	if work.Type != "in-person" {
		var scale float32 = float32(max-min) / float32(maxPoints)
		grade = w.TransformPointsToGrade(
			scale,
			min,
			points,
		)
	} else {
		grade = float64(points) / 1000
	}
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
			Err:        fmt.Errorf("todavía no se pueden evaluar preguntas en este formulario"),
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
				Err:        fmt.Errorf("no existe la programación de calificación"),
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
			Err:        fmt.Errorf("puntaje fuera de rango. Debe ser entre cero y máx %v", question.Points),
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
	if work.IsRevised && work.IsQualified {
		err = w.updateGrade(work, idObjStudent, idObjEvaluator, 0)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
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
			Err:        fmt.Errorf("todavía no se puede evaluar el trabajo"),
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

	if work.IsQualified {
		cursor := gradeProgramModel.GetByID(work.Grade)
		if err := cursor.Decode(&gradeProgram); err != nil {
			if err.Error() == db.NO_SINGLE_DOCUMENT {
				return &res.ErrorRes{
					Err:        fmt.Errorf("no existe la programación de calificación"),
					StatusCode: http.StatusNotFound,
				}
			}
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
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
						Err:        fmt.Errorf("los puntos evaluados superan el máx. del item"),
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
	if work.IsRevised && work.IsQualified {
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

func (w *WorkSerice) UploadEvaluateInperson(
	evalute *forms.EvaluateInperson,
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
	if work.Type != "in-person" {
		return &res.ErrorRes{
			Err:        fmt.Errorf("este trabajo no es de tipo presencial"),
			StatusCode: http.StatusBadRequest,
		}
	}
	now := time.Now()
	if now.Before(work.DateLimit.Time()) {
		return &res.ErrorRes{
			Err:        fmt.Errorf("todavía no se puede evaluar el trabajo"),
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
	var number int
	if work.IsQualified {
		var gradeProgram models.GradesProgram
		cursor := gradeProgramModel.GetByID(work.Grade)
		if err := cursor.Decode(&gradeProgram); err != nil {
			if err.Error() == db.NO_SINGLE_DOCUMENT {
				return &res.ErrorRes{
					Err:        fmt.Errorf("no existe la programación de calificación"),
					StatusCode: http.StatusNotFound,
				}
			}
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}

		number = gradeProgram.Number
	}
	// Check req
	time, err := time.Parse("2006-01-02", evalute.InDate)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	for _, session := range work.Sessions {
		if session.Block.Hex() == evalute.Block {
			exists := false
			for _, date := range session.Dates {
				if date.Time().Equal(time) {
					exists = true
				}
			}
			if !exists {
				return &res.ErrorRes{
					Err:        errors.New("no existe la sesión"),
					StatusCode: http.StatusBadRequest,
				}
			}
		}
	}
	min, max, err := GetMinNMaxGrade()
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if min > int(evalute.Pregrade) && max < int(evalute.Pregrade) {
		return &res.ErrorRes{
			Err:        fmt.Errorf("la calificación debe estar entre %d y %d", min, max),
			StatusCode: http.StatusBadRequest,
		}
	}

	// Upload
	// Exists session
	var session *models.Session
	cursor := sessionModel.GetOne(bson.D{
		{
			Key:   "student",
			Value: idObjStudent,
		},
		{
			Key:   "work",
			Value: idObjWork,
		},
	})
	if err := cursor.Decode(&session); err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
	}
	if session == nil {
		newSession, err := models.NewModelSession(
			evalute,
			idObjStudent,
			idObjWork,
		)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusBadRequest,
			}
		}

		_, err = sessionModel.NewDocument(newSession)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
	} else {
		idObjBlock, err := primitive.ObjectIDFromHex(evalute.Block)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusBadRequest,
			}
		}

		_, err = sessionModel.Use().UpdateByID(db.Ctx, session.ID, bson.D{{
			Key: "$set",
			Value: bson.M{
				"id_date":  primitive.NewDateTimeFromTime(time),
				"block":    idObjBlock,
				"pregrade": evalute.Pregrade,
			},
		}})
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
	}
	if reavaluate {
		err = w.updateGrade(work, idObjStudent, idObjEvaluator, int(evalute.Pregrade*1000))
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
	}
	// Send notifications
	if work.IsRevised && work.IsQualified {
		nats.PublishEncode("notify/classroom", res.NotifyClassroom{
			Title: fmt.Sprintf("Calificación N%d° actualizada", number),
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

// Grade works
func (w *WorkSerice) gradeForm(
	work *models.Work,
	program *models.GradesProgram,
	students []Student,
	idObjUser primitive.ObjectID,
) ([]StudentGrades, *res.ErrorRes) {
	// Get form
	var form *models.Form
	cursor := formModel.GetByID(work.Form)
	if err := cursor.Decode(&form); err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if !form.HasPoints {
		return nil, &res.ErrorRes{
			Err:        fmt.Errorf("este formulario no puede ser calificado, el formulario no tiene puntos"),
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get questions form
	var questionsWPoints []models.ItemQuestion
	questions, err := w.getQuestionsFromIdForm(work.Form)
	if err != nil {
		return nil, &res.ErrorRes{
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
	var lock sync.Mutex

	errRes := utils.Concurrency(
		10,
		len(students),
		func(index int, setError func(errRes *res.ErrorRes)) {
			student := students[index]

			idObjStudent, err := primitive.ObjectIDFromHex(student.User.ID)
			if err != nil {
				setError(&res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusBadRequest,
				})
				return
			}
			// Get grade if exists
			existsGrade, errRes := gradesService.ExistsGrade(idObjStudent, work.ID)
			if errRes != nil {
				setError(errRes)
				return
			}
			// Get form access
			_, err = w.getAccessFromIdStudentNIdWork(
				idObjStudent,
				work.ID,
			)
			if err != nil {
				if !errors.Is(err, mongo.ErrNoDocuments) {
					setError(&res.ErrorRes{
						Err:        err,
						StatusCode: http.StatusServiceUnavailable,
					})
					return
				}
				lock.Lock()
				studentsWithoutAccess = append(studentsWithoutAccess, idObjStudent)
				studentsPoints = append(studentsPoints, StudentPoints{
					ID:          idObjStudent,
					Points:      0,
					ExistsGrade: existsGrade,
				})
				lock.Unlock()
				return
			}
			// Get evaluate
			points, prom, err := w.getStudentEvaluate(
				questionsWPoints,
				idObjStudent,
				work.ID,
			)
			if err != nil {
				setError(&res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				})
				return
			}
			if prom != 100 {
				setError(&res.ErrorRes{
					Err:        errors.New("no todos los alumnos están completamente evaluados"),
					StatusCode: http.StatusBadRequest,
				})
				return
			}
			lock.Lock()
			studentsPoints = append(studentsPoints, StudentPoints{
				ID:          idObjStudent,
				Points:      points,
				ExistsGrade: existsGrade,
			})
			lock.Unlock()
		},
	)
	if errRes != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Get min max grade
	minGrade, maxGrade, err := GetMinNMaxGrade()
	if err != nil {
		return nil, &res.ErrorRes{
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
				work.ID,
			)
			insertStudents = append(insertStudents, modelAccess)
		}
		_, err = formAccessModel.Use().InsertMany(db.Ctx, insertStudents)
		if err != nil {
			return nil, &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
	}
	_, err = formAccessModel.Use().UpdateMany(
		db.Ctx,
		bson.D{{
			Key:   "work",
			Value: work.ID,
		}},
		bson.D{{
			Key: "$set",
			Value: bson.M{
				"status": "revised",
			},
		}},
	)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	return studentsGrade, nil
}

func (w *WorkSerice) gradeFiles(
	work *models.Work,
	program *models.GradesProgram,
	students []Student,
	idObjUser primitive.ObjectID,
) ([]StudentGrades, *res.ErrorRes) {
	// Get points student
	type StudentPoints struct {
		Student     primitive.ObjectID
		Points      int
		ExistsGrade bool
	}
	var lock sync.Mutex
	var studentsPoints []StudentPoints

	errRes := utils.Concurrency(
		10,
		len(students),
		func(index int, setError func(errRes *res.ErrorRes)) {
			student := students[index]

			idObjStudent, err := primitive.ObjectIDFromHex(student.User.ID)
			if err != nil {
				setError(&res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusBadRequest,
				})
				return
			}
			// Get grade if exists
			existsGrade, errRes := gradesService.ExistsGrade(idObjStudent, work.ID)
			if errRes != nil {
				setError(errRes)
				return
			}
			// Get files uploaded W Points
			var fUC *models.FileUploadedClassroom
			cursor := fileUCModel.GetOne(bson.D{
				{
					Key:   "work",
					Value: work.ID,
				},
				{
					Key:   "student",
					Value: idObjStudent,
				},
			})
			if err := cursor.Decode(&fUC); err != nil {
				if !errors.Is(err, mongo.ErrNoDocuments) {
					setError(&res.ErrorRes{
						Err:        err,
						StatusCode: http.StatusServiceUnavailable,
					})
				}
				lock.Lock()
				studentsPoints = append(studentsPoints, StudentPoints{
					Student:     idObjStudent,
					Points:      0,
					ExistsGrade: existsGrade,
				})
				lock.Unlock()
				return
			}
			// Evaluate
			if len(fUC.Evaluate) != len(work.Pattern) {
				setError(&res.ErrorRes{
					Err:        errors.New("no todos los alumnos están completamente evaluados con todos los items"),
					StatusCode: http.StatusBadRequest,
				})
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
				ExistsGrade: existsGrade,
			})
			lock.Unlock()
		},
	)
	if errRes != nil {
		return nil, errRes
	}
	// Get min max grade
	minGrade, maxGrade, err := GetMinNMaxGrade()
	if err != nil {
		return nil, &res.ErrorRes{
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
	_, err = workModel.Use().UpdateByID(db.Ctx, work.ID, bson.D{{
		Key: "$set",
		Value: bson.M{
			"is_revised": true,
		},
	}})
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	return studentsGrade, nil
}

func (w *WorkSerice) gradeInperson(
	work *models.Work,
	program *models.GradesProgram,
	students []Student,
	idObjUser primitive.ObjectID,
) ([]StudentGrades, *res.ErrorRes) {
	// Get grades student
	var studentsGrade []StudentGrades
	var lock sync.Mutex

	errRes := utils.Concurrency(
		10,
		len(students),
		func(index int, setError func(errRes *res.ErrorRes)) {
			student := students[index]

			idObjStudent, err := primitive.ObjectIDFromHex(student.User.ID)
			if err != nil {
				setError(&res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusBadRequest,
				})
				return
			}
			// Get grade if exists
			existsGrade, errRes := gradesService.ExistsGrade(idObjStudent, work.ID)
			if errRes != nil {
				setError(&res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				})
				return
			}
			// Get session
			var session *models.Session

			cursor := sessionModel.GetOne(bson.D{
				{
					Key:   "student",
					Value: idObjStudent,
				},
				{
					Key:   "work",
					Value: work.ID,
				},
			})
			if err := cursor.Decode(&session); err != nil {
				if !errors.Is(err, mongo.ErrNoDocuments) {
					setError(&res.ErrorRes{
						Err:        err,
						StatusCode: http.StatusServiceUnavailable,
					})
				} else {
					setError(&res.ErrorRes{
						Err:        errors.New("no todos los alumnos tienen sesión"),
						StatusCode: http.StatusBadRequest,
					})
				}
				return
			}
			// Push grade
			lock.Lock()
			studentsGrade = append(studentsGrade, StudentGrades{
				ID:          idObjStudent,
				Grade:       session.PreGrade,
				ExistsGrade: existsGrade,
			})
			lock.Unlock()
		},
	)
	if errRes != nil {
		return nil, &res.ErrorRes{
			Err:        errRes.Err,
			StatusCode: errRes.StatusCode,
		}
	}
	return studentsGrade, nil
}

func (w *WorkSerice) GradeWork(
	idWork,
	idUser,
	workType string,
) *res.ErrorRes {
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
	// Check work
	work, err := workRepository.GetWorkFromId(idObjWork)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if work.Type != workType {
		var trans string
		if workType == "in-person" {
			trans = "presencial"
		} else if workType == "files" {
			trans = "archivos"
		} else if workType == "form" {
			trans = "formulario"
		}

		return &res.ErrorRes{
			Err:        fmt.Errorf("el trabajo no es de tipo %s", trans),
			StatusCode: http.StatusBadRequest,
		}
	}
	if time.Now().Before(work.DateLimit.Time()) {
		return &res.ErrorRes{
			Err:        fmt.Errorf("este trabajo todavía no se puede calificar"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	if work.IsRevised {
		return &res.ErrorRes{
			Err:        fmt.Errorf("este trabajo ya está evaluado"),
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

	if work.IsQualified {
		cursor := gradeProgramModel.GetByID(work.Grade)
		if err := cursor.Decode(&program); err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
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
	// Custom worktype
	var errRes *res.ErrorRes
	var studentsGrade []StudentGrades

	if workType == "in-person" {
		studentsGrade, errRes = w.gradeInperson(work, program, students, idObjUser)
	} else if workType == "files" {
		studentsGrade, errRes = w.gradeFiles(work, program, students, idObjUser)
	} else if workType == "form" {
		studentsGrade, errRes = w.gradeForm(work, program, students, idObjUser)
	}
	if errRes != nil {
		return errRes
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
