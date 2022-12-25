package repositories

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

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
			Err:        fmt.Errorf("work not found"),
			StatusCode: http.StatusNotFound,
		}
	}
	return &works[0], nil
}

func (w *WorkRepository) GetModulesWorks(
	modulesOr bson.A,
	idObjUser primitive.ObjectID,
) (response []WorkStatus, error *res.ErrorRes) {
	// Recovery if close channel
	defer func() {
		recovery := recover()
		if recovery != nil {
			fmt.Printf("A channel closed")
		}
	}()
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
	var errRes res.ErrorRes
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
				formAccessRepo := NewFormAccessRepository()

				formAccess, errRes := formAccessRepo.GetFormAccess(bson.D{
					{
						Key:   "work",
						Value: work.ID,
					},
					{
						Key:   "student",
						Value: idObjUser,
					},
				})
				if errRes != nil {
					close(c)
					return
				}
				if formAccess != nil {
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
					*errRet = res.ErrorRes{
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
		}(work, i, &wg, &errRes)
	}
	wg.Wait()
	if errRes.Err != nil {
		return nil, &errRes
	}
	// Order by status asc
	sort.Slice(workStatus, func(i, j int) bool {
		return workStatus[i].Status < workStatus[j].Status
	})
	return workStatus, nil
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
