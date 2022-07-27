package services

import (
	"fmt"
	"sort"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

func (g *GradesService) UploadProgram(program *forms.GradeProgramForm, idModule string) error {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return err
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
		return err
	}
	if pgData != nil {
		return fmt.Errorf("Esta calificación ya está programada")
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
	if err := cursorP.All(db.Ctx, &percentage); err != nil {
		return err
	}
	if len(percentage) > 0 && percentage[0].Total+program.Percentage > 100 {
		return fmt.Errorf(
			"El porcentaje indicado superado el 100 por ciento. Queda %v por ciento libre",
			100-percentage[0].Total,
		)
	} else if program.Percentage > 100 {
		return fmt.Errorf(
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
			return fmt.Errorf(
				"El porcentaje sumatorio de las calificaciones acumulativas debe ser exactamente 100 por cierto",
			)
		}
	}
	// Insert
	model := models.NewModelGradesProgram(program, idObjModule)
	_, err = gradeProgramModel.NewDocument(model)
	if err != nil {
		return err
	}
	return nil
}

func NewGradesService() *GradesService {
	if gradesService == nil {
		gradesService = &GradesService{}
	}
	return gradesService
}
