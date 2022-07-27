package services

import (
	"encoding/json"
	"fmt"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

func (module *ModulesService) GetModules(sectionIds []ModuleIDs, userType string) ([]models.ModuleWithLookup, error) {
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
	// Get only courses
	return modulesData, nil
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
