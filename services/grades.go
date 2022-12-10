package services

import (
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const MAX_GRADES = 30

var gradesService *GradesService

type GradesService struct{}

func (g *GradesService) GetGradePrograms(idModule string) ([]models.GradesProgram, *res.ErrorRes) {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get
	var programs []models.GradesProgram
	cursor, err := gradeProgramModel.GetAll(bson.D{
		{
			Key:   "module",
			Value: idObjModule,
		},
	}, &options.FindOptions{})
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if err := cursor.All(db.Ctx, &programs); err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	sort.Slice(programs, func(i, j int) bool {
		return programs[i].Number < programs[j].Number
	})
	return programs, nil
}

func (g *GradesService) getIndexAcumulative(
	idProgram primitive.ObjectID,
	acumulative []models.Acumulative,
) int {
	for i, program := range acumulative {
		if program.ID == idProgram {
			return i
		}
	}
	return -1
}

func (g *GradesService) orderInSliceGradesStudent(
	grades []models.GradeWLookup,
	programs []models.GradesProgram,
) []*OrderedGrade {
	orderedGrades := make([]*OrderedGrade, len(programs))

	for i, program := range programs {
		for _, grade := range grades {
			if program.ID == grade.Program {
				if !program.IsAcumulative {
					orderedGrades[i] = &OrderedGrade{
						ID:            grade.ID.Hex(),
						Grade:         grade.Grade,
						IsAcumulative: false,
						Evaluator:     grade.Evaluator,
						Date:          grade.Date.Time(),
					}
				} else if program.IsAcumulative && orderedGrades[i] == nil {
					// Acumulative
					acumulative := make([]*Acumulative, len(program.Acumulative))
					indexAcumulative := g.getIndexAcumulative(
						grade.Acumulative,
						program.Acumulative,
					)
					percentage := program.Acumulative[indexAcumulative].Percentage
					acumulative[indexAcumulative] = &Acumulative{
						ID:        grade.ID.Hex(),
						Grade:     grade.Grade,
						Evaluator: &grade.Evaluator,
						Date:      grade.Date.Time(),
					}
					// Add to grades
					orderedGrades[i] = &OrderedGrade{
						Grade:         (grade.Grade * float64(percentage)) / 100,
						IsAcumulative: true,
						Acumulative:   acumulative,
					}
				} else {
					// Add to acumulative
					indexAcumulative := g.getIndexAcumulative(
						grade.Acumulative,
						program.Acumulative,
					)
					percentage := program.Acumulative[indexAcumulative].Percentage
					// Grade
					orderedGrades[i].Grade += (grade.Grade * float64(percentage)) / 100
					orderedGrades[i].Acumulative[indexAcumulative] = &Acumulative{
						ID:        grade.ID.Hex(),
						Grade:     grade.Grade,
						Date:      grade.Date.Time(),
						Evaluator: &grade.Evaluator,
					}
				}
			}
		}
	}
	return orderedGrades
}

func (g *GradesService) GetStudentsGrades(idModule string) ([]StudentGrade, *res.ErrorRes) {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get students
	students, err := workService.getStudentsFromIdModule(idModule)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if len(students) == 0 {
		return nil, nil
	}
	// Get grades programs
	programs, errRes := g.GetGradePrograms(idModule)
	if errRes.Err != nil {
		return nil, errRes
	}
	// Get students grades
	studentsGrades := make([]StudentGrade, len(students))
	var wg sync.WaitGroup
	c := make(chan int, 5)

	for i, student := range students {
		wg.Add(1)
		c <- 1

		go func(student Student, i int, errRet *res.ErrorRes, wg *sync.WaitGroup) {
			defer wg.Done()

			idObjStudent, err := primitive.ObjectIDFromHex(student.User.ID)
			if err != nil {
				*errRet = res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusBadRequest,
				}
				close(c)
				return
			}
			// Student
			studentGrade := StudentGrade{
				Student: student.User,
			}
			// Get grades
			var grades []models.GradeWLookup

			match := bson.D{{
				Key: "$match",
				Value: bson.M{
					"module":  idObjModule,
					"student": idObjStudent,
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
							"_id":            1,
							"name":           1,
							"first_lastname": 1,
						},
					}}},
				},
			}}
			project := bson.D{{
				Key: "$set",
				Value: bson.M{
					"evaluator": bson.M{
						"$first": "$evaluator",
					},
				},
			}}
			cursor, err := gradeModel.Aggreagate(mongo.Pipeline{
				match,
				lookup,
				project,
			})
			if err != nil {
				errRet = &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
				close(c)
				return
			}
			if err := cursor.All(db.Ctx, &grades); err != nil {
				errRet = &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
				close(c)
				return
			}
			// Order grades
			orderedGrades := g.orderInSliceGradesStudent(grades, programs)
			studentGrade.Grades = orderedGrades

			studentsGrades[i] = studentGrade
			<-c
		}(student, i, errRes, &wg)
	}
	wg.Wait()
	if errRes.Err != nil {
		return nil, errRes
	}
	return studentsGrades, nil
}

