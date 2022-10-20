package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	natsPackage "github.com/nats-io/nats.go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var modulesService *ModulesService

type ModulesService struct{}

func getMatch(sectionIds []ModuleIDs, userType string) bson.D {
	var sectiondIdsString bson.A
	for _, moduleIDs := range sectionIds {
		courseFilter := bson.M{
			"section": moduleIDs.IDCourse.Hex(),
		}
		if userType == models.TEACHER {
			sectiondIdsString = append(sectiondIdsString, bson.M{
				"$and": bson.A{
					bson.M{
						"subject": moduleIDs.IDSubject.Hex(),
					},
					courseFilter,
				},
			})
		} else {
			sectiondIdsString = append(sectiondIdsString, courseFilter)
		}
	}
	return bson.D{
		{
			Key: "$match",
			Value: bson.M{
				"$and": bson.A{
					bson.M{
						"status": false,
					},
					bson.M{
						"$or": sectiondIdsString,
					},
				},
			},
		},
	}
}

func getAddFields() bson.D {
	return bson.D{
		{
			Key: "$addFields",
			Value: bson.M{
				"section": bson.M{
					"$toObjectId": "$section",
				},
				"subject": bson.M{
					"$toObjectId": "$subject",
				},
				"semester": bson.M{
					"$toObjectId": "$semester",
				},
			},
		},
	}
}

func getLookupSection() bson.D {
	return bson.D{
		{
			Key: "$lookup",
			Value: bson.M{
				"from":         models.SECTION_COLLECTION,
				"localField":   "section",
				"foreignField": "_id",
				"as":           "section",
				"pipeline": bson.A{
					bson.M{
						"$addFields": bson.M{
							"course": bson.M{
								"$toObjectId": "$course",
							},
							"file": bson.M{
								"$toObjectId": "$file",
							},
						},
					},
					bson.M{
						"$lookup": bson.M{
							"from":         models.COURSE_COLLECTION,
							"localField":   "course",
							"foreignField": "_id",
							"as":           "course",
							"pipeline": bson.A{
								bson.M{
									"$project": bson.M{
										"course": 1,
									},
								},
							},
						},
					},
					bson.M{
						"$lookup": bson.M{
							"from":         models.FILES_COLLECTION,
							"localField":   "file",
							"foreignField": "_id",
							"as":           "file",
							"pipeline": bson.A{
								bson.M{
									"$project": bson.M{
										"key": 1,
									},
								},
							},
						},
					},
					bson.M{
						"$project": bson.M{
							"section": 1,
							"file": bson.M{
								"$arrayElemAt": bson.A{
									"$file", 0,
								},
							},
							"course": bson.M{
								"$arrayElemAt": bson.A{
									"$course", 0,
								},
							},
						},
					},
				},
			},
		},
	}
}

func getLookupSubject() bson.D {
	return bson.D{
		{
			Key: "$lookup",
			Value: bson.M{
				"from":         models.SUBJECT_COLLECTION,
				"localField":   "subject",
				"foreignField": "_id",
				"as":           "subject",
				"pipeline": bson.A{
					bson.M{
						"$project": bson.M{
							"subject": 1,
						},
					},
				},
			},
		},
	}
}

func getLookupSemester() bson.D {
	return bson.D{
		{
			Key: "$lookup",
			Value: bson.M{
				"from":         models.SEMESTER_COLLECTION,
				"localField":   "semester",
				"foreignField": "_id",
				"as":           "semester",
				"pipeline": bson.A{
					bson.M{
						"$project": bson.M{
							"year":     1,
							"semester": 1,
						},
					},
				},
			},
		},
	}
}

func getProject() bson.D {
	return bson.D{
		{
			Key: "$project",
			Value: bson.M{
				"section": bson.M{
					"$arrayElemAt": bson.A{
						"$section", 0,
					},
				},
				"subject": bson.M{
					"$arrayElemAt": bson.A{
						"$subject", 0,
					},
				},
				"semester": bson.M{
					"$arrayElemAt": bson.A{
						"$semester", 0,
					},
				},
				"sub_sections": 1,
			},
		},
	}
}

func (module *ModulesService) GetCourses() {
	nats.Subscribe("get_courses", func(m *natsPackage.Msg) {
		payload, err := nats.DecodeDataNest(m.Data)
		if err != nil {
			return
		}
		courses, err := FindCourses(&Claims{
			ID:       payload["_id"].(string),
			UserType: payload["user_type"].(string),
		})
		if err != nil {
			return
		}

		coursesJson, err := json.Marshal(courses)
		if err != nil {
			return
		}
		m.RespondMsg(&natsPackage.Msg{
			Data:    coursesJson,
			Reply:   m.Reply,
			Subject: m.Subject,
		})
	})
}

func (module *ModulesService) GetModuleFromID(idModule string) (*models.Module, error) {
	objId, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return nil, err
	}

	var moduleData *models.Module
	cursor := moduleModel.GetByID(objId)
	err = cursor.Decode(&moduleData)
	if err != nil {
		return nil, err
	}
	return moduleData, nil
}

