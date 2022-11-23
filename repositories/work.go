package repositories

import (
	"fmt"
	"net/http"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type WorkRepository struct{}

func (w *WorkRepository) GetWorkFromId(idWork primitive.ObjectID) (*models.Work, error) {
	var work *models.Work
	cursor := workModel.GetByID(idWork)
	if err := cursor.Decode(&work); err != nil {
		return nil, err
	}
	return work, nil
}

func (w *WorkRepository) GetWork(idObjWork primitive.ObjectID) (*models.WorkWLookupNFiles, *res.ErrorRes) {
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
		w.getLookupUser(),
		w.getLookupGrade(),
		setAuthorNGrade,
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
	if len(works) == 0 {
		return nil, &res.ErrorRes{
			Err:        fmt.Errorf("Work not found"),
			StatusCode: http.StatusNotFound,
		}
	}
	return &works[0], nil
}

func (w *WorkRepository) GetWorks(idObjModule primitive.ObjectID) ([]models.WorkWLookup, *res.ErrorRes) {
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
	lookupUser := w.getLookupUser()
	lookupGrade := w.getLookupGrade()
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

func NewWorkRepository() *WorkRepository {
	return &WorkRepository{}
}