func (g *GradesService) GetStudentGrades(idModule, idStudent string) ([]*OrderedGrade, *res.ErrorRes) {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
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
	// Get grades programs
	programs, errRes := g.GetGradePrograms(idModule)
	if errRes.Err != nil {
		return nil, errRes
	}
	// Get grades
	var grades []models.GradeWLookup

	match := bson.D{{
		Key: "$match",
		Value: bson.M{
			"module":  idObjModule,
			"student": idObjStudent,
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
					"_id":            1,
					"name":           1,
					"first_lastname": 1,
				},
			}}},
		},
	}}
	project := bson.D{{
		Key: "$set",
		Value: bson.M{
			"evaluator": bson.M{
				"$first": "$evaluator",
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
	if err := cursor.All(db.Ctx, &grades); err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Order grades
	orderedGrades := g.orderInSliceGradesStudent(grades, programs)
	return orderedGrades, nil
}

func (g *GradesService) getProgramGradeById(idObjProgram primitive.ObjectID) (*models.GradesProgram, error) {
	var program *models.GradesProgram
	cursor := gradeProgramModel.GetByID(idObjProgram)
	if err := cursor.Decode(&program); err != nil {
		return nil, err
	}
	return program, nil
}

func (g *GradesService) UploadProgram(program *forms.GradeProgramForm, idModule string) (interface{}, *res.ErrorRes) {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}

	// Program does not exist
	var pgData *models.GradesProgram
	cursor := gradeProgramModel.GetOne(bson.D{
		{
			Key:   "module",
			Value: idObjModule,
		},
		{
			Key:   "number",
			Value: program.Number,
		},
	})
	if err := cursor.Decode(&pgData); err != nil && err.Error() != "mongo: no documents in result" {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if pgData != nil {
		return nil, &res.ErrorRes{
			Err:        fmt.Errorf("esta calificación ya está programada"),
			StatusCode: http.StatusConflict,
		}
	}
	// Percentage
	type Percentage struct {
		ID    primitive.ObjectID `bson:"_id"`
		Total float32            `bson:"percentage"`
	}
	var percentage []Percentage
	match := bson.D{
		{
			Key: "$match",
			Value: bson.M{
				"module": idObjModule,
			},
		},
	}
	group := bson.D{
		{
			Key: "$group",
			Value: bson.M{
				"_id": idObjModule,
				"percentage": bson.M{
					"$sum": "$percentage",
				},
			},
		},
	}
	cursorP, err := gradeProgramModel.Aggreagate(mongo.Pipeline{
		match,
		group,
	})
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if err := cursorP.All(db.Ctx, &percentage); err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if len(percentage) > 0 && percentage[0].Total+program.Percentage > 100 {
		return nil, &res.ErrorRes{
			Err: fmt.Errorf(
				"el porcentaje indicado superado el 100 por ciento. Queda %v por ciento libre",
				100-percentage[0].Total,
			),
			StatusCode: http.StatusBadRequest,
		}
	} else if program.Percentage > 100 {
		return nil, &res.ErrorRes{
			Err:        fmt.Errorf("el porcentaje indicado superado el 100 por ciento"),
			StatusCode: http.StatusBadRequest,
		}
	}
	// Validate acumulative
	if *program.IsAcumulative {
		var sum float32
		for _, acumulative := range program.Acumulative {
			sum += acumulative.Percentage
		}
		if sum != 100 {
			return nil, &res.ErrorRes{
				Err: fmt.Errorf(
					"el porcentaje sumatorio de las calificaciones acumulativas debe ser exactamente 100 por cierto",
				),
				StatusCode: http.StatusBadRequest,
			}
		}
	}
	// Insert
	model := models.NewModelGradesProgram(program, idObjModule)
	insertedProgram, err := gradeProgramModel.NewDocument(model)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	return insertedProgram.InsertedID, nil
}

func (g *GradesService) UploadGrade(
	grade *forms.GradeForm,
	idModule,
	idStudent,
	idUser string,
) (interface{}, *res.ErrorRes) {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
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
	idObjProgram, err := primitive.ObjectIDFromHex(grade.Program)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjUser, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get module
	module, err := moduleService.GetModuleFromID(idModule)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Evaluate grade
	min, max, err := GetMinNMaxGrade()
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if *grade.Grade < float64(min) || *grade.Grade > float64(max) {
		return nil, &res.ErrorRes{
			Err:        fmt.Errorf("calificación inválida. Mín: %v. Máx: %v", min, max),
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get program
	program, err := g.getProgramGradeById(idObjProgram)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}

	idAcumulative := primitive.NilObjectID
	if program.IsAcumulative {
		idObjAcumulative, err := primitive.ObjectIDFromHex(grade.Acumulative)
		if err != nil {
			return nil, &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusBadRequest,
			}
		}
		var exists bool
		for _, acumulative := range program.Acumulative {
			if idObjAcumulative == acumulative.ID {
				idAcumulative = idObjAcumulative
				exists = true
			}
		}
		if !exists {
			return nil, &res.ErrorRes{
				Err:        fmt.Errorf("la calificación acumulativa no existe"),
				StatusCode: http.StatusConflict,
			}
		}
	}
	// Get grade
	var gradeData *models.Grade
	filter := bson.D{
		{
			Key:   "module",
			Value: idObjModule,
		},
		{
			Key:   "student",
			Value: idObjStudent,
		},
		{
			Key:   "program",
			Value: program.ID,
		},
	}
	if program.IsAcumulative {
		filter = append(filter, bson.E{
			Key:   "acumulative",
			Value: idAcumulative,
		})
	}
	cursor := gradeModel.GetOne(filter)
	if err := cursor.Decode(&gradeData); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if gradeData != nil {
		return nil, &res.ErrorRes{
			Err:        fmt.Errorf("no se puede agregar una calificación ya subida"),
			StatusCode: http.StatusConflict,
		}
	}
	// Insert grade
	modelGrade := models.NewModelGrade(
		idObjModule,
		idObjStudent,
		idAcumulative,
		idObjProgram,
		idObjUser,
		*grade.Grade,
		program.IsAcumulative,
	)
	inserted, err := gradeModel.NewDocument(modelGrade)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Send notifications
	nats.PublishEncode("notify/classroom", res.NotifyClassroom{
		Title: fmt.Sprintf("Calificación N%d° subida", program.Number),
		Link: fmt.Sprintf(
			"/aula_virtual/clase/%s/calificaciones",
			idModule,
		),
		Where:  module.Subject.Hex(),
		Room:   module.Section.Hex(),
		Type:   res.GRADE,
		IDUser: idStudent,
	})
	return inserted.InsertedID, nil
}

func (g *GradesService) DeleteGradeProgram(idModule, idProgram string) *res.ErrorRes {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjProgram, err := primitive.ObjectIDFromHex(idProgram)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get grade program
	var gradeProgram models.GradesProgram
	cursor := gradeProgramModel.GetByID(idObjProgram)
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
	if idObjModule != gradeProgram.Module {
		return &res.ErrorRes{
			Err:        fmt.Errorf("esta programación de calificación no pertenece al módulo indicado"),
			StatusCode: http.StatusConflict,
		}
	}
	// Get work in use
	var work *models.Work
	cursor = workModel.GetOne(bson.D{{
		Key:   "grade",
		Value: idObjProgram,
	}})
	if err := cursor.Decode(&work); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if work != nil {
		return &res.ErrorRes{
			Err:        fmt.Errorf("esta programación está en uso en el trabajo %s", work.Title),
			StatusCode: http.StatusConflict,
		}
	}
	// Get grade in use
	var allGrades bson.A
	if gradeProgram.IsAcumulative {
		for _, acumulative := range gradeProgram.Acumulative {
			allGrades = append(allGrades, bson.M{
				"program": acumulative.ID,
			})
		}
	}
	allGrades = append(allGrades, bson.M{
		"program": idObjProgram,
	})

	var grade *models.Grade
	cursor = gradeModel.GetOne(bson.D{{
		Key:   "$or",
		Value: allGrades,
	}})
	if err := cursor.Decode(&grade); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if grade != nil {
		return &res.ErrorRes{
			Err:        fmt.Errorf("esta programación está en uso en alguna calificación"),
			StatusCode: http.StatusConflict,
		}
	}
	// Delete grade
	_, err = gradeProgramModel.Use().DeleteOne(db.Ctx, bson.D{{
		Key:   "_id",
		Value: idObjProgram,
	}})
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	return nil
}

func (g *GradesService) UpdateGrade(grade *forms.UpdateGradeForm, idModule, idGrade string) *res.ErrorRes {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjGrade, err := primitive.ObjectIDFromHex(idGrade)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get grade
	var gradeData *models.Grade
	cursor := gradeModel.GetByID(idObjGrade)
	if err := cursor.Decode(&gradeData); err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if gradeData.Module != idObjModule {
		return &res.ErrorRes{
			Err:        fmt.Errorf("esta calificación no pertenece al módulo"),
			StatusCode: http.StatusConflict,
		}
	}
	// Get grade program
	var gradeProgram models.GradesProgram
	cursor = gradeProgramModel.GetByID(gradeData.Program)
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
	// Get module
	module, err := moduleService.GetModuleFromID(idModule)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Min max
	min, max, err := GetMinNMaxGrade()
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if float64(min) > *grade.Grade || float64(max) < *grade.Grade {
		return &res.ErrorRes{
			Err:        fmt.Errorf("calificación inválida. Mín: %v. Máx: %v", min, max),
			StatusCode: http.StatusBadRequest,
		}
	}
	// Update
	_, err = gradeModel.Use().UpdateByID(db.Ctx, idObjGrade, bson.D{{
		Key: "$set",
		Value: bson.M{
			"grade": *grade.Grade,
		},
	}})
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
			idModule,
		),
		Where:  module.Subject.Hex(),
		Room:   module.Section.Hex(),
		Type:   res.GRADE,
		IDUser: gradeData.Student.Hex(),
	})
	return nil
}

func NewGradesService() *GradesService {
	if gradesService == nil {
		gradesService = &GradesService{}
	}
	return gradesService
}