func (module *ModulesService) GetModule(moduleId string) (*models.ModuleWithLookup, error) {
	objId, err := primitive.ObjectIDFromHex(moduleId)
	if err != nil {
		return nil, err
	}

	var moduleData []*models.ModuleWithLookup
	match := bson.D{
		{
			Key: "$match",
			Value: bson.M{
				"_id": objId,
			},
		},
	}
	cursor, err := moduleModel.Use().Aggregate(db.Ctx, mongo.Pipeline{
		match,
		getAddFields(),
		getLookupSection(),
		getLookupSubject(),
		getLookupSemester(),
		getProject(),
	})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(db.Ctx, &moduleData); err != nil {
		return nil, err
	}
	return moduleData[0], nil
}

func (module *ModulesService) GetModules(sectionIds []ModuleIDs, userType string, simple bool) ([]models.ModuleWithLookup, error) {
	// Section IDs must be > 0
	if len(sectionIds) == 0 {
		return nil, nil
	}

	var modulesData []models.ModuleWithLookup

	cursor, err := moduleModel.Aggreagate(mongo.Pipeline{
		getMatch(sectionIds, userType),
		getAddFields(),
		getLookupSection(),
		getLookupSubject(),
		getLookupSemester(),
		getProject(),
	})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(db.Ctx, &modulesData); err != nil {
		return nil, err
	}

	if len(modulesData) == 0 {
		return modulesData, nil
	}
	if !simple {
		// Get aws keys
		var images []string
		for i := 0; i < len(modulesData); i++ {
			images = append(images, modulesData[i].Section.File.Key)
		}
		data, err := json.Marshal(images)
		if err != nil {
			return nil, err
		}
		msg, err := nats.Request("get_aws_token_access", data)
		if err != nil {
			return nil, err
		}

		var imagesURLs []string
		json.Unmarshal(msg.Data, &imagesURLs)
		// Add image URLs to modules
		for i := 0; i < len(modulesData); i++ {
			modulesData[i].Section.File.Url = imagesURLs[i]
		}
		// Get next works
		var wg sync.WaitGroup

		for i, module := range modulesData {
			wg.Add(1)

			go func(module models.ModuleWithLookup, i int, wg *sync.WaitGroup, errRet *error) {
				defer wg.Done()

				var works []models.Work

				match := bson.D{{
					Key: "$match",
					Value: bson.M{
						"module":     module.ID,
						"is_revised": false,
						"date_limit": bson.M{
							"$gte": primitive.NewDateTimeFromTime(time.Now()),
						},
					},
				}}
				limit := bson.D{{
					Key:   "$limit",
					Value: 3,
				}}
				sort := bson.D{{
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
						"date_limit":   1,
					},
				}}
				cursor, err := workModel.Aggreagate(mongo.Pipeline{
					match,
					limit,
					sort,
					project,
				})
				if err != nil {
					*errRet = err
					return
				}
				if err := cursor.All(db.Ctx, &works); err != nil {
					*errRet = err
					return
				}
				modulesData[i].Works = works
			}(module, i, &wg, &err)
		}
		wg.Wait()
		if err != nil {
			return nil, err
		}
	}
	// Get only courses
	return modulesData, nil
}

func (module *ModulesService) GetModulesHistory(
	idUser string,
	limit, // If is zero, not limit
	skip int,
	total bool,
	simple bool,
	idSemester string,
) ([]models.ModuleWithLookup, int, error) {
	var totalModules int

	idObjStudent, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return nil, totalModules, err
	}
	// Find modules
	var modules []models.ModuleHistory

	var match bson.D
	if idSemester == "" {
		match = bson.D{{
			Key: "$match",
			Value: bson.M{
				"students": bson.M{
					"$in": bson.A{idObjStudent},
				},
			},
		}}
	} else {
		idObjSemester, err := primitive.ObjectIDFromHex(idSemester)
		if err != nil {
			return nil, 0, err
		}
		match = bson.D{{
			Key: "$match",
			Value: bson.M{
				"students": bson.M{
					"$in": bson.A{idObjStudent},
				},
				"semester": idObjSemester,
			},
		}}
	}
	project := bson.D{{
		Key: "$project",
		Value: bson.M{
			"module": 1,
		},
	}}
	sort := bson.D{{
		Key: "$sort",
		Value: bson.M{
			"date": -1,
		},
	}}
	skipPl := bson.D{{
		Key:   "$skip",
		Value: skip,
	}}
	limitPl := bson.D{{
		Key:   "$limit",
		Value: limit,
	}}

	pipeline := mongo.Pipeline{
		match,
		project,
		sort,
		skipPl,
	}
	if limit != 0 {
		pipeline = append(pipeline, limitPl)
	}
	cursor, err := moduleHistoryModel.Aggreagate(pipeline)
	if err != nil {
		return nil, totalModules, err
	}
	err = cursor.All(db.Ctx, &modules)
	if err != nil {
		return nil, totalModules, err
	}
	// Get modules data
	modulesId := bson.A{}
	for _, module := range modules {
		modulesId = append(modulesId, module.Module)
	}

	var modulesData []models.ModuleWithLookup

	cursor, err = moduleModel.Aggreagate(mongo.Pipeline{
		bson.D{{
			Key: "$match",
			Value: bson.M{
				"_id": bson.M{
					"$in": modulesId,
				},
			},
		}},
		getAddFields(),
		getLookupSection(),
		getLookupSubject(),
		getLookupSemester(),
		getProject(),
	})
	if err != nil {
		return nil, totalModules, err
	}
	if err := cursor.All(db.Ctx, &modulesData); err != nil {
		return nil, totalModules, err
	}
	// Get aws keys
	if !simple {
		var images []string
		for i := 0; i < len(modulesData); i++ {
			images = append(images, modulesData[i].Section.File.Key)
		}
		data, err := json.Marshal(images)
		if err != nil {
			return nil, totalModules, err
		}
		msg, err := nats.Request("get_aws_token_access", data)
		if err != nil {
			return nil, totalModules, err
		}

		var imagesURLs []string
		json.Unmarshal(msg.Data, &imagesURLs)
		// Add image URLs to modules
		for i := 0; i < len(modulesData); i++ {
			modulesData[i].Section.File.Url = imagesURLs[i]
		}
		// Get total of modules
		if total {
			totalOfDocuments, err := moduleHistoryModel.Use().CountDocuments(db.Ctx, bson.D{{
				Key: "students",
				Value: bson.M{
					"$in": bson.A{idObjStudent},
				},
			}})
			if err != nil {
				return nil, totalModules, err
			}
			totalModules = int(totalOfDocuments)
		}
	}
	return modulesData, totalModules, nil
}

