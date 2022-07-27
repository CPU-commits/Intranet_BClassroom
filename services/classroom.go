package services

import (
	"encoding/json"
	"fmt"

	"github.com/CPU-commits/Intranet_BClassroom/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ModuleIDs struct {
	IDCourse  primitive.ObjectID
	IDSubject primitive.ObjectID
}

// Service module
var moduleService = NewModulesService()

func FindCourses(claims *Claims) ([]ModuleIDs, error) {
	filter := bson.D{
		{
			Key:   "user",
			Value: claims.ID,
		},
	}
	if claims.UserType == models.STUDENT || claims.UserType == models.STUDENT_DIRECTIVE {
		var studentData *models.Student

		cursor := studentModel.GetOne(filter)
		err := cursor.Decode(&studentData)
		if studentData.Course.IsZero() {
			return nil, fmt.Errorf("No estás asignado a ningún curso")
		}

		if err != nil {
			return nil, err
		}
		return []ModuleIDs{{
			IDCourse: studentData.Course,
		}}, nil
	} else {
		var teacherData *models.Teacher

		cursor := teacherModel.GetOne(filter)
		err := cursor.Decode(&teacherData)
		if err != nil {
			return nil, err
		}
		if len(teacherData.Imparted) == 0 {
			return nil, fmt.Errorf("No tienes ningún curso asignado")
		}

		var courses []ModuleIDs
		for _, imparted := range teacherData.Imparted {
			courses = append(courses, ModuleIDs{
				IDCourse:  imparted.Course,
				IDSubject: imparted.Subject,
			})
		}
		return courses, nil
	}
}

func AuthorizedRouteFromIdModule(idModule string, claims *Claims) error {
	// Get courses
	courses, err := FindCourses(claims)
	if err != nil {
		return err
	}
	// Get module
	module, err := moduleService.GetModuleFromID(idModule)
	if err != nil {
		return err
	}
	// Compare
	flag := false
	for _, course := range courses {
		if claims.UserType == models.TEACHER {
			if course.IDSubject.Hex() == module.Subject.Hex() && course.IDCourse.Hex() == module.Section.Hex() {
				flag = true
				break
			}
		} else {
			if course.IDCourse.Hex() == module.Section.Hex() {
				flag = true
				break
			}
		}
	}
	if !flag {
		return fmt.Errorf("No tienes acceso a esta sección")
	}
	return nil
}

func GetMinNMaxGrade() (int, int, error) {
	return 20, 70, nil
}

// Get URL file
func GetAwsTokenFiles(keys []string) ([]string, error) {
	// Request nats
	data, err := json.Marshal(keys)
	if err != nil {
		return nil, err
	}
	msg, err := nats.Request("get_aws_token_access", data)
	if err != nil {
		return nil, err
	}
	var tokenUrls []string
	json.Unmarshal(msg.Data, &tokenUrls)
	return tokenUrls, nil
}
