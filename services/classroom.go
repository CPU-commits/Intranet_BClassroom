package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"sync"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/funct"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/stack"
	natsPackage "github.com/nats-io/nats.go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Service module
var moduleService = NewModulesService()

func init() {
	validateDirectivesModule()
	closeGrades()
}

func getParentStudents(idObjUser primitive.ObjectID) ([]primitive.ObjectID, *res.ErrorRes) {
	var parentData *models.Parent

	cursor := parentModel.GetOne(bson.D{{
		Key:   "user",
		Value: idObjUser,
	}})
	err := cursor.Decode(&parentData)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if len(parentData.Students) == 0 {
		return nil, &res.ErrorRes{
			Err:        errors.New("no tienes estudiantes asignados"),
			StatusCode: http.StatusBadRequest,
		}
	}

	return parentData.Students, nil
}

func FindCourses(claims *Claims) ([]ModuleIDs, *res.ErrorRes) {
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
			return nil, &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
		if studentData.Course.IsZero() {
			return nil, &res.ErrorRes{
				Err:        fmt.Errorf("no estás asignado a ningún curso"),
				StatusCode: http.StatusForbidden,
			}
		}
		return []ModuleIDs{{
			IDCourse: studentData.Course,
		}}, nil
	} else if claims.UserType == models.TEACHER {
		var teacherData *models.Teacher

		cursor := teacherModel.GetOne(filter)
		err := cursor.Decode(&teacherData)
		if err != nil {
			return nil, &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
		if len(teacherData.Imparted) == 0 {
			return nil, &res.ErrorRes{
				Err:        fmt.Errorf("no tienes ningún curso asignado"),
				StatusCode: http.StatusForbidden,
			}
		}

		var courses []ModuleIDs
		for _, imparted := range teacherData.Imparted {
			courses = append(courses, ModuleIDs{
				IDCourse:  imparted.Course,
				IDSubject: imparted.Subject,
			})
		}
		return courses, nil
	} else {
		students, err := getParentStudents(claims.IDObj)
		if err != nil {
			return nil, err
		}
		// Get modules students
		var studentsCourses []ModuleIDs

		for _, student := range students {
			courses, err := FindCourses(&Claims{
				ID:       student.Hex(),
				UserType: models.STUDENT,
			})
			if err != nil {
				return nil, err
			}
			for i, course := range courses {
				course.IDUsers = []primitive.ObjectID{
					student,
				}

				courses[i] = course
			}
			studentsCourses = append(studentsCourses, courses...)
		}
		// Transform modules
		var studentsCoursesFilter []ModuleIDs

		for _, studentCourse := range studentsCourses {
			inFilter := funct.Some(studentsCoursesFilter, func(sc ModuleIDs) bool {
				return sc.IDCourse.Hex() == studentCourse.IDCourse.Hex()
			})
			if !inFilter {
				studentsCoursesFilter = append(studentsCoursesFilter, studentCourse)
			} else {
				index := funct.Index(studentsCoursesFilter, func(sc ModuleIDs) bool {
					return sc.IDCourse.Hex() == studentCourse.IDCourse.Hex()
				})
				studentsCoursesFilter[index].IDUsers = append(
					studentsCoursesFilter[index].IDUsers,
					studentCourse.IDUsers...,
				)
			}
		}

		return studentsCoursesFilter, nil
	}
}

