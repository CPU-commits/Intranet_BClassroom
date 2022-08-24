package services

import (
	"encoding/json"
	"fmt"

	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/stack"
	"go.mongodb.org/mongo-driver/bson"
)

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
		if err != nil {
			return nil, err
		}
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
	data, err := formatRequestToNestjsNats("")
	if err != nil {
		return 0, 0, err
	}
	var response stack.NatsNestJSRes
	msg, err := nats.Request("get_min_max_grades", data)
	if err != nil {
		return 0, 0, err
	}

	err = json.Unmarshal(msg.Data, &response)
	if err != nil {
		return 0, 0, err
	}
	jsonString, err := json.Marshal(response.Response)
	if err != nil {
		return 0, 0, err
	}
	var minMax map[string]int
	err = json.Unmarshal(jsonString, &minMax)
	if err != nil {
		return 0, 0, err
	}
	return minMax["min"], minMax["max"], nil
}

func getCurrentSemester() (*models.Semester, error) {
	data, err := formatRequestToNestjsNats("")
	if err != nil {
		return nil, err
	}
	var response stack.NatsNestJSRes
	msg, err := nats.Request("get_valid_semester", data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(msg.Data, &response)
	if err != nil {
		return nil, err
	}

	jsonString, err := json.Marshal(response.Response)
	if err != nil {
		return nil, err
	}
	var semester models.Semester
	err = json.Unmarshal(jsonString, &semester)
	if err != nil {
		return nil, err
	}
	return &semester, nil
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
