package repositories

import (
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"go.mongodb.org/mongo-driver/bson"
)

func (w *WorkRepository) getLookupUser() bson.D {
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

func (w *WorkRepository) getLookupGrade() bson.D {
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