func AuthorizedRouteFromIdModule(idModule string, claims *Claims) error {
	// Get courses
	courses, errRes := FindCourses(claims)
	if errRes != nil {
		return errRes.Err
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
		// Try to get of history
		idObjModule, err := primitive.ObjectIDFromHex(idModule)
		if err != nil {
			return err
		}
		idObjStudent, err := primitive.ObjectIDFromHex(claims.ID)
		if err != nil {
			return err
		}

		var moduleHistory *models.ModuleHistory
		cursor := moduleHistoryModel.GetOne(bson.D{
			{
				Key:   "module",
				Value: idObjModule,
			},
			{
				Key: "students",
				Value: bson.M{
					"$in": idObjStudent,
				},
			},
		})
		if cursor.Decode(&moduleHistory); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
			return err
		}
		if moduleHistory == nil {
			return fmt.Errorf("no tienes acceso a esta sección")
		}
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
	var minMax stack.DefaultNatsResponse[map[string]int]
	err = json.Unmarshal(jsonString, &minMax)
	if err != nil {
		return 0, 0, err
	}
	if !minMax.Success {
		return 0, 0, errors.New(minMax.Message)
	}
	return minMax.Data["min"], minMax.Data["max"], nil
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

func getSemester(idSemester string) (*models.Semester, error) {
	data, err := formatRequestToNestjsNats(idSemester)
	if err != nil {
		return nil, err
	}
	var response stack.NatsNestJSRes
	msg, err := nats.Request("get_semester", data)
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

func getLastSemester(idSemester string) (*models.Semester, error) {
	data, err := formatRequestToNestjsNats(idSemester)
	if err != nil {
		return nil, err
	}
	var response stack.NatsNestJSRes
	msg, err := nats.Request("get_last_semester", data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(msg.Data, &response)
	if err != nil {
		return nil, err
	}
	if response.Response == nil {
		return nil, nil
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

func validateDirectivesModule() {
	nats.Queue("validate_directives_module", func(m *natsPackage.Msg) {
		var success bool = true
		var messages []string

		payload, err := nats.DecodeDataNest(m.Data)
		if err != nil {
			return
		}
		// Validate directives
		idModule := payload["module"].(string)
		programs, errRes := gradesService.GetGradePrograms(idModule)
		if errRes != nil {
			return
		}

		if payload["all_grades"] == true {
			studentGrades, err := gradesService.GetStudentsGrades(
				idModule,
				false,
				nil,
			)
			if err != nil {
				return
			}

			var breakGrades bool
			for _, studentGrade := range studentGrades {
				for _, grade := range studentGrade.Grades {
					if grade == nil {
						success = false
						messages = append(messages, "all_grades")
						breakGrades = true
						break
					} else if grade.IsAcumulative {
						for _, acum := range grade.Acumulative {
							if acum == nil {
								success = false
								messages = append(messages, "all_grades")
								breakGrades = true
								break
							}
						}
					}
					if breakGrades {
						break
					}
				}
			}
		}
		if payload["continuous"] == true {
			var after int
			for _, program := range programs {
				if after+1 != program.Number {
					success = false
					messages = append(messages, "continuous")
					break
				}
				after = program.Number
			}
		}

		min_grades := make(map[string]interface{})
		v := reflect.ValueOf(payload["min_grades"])
		if v.Kind() == reflect.Map {
			for _, key := range v.MapKeys() {
				strct := v.MapIndex(key)
				min_grades[key.String()] = strct.Interface()
			}
		} else {
			return
		}
		if min_grades["actived"] == true && float64(len(programs)) < min_grades["min_grade"].(float64) {
			success = false
			messages = append(messages, "min_grades")
		}
		// Respond
		responseData := make(map[string]interface{})
		responseData["success"] = success
		responseData["messages"] = messages
		response, err := json.Marshal(responseData)
		if err != nil {
			return
		}
		m.Respond(response)
	})
}

func closeGrades() {
	nats.Subscribe("close_grades_semester", func(m *natsPackage.Msg) {
		// Recovery if close channel
		defer func() {
			recovery := recover()
			if recovery != nil {
				fmt.Printf("A channel closed")
			}
		}()

		allModules, err := moduleService.GetAllModulesSemester()
		if err != nil {
			return
		}
		min, _, err := GetMinNMaxGrade()
		if err != nil {
			return
		}
		semester, err := getCurrentSemester()
		if err != nil {
			return
		}
		minGrade := float64(min)

		var gradesToRegister []interface{}
		var allStudents []Student
		c := make(chan int, 3)
		var wg sync.WaitGroup
		var lock sync.Mutex
		for _, module := range allModules {
			wg.Add(1)
			c <- 1
			go func(module models.Module, wg *sync.WaitGroup, lock *sync.Mutex, errRet *error) {
				defer wg.Done()

				idModule := module.ID.Hex()
				// Get grades program
				programs, errRes := gradesService.GetGradePrograms(idModule)
				if errRes != nil {
					*errRet = errRes.Err
					close(c)
					return
				}
				// Get module students
				students, err := workService.getStudentsFromIdModule(idModule)
				if err != nil {
					*errRet = err
					close(c)
					return
				}

				for _, student := range students {
					// Add Student to all students
					var inStudents bool
					lock.Lock()
					for _, st := range allStudents {
						if st.User.ID == student.User.ID {
							inStudents = true
							break
						}
					}
					lock.Unlock()
					if !inStudents {
						lock.Lock()
						allStudents = append(allStudents, student)
						lock.Unlock()
					}
					// Register grades
					idObjStudent, _ := primitive.ObjectIDFromHex(student.User.ID)

					grades, errRes := gradesService.GetStudentGrades(idModule, student.User.ID)
					if errRes != nil {
						*errRet = errRes.Err
						close(c)
						return
					}
					// Evaluate grades
					for i, grade := range grades {
						if grade == nil {
							gradeModel := models.NewModelGrade(
								module.ID,
								idObjStudent,
								primitive.NilObjectID,
								programs[i].ID,
								primitive.NilObjectID, // System
								minGrade,
								false,
							)

							lock.Lock()
							gradesToRegister = append(gradesToRegister, gradeModel)
							lock.Unlock()
						} else if grade.IsAcumulative {
							for j, acum := range grade.Acumulative {
								if acum == nil {
									gradeModel := models.NewModelGrade(
										module.ID,
										idObjStudent,
										programs[i].Acumulative[j].ID,
										programs[i].ID,
										primitive.NilObjectID,
										minGrade,
										true,
									)

									lock.Lock()
									gradesToRegister = append(gradesToRegister, gradeModel)
									lock.Unlock()
								}
							}
						}
					}
				}
				<-c
			}(module, &wg, &lock, &err)
		}
		wg.Wait()
		if err != nil {
			return
		}

		if len(gradesToRegister) > 0 {
			_, err = gradeModel.Use().InsertMany(db.Ctx, gradesToRegister)
			if err != nil {
				return
			}
		}
		// Insert all averages
		var averages []interface{}
		cS := make(chan int, 5)

		for _, student := range allStudents {
			wg.Add(1)
			cS <- 1
			go func(student Student, wg *sync.WaitGroup, lock *sync.Mutex, errRet *error) {
				defer wg.Done()

				idObjStudent, err := primitive.ObjectIDFromHex(student.User.ID)
				if err != nil {
					*errRet = err
					close(cS)
					return
				}
				courses, errRes := FindCourses(&Claims{
					ID:       student.User.ID,
					UserType: models.STUDENT,
				})
				if errRes != nil {
					*errRet = errRes.Err
					close(cS)
					return
				}
				modules, errRes := modulesService.GetModules(courses, models.STUDENT, true)
				if errRes != nil {
					*errRet = errRes.Err
					close(cS)
					return
				}

				var average float64
				var haveAverage bool
				for _, module := range modules {
					program, errRes := gradesService.GetGradePrograms(module.ID.Hex())
					if errRes != nil {
						*errRet = errRes.Err
						close(cS)
						return
					}
					if len(program) > 0 {
						haveAverage = true
						grades, errRes := gradesService.GetStudentGrades(module.ID.Hex(), student.User.ID)
						if errRes != nil {
							*errRet = errRes.Err
							close(cS)
							return
						}
						for i, grade := range grades {
							average += grade.Grade * (float64(program[i].Percentage) / 100)
						}
					}
				}
				if haveAverage {
					var existsAverage *models.Average
					cursor := averageModel.GetOne(bson.D{
						{
							Key:   "semester",
							Value: semester.ID,
						},
						{
							Key:   "student",
							Value: idObjStudent,
						},
					})
					if err := cursor.Decode(&existsAverage); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
						*errRet = err
						return
					}
					if existsAverage == nil {
						average = math.Round(average)
						modelAverage := models.NewModelAverage(
							average,
							semester.ID,
							idObjStudent,
						)
						lock.Lock()
						averages = append(averages, modelAverage)
						lock.Unlock()
					}
				}
				<-cS
			}(student, &wg, &lock, &err)
		}
		wg.Wait()
		if err != nil {
			return
		}
		if len(averages) > 0 {
			_, err = averageModel.Use().InsertMany(db.Ctx, averages)
			if err != nil {
				return
			}
		}
		// Respond
		response := map[string]bool{
			"success": true,
		}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return
		}
		m.Respond(jsonResponse)
	})
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