func (module *ModulesService) GetAllModulesSemester() ([]models.Module, error) {
	var modules []models.Module

	cursor, err := moduleModel.GetAll(bson.D{{
		Key:   "status",
		Value: false,
	}}, &options.FindOptions{})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(db.Ctx, &modules); err != nil {
		return nil, err
	}
	return modules, nil
}

func (module *ModulesService) NewSubSection(
	subSectionData *forms.SubSectionData,
	idSection string,
) (interface{}, error) {
	sectionId := primitive.NewObjectID()
	subSection := &models.SubSection{
		ID:   sectionId,
		Name: subSectionData.SubSection,
	}
	objId, err := primitive.ObjectIDFromHex(idSection)
	if err != nil {
		return nil, err
	}
	_, err = moduleModel.Use().UpdateByID(db.Ctx, objId, bson.M{
		"$push": bson.M{
			"sub_sections": subSection,
		},
	})
	if err != nil {
		return nil, err
	}
	return sectionId.Hex(), nil
}

func (module *ModulesService) Search(idModule, search string) (interface{}, error) {
	simpleQuery := fmt.Sprintf(
		`"bool": {"must": { "simple_query_string": { "query": "%s*", "analyzer": "standard" } },`,
		search,
	)
	simpleQuery += fmt.Sprintf(`"filter": { "term": { "id_module": "%s" } } }`, idModule)

	query := db.ConstructQuery(simpleQuery)
	var mapRes map[string]interface{}
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}
	es, err := db.NewConnectionEs()
	if err != nil {
		return nil, err
	}
	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex(models.PUBLICATIONS_INDEX, models.WORKS_INDEX),
		es.Search.WithBody(query),
		es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&mapRes); err != nil {
		return nil, err
	}
	return mapRes["hits"], nil
}

func (module *ModulesService) DownloadModuleFile(
	idModule,
	idFile string,
	claims *Claims,
) ([]string, error) {
	idFileObj, err := primitive.ObjectIDFromHex(idFile)
	if err != nil {
		return nil, err
	}
	idUserObj, err := primitive.ObjectIDFromHex(claims.ID)
	if err != nil {
		return nil, err
	}
	// Get file data
	file, err := fileModel.GetFileByID(idFileObj)
	if err != nil {
		return nil, err
	}
	if file.ID == idUserObj {
		tokenUrls, err := GetAwsTokenFiles([]string{file.Key})
		if err != nil {
			return nil, err
		}
		return tokenUrls, err
	}
	// If is a Student
	if file.Permissions == "private" {
		return nil, fmt.Errorf("No tienes acceso a este archivo")
	}
	moduleData, err := module.GetModuleFromID(idModule)
	if err != nil {
		return nil, err
	}
	courses, err := FindCourses(claims)
	if err != nil {
		return nil, err
	}

	for _, course := range courses {
		if course.IDCourse.Hex() == moduleData.Section.Hex() {
			tokenUrls, err := GetAwsTokenFiles([]string{file.Key})
			if err != nil {
				return nil, err
			}
			return tokenUrls, err
		}
	}
	return nil, fmt.Errorf("No tienes acceso a este archivo")
}

func NewModulesService() *ModulesService {
	if modulesService == nil {
		modulesService = &ModulesService{}
	}
	return modulesService
}
