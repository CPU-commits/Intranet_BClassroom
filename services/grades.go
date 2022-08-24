package services

import (
	"fmt"
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

func (g *GradesService) GetGradePrograms(idModule string) ([]models.GradesProgram, error) {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	if err := cursor.All(db.Ctx, &programs); err != nil {
		return nil, err
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
					acumulative[indexAcumulative] = &Acumulative{
						ID:        grade.ID.Hex(),
						Grade:     grade.Grade,
						Evaluator: &grade.Evaluator,
						Date:      grade.Date.Time(),
					}
					// Add to grades
					orderedGrades[i] = &OrderedGrade{
						Grade:         (grade.Grade * float64(program.Percentage)) / 100,
						IsAcumulative: true,
						Acumulative:   acumulative,
					}
				} else {
					// Grade
					orderedGrades[i].Grade += (grade.Grade * float64(program.Percentage)) / 100
					// Add to acumulative
					indexAcumulative := g.getIndexAcumulative(
						grade.Acumulative,
						program.Acumulative,
					)
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

func (g *GradesService) GetStudentsGrades(idModule string) ([]StudentGrade, error) {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return nil, err
	}
	// Get students
	students, err := workService.getStudentsFromIdModule(idModule)
	if err != nil {
		return nil, err
	}
	// Get grades programs
	programs, err := g.GetGradePrograms(idModule)
	if err != nil {
		return nil, err
	}
	// Get students grades
	studentsGrades := make([]StudentGrade, len(students))
	var wg sync.WaitGroup

	for i, student := range students {
		wg.Add(1)

		go func(student Student, i int, errRet *error, wg *sync.WaitGroup) {
			defer wg.Done()

			idObjStudent, err := primitive.ObjectIDFromHex(student.User.ID)
			if err != nil {
				*errRet = err
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
				*errRet = err
				return
			}
			if err := cursor.All(db.Ctx, &grades); err != nil {
				*errRet = err
				return
			}
			// Order grades
			orderedGrades := g.orderInSliceGradesStudent(grades, programs)
			studentGrade.Grades = orderedGrades

			studentsGrades[i] = studentGrade
		}(student, i, &err, &wg)
	}
	wg.Wait()
	if err != nil {
		return nil, err
	}
	return studentsGrades, nil
}

func (g *GradesService) GetStudentGrades(idModule, idStudent string) ([]*OrderedGrade, error) {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return nil, err
	}
	idObjStudent, err := primitive.ObjectIDFromHex(idStudent)
	if err != nil {
		return nil, err
	}
	// Get grades programs
	programs, err := g.GetGradePrograms(idModule)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	if err := cursor.All(db.Ctx, &grades); err != nil {
		return nil, err
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

func (g *GradesService) UploadProgram(program *forms.GradeProgramForm, idModule string) (interface{}, error) {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	if pgData != nil {
		return nil, fmt.Errorf("Esta calificación ya está programada")
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
		return nil, err
	}
	if err := cursorP.All(db.Ctx, &percentage); err != nil {
		return nil, err
	}
	if len(percentage) > 0 && percentage[0].Total+program.Percentage > 100 {
		return nil, fmt.Errorf(
			"El porcentaje indicado superado el 100 por ciento. Queda %v por ciento libre",
			100-percentage[0].Total,
		)
	} else if program.Percentage > 100 {
		return nil, fmt.Errorf(
			"El porcentaje indicado superado el 100 por ciento.",
		)
	}
	// Validate acumulative
	if *program.IsAcumulative {
		var sum float32
		for _, acumulative := range program.Acumulative {
			sum += acumulative.Percentage
		}
		if sum != 100 {
			return nil, fmt.Errorf(
				"El porcentaje sumatorio de las calificaciones acumulativas debe ser exactamente 100 por cierto",
			)
		}
	}
	// Insert
	model := models.NewModelGradesProgram(program, idObjModule)
	insertedProgram, err := gradeProgramModel.NewDocument(model)
	if err != nil {
		return nil, err
	}
	return insertedProgram.InsertedID, nil
}

func (g *GradesService) UploadGrade(
	grade *forms.GradeForm,
	idModule,
	idStudent,
	idUser string,
) (interface{}, error) {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return nil, err
	}
	idObjStudent, err := primitive.ObjectIDFromHex(idStudent)
	if err != nil {
		return nil, err
	}
	idObjProgram, err := primitive.ObjectIDFromHex(grade.Program)
	if err != nil {
		return nil, err
	}
	idObjUser, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return nil, err
	}
	// Get module
	module, err := moduleService.GetModuleFromID(idModule)
	if err != nil {
		return nil, err
	}
	// Evaluate grade
	min, max, err := GetMinNMaxGrade()
	if err != nil {
		return nil, err
	}
	if *grade.Grade < float64(min) || *grade.Grade > float64(max) {
		return nil, fmt.Errorf("Calificación inválida. Mín: %v. Máx: %v", min, max)
	}
	// Get program
	program, err := g.getProgramGradeById(idObjProgram)
	if err != nil {
		return nil, err
	}

	idAcumulative := primitive.NilObjectID
	if program.IsAcumulative {
		idObjAcumulative, err := primitive.ObjectIDFromHex(grade.Acumulative)
		if err != nil {
			return nil, err
		}
		var exists bool
		for _, acumulative := range program.Acumulative {
			if idObjAcumulative == acumulative.ID {
				idAcumulative = idObjAcumulative
				exists = true
			}
		}
		if !exists {
			return nil, fmt.Errorf("La calificación acumulativa no existe")
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
		return nil, err
	}
	if gradeData != nil {
		return nil, fmt.Errorf("No se puede agregar una calificación ya subida")
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
		return nil, err
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

func (g *GradesService) DeleteGradeProgram(idModule, idProgram string) error {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return err
	}
	idObjProgram, err := primitive.ObjectIDFromHex(idProgram)
	if err != nil {
		return err
	}
	// Get grade program
	var gradeProgram models.GradesProgram
	cursor := gradeProgramModel.GetByID(idObjProgram)
	if err := cursor.Decode(&gradeProgram); err != nil {
		if err.Error() == db.NO_SINGLE_DOCUMENT {
			return fmt.Errorf("No existe la programación de calificación")
		}
		return err
	}
	if idObjModule != gradeProgram.Module {
		return fmt.Errorf("Esta programación de calificación no pertenece al módulo indicado")
	}
	// Get work in use
	var work *models.Work
	cursor = workModel.GetOne(bson.D{{
		Key:   "grade",
		Value: idObjProgram,
	}})
	if err := cursor.Decode(&work); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
		return err
	}
	if work != nil {
		return fmt.Errorf("Esta programación está en uso en el trabajo %s", work.Title)
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
		return err
	}
	if grade != nil {
		return fmt.Errorf("Esta programación está en uso en alguna calificación")
	}
	// Delete grade
	_, err = gradeProgramModel.Use().DeleteOne(db.Ctx, bson.D{{
		Key:   "_id",
		Value: idObjProgram,
	}})
	if err != nil {
		return err
	}
	return nil
}

func (g *GradesService) UpdateGrade(grade *forms.UpdateGradeForm, idModule, idGrade string) error {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return err
	}
	idObjGrade, err := primitive.ObjectIDFromHex(idGrade)
	if err != nil {
		return err
	}
	// Get grade
	var gradeData *models.Grade
	cursor := gradeModel.GetByID(idObjGrade)
	if err := cursor.Decode(&gradeData); err != nil {
		return err
	}
	if gradeData.Module != idObjModule {
		return fmt.Errorf("Esta calificación no pertenece al módulo")
	}
	// Get grade program
	var gradeProgram models.GradesProgram
	cursor = gradeProgramModel.GetByID(gradeData.Program)
	if err := cursor.Decode(&gradeProgram); err != nil {
		if err.Error() == db.NO_SINGLE_DOCUMENT {
			return fmt.Errorf("No existe la programación de calificación")
		}
		return err
	}
	// Get module
	module, err := moduleService.GetModuleFromID(idModule)
	if err != nil {
		return err
	}
	// Min max
	min, max, err := GetMinNMaxGrade()
	if err != nil {
		return err
	}
	if float64(min) > *grade.Grade || float64(max) < *grade.Grade {
		return fmt.Errorf("Calificación inválida. Mín: %v. Máx: %v", min, max)
	}
	// Update
	_, err = gradeModel.Use().UpdateByID(db.Ctx, idObjGrade, bson.D{{
		Key: "$set",
		Value: bson.M{
			"grade": *grade.Grade,
		},
	}})
	if err != nil {
		return err
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
