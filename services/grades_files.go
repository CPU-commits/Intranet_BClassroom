package services

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/stack"
	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (g *GradesService) ExportGrades(idModule string, w io.Writer) (*excelize.File, *res.ErrorRes) {
	_, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get grades
	data, errRes := g.GetStudentsGrades(idModule)
	if errRes != nil {
		return nil, errRes
	}
	// Get programs
	programs, errRes := g.GetGradePrograms(idModule)
	if errRes.Err != nil {
		return nil, errRes
	}
	// Init file
	file := excelize.NewFile()
	sheetName := "Calificaciones"
	file.SetSheetName("Sheet1", sheetName)
	// Set columns
	file.SetCellValue(sheetName, "A1", "Estudiante")
	for i, program := range programs {
		value := fmt.Sprintf("%v (%v%v)", program.Number, program.Percentage, string('%'))
		column := fmt.Sprintf("%v1", string(rune('A'+i+1)))
		file.SetCellValue(sheetName, column, value)
	}
	// Set values
	for i, student := range data {
		studentName := fmt.Sprintf(
			"%v %v %v",
			student.Student.Name,
			student.Student.FirstLastname,
			student.Student.Rut,
		)

		column := fmt.Sprintf("A%v", i+2)
		file.SetCellValue(sheetName, column, studentName)
		for j, grade := range student.Grades {
			if grade != nil {
				column := fmt.Sprintf("%v%v", string(rune('A'+j+1)), i+2)
				if !grade.IsAcumulative {
					file.SetCellValue(sheetName, column, grade.Grade)
				} else {
					value := ""
					for k, acumulative := range grade.Acumulative {
						if acumulative != nil {
							value += fmt.Sprintf(
								"%v (%v%v) - ",
								acumulative.Grade,
								programs[j].Acumulative[k].Percentage,
								string('%'),
							)
						}
					}
					file.SetCellValue(sheetName, column, value)
				}
			}
		}
	}

	if err := file.Write(w); err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusNotExtended,
		}
	}
	return file, nil
}

func (g *GradesService) ExportGradesStudent(claims *Claims, idSemester string, w io.Writer) *res.ErrorRes {
	// Init PDF
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddUTF8Font("times_utf8", "", "./fonts/times.ttf")
	defer pdf.Close()

	data, err := formatRequestToNestjsNats("")
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	msg, err := nats.Request("get_college_data", data)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}

	// Get college data
	var response stack.NatsNestJSRes
	err = json.Unmarshal(msg.Data, &response)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusInternalServerError,
		}
	}
	// Decode data
	var collegeData map[string]string
	jsonString, err := json.Marshal(response.Response)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusInternalServerError,
		}
	}
	err = json.Unmarshal(jsonString, &collegeData)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusInternalServerError,
		}
	}
	// Get semester
	var semester *models.Semester
	if idSemester == "" {
		semester, err = getCurrentSemester()
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
	} else {
		semester, err = getSemester(idSemester)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
	}
	pdf.SetFont("times_utf8", "", 10)
	pdf.AddPage()
	// Set college data
	contact := fmt.Sprintf("%v - %v", collegeData["phone"], collegeData["email"])
	pdf.Text(5, 10, settingsData.COLLEGE_NAME)
	pdf.Text(5, 15, collegeData["direction"])
	pdf.Text(5, 20, contact)
	// Set semester data
	width, height := pdf.GetPageSize()
	rightMargin := width - 5

	semesterString := fmt.Sprintf("%v° Semestre - %v", semester.Semester, semester.Year)

	pdf.Text(rightMargin-pdf.GetStringWidth(claims.Name), 10, claims.Name)
	pdf.Text(rightMargin-pdf.GetStringWidth(semesterString), 15, semesterString)
	// Footer
	date := fmt.Sprintf("Emitido el %s", time.Now().Format("2006-01-02"))
	pdf.Text(5, height-5, date)
	// Table
	// Grades numbers
	pdf.SetXY(5, 30)
	pdf.CellFormat(
		55,
		4,
		"Materia",
		"1",
		0,
		"",
		false,
		0,
		"",
	)
	var sum float64 = 60
	for i := 0; i < MAX_GRADES; i++ {
		pdf.SetXY(sum, 30)

		grade := strconv.Itoa(i + 1)
		pdf.CellFormat(
			7,
			4,
			grade,
			"1",
			0,
			"C",
			false,
			0,
			"",
		)
		sum += 7
	}
	pdf.SetXY(sum, 30)

	promString := "Prom."
	widthPromString := pdf.GetStringWidth(promString) + 4
	pdf.CellFormat(
		widthPromString,
		4,
		promString,
		"1",
		0,
		"C",
		false,
		0,
		"",
	)

	// Subjects
	var modulesData []models.ModuleWithLookup
	if idSemester == "" {
		courses, err := FindCourses(claims)
		if err != nil {
			return err
		}
		modulesData, err = moduleService.GetModules(courses, claims.UserType, true)
		if err != nil {
			return err
		}
	} else {
		var errRes *res.ErrorRes
		modulesData, _, errRes = moduleService.GetModulesHistory(
			claims.ID,
			0,
			0,
			false,
			true,
			idSemester,
		)
		if errRes != nil {
			return errRes
		}
	}

	var sumHeight float64 = 34
	var averages []float64
	for _, module := range modulesData {
		pdf.SetXY(5, sumHeight)

		pdf.CellFormat(
			55,
			4,
			module.Subject.Subject,
			"1",
			0,
			"",
			false,
			0,
			"",
		)
		sumHeight += 4
		// Get grades
		grades, err := g.GetStudentGrades(module.ID.Hex(), claims.ID)
		if err != nil {
			return err
		}
		// Get program grades
		program, errRes := g.GetGradePrograms(module.ID.Hex())
		if errRes.Err != nil {
			return err
		}
		// Print grades
		var sum float64 = 60
		var average float64

		for i := 0; i < MAX_GRADES; i++ {
			pdf.SetXY(sum, 34)

			var toPrint string
			// Get grade
			for j, p := range program {
				if p.Number == i+1 {
					if grades[j] != nil {
						if !grades[j].IsAcumulative {
							toPrint = strconv.Itoa(int(grades[j].Grade))
							average += (grades[j].Grade * float64(p.Percentage)) / 100
						} else if grades[j].IsAcumulative && len(grades[j].Acumulative) == len(p.Acumulative) {
							var grade float64
							for k, acu := range grades[j].Acumulative {
								if acu != nil {
									grade += (acu.Grade * float64(p.Acumulative[k].Percentage)) / 100
								}
							}
							grade = math.Round(grade)
							toPrint = strconv.Itoa(int(grade))
							average += (grade * float64(p.Percentage)) / 100
						}
						break
					}
				}
			}
			pdf.CellFormat(
				7,
				4,
				toPrint,
				"1",
				0,
				"C",
				false,
				0,
				"",
			)
			sum += 7
		}
		averageRound := math.Round(average)
		// Print average
		pdf.CellFormat(
			widthPromString,
			4,
			strconv.Itoa(int(averageRound)),
			"1",
			0,
			"C",
			false,
			0,
			"",
		)
		// Append average
		averages = append(averages, averageRound)
	}

	var averageFinal float64
	for _, average := range averages {
		averageFinal += average
	}
	averageFinal /= float64(len(averages))
	if semester.Semester == 2 {
		idObjStudent, err := primitive.ObjectIDFromHex(claims.ID)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusBadRequest,
			}
		}
		// Get last semester
		var _idSemester string
		if idSemester == "" {
			_idSemester = semester.ID.Hex()
		} else {
			_idSemester = idSemester
		}
		lastSemester, err := getLastSemester(_idSemester)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
		if lastSemester != nil {
			// Get average
			var average *models.Average
			cursor := averageModel.GetOne(bson.D{
				{
					Key:   "semester",
					Value: lastSemester.ID,
				},
				{
					Key:   "student",
					Value: idObjStudent,
				},
			})
			if err := cursor.Decode(&average); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
				return &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
			}
			if average != nil {
				pdf.SetY(sumHeight)

				x := float64(5 + 42 + 30*7)
				pdf.SetX(x)
				pdf.CellFormat(
					widthPromString,
					4,
					strconv.Itoa(int(average.Average)),
					"1",
					0,
					"C",
					false,
					0,
					"",
				)

				pdf.SetX(x - 15)
				pdf.CellFormat(
					15,
					4,
					"1° Sem:",
					"1",
					0,
					"C",
					false,
					0,
					"",
				)
			}
		}
	}
	// 5 First margin, 55 subject, 30 times * 7 width grades
	x := float64(5 + 55 + 30*7)
	pdf.SetXY(x, sumHeight)
	pdf.CellFormat(
		widthPromString,
		4,
		strconv.Itoa(int(math.Round(averageFinal))),
		"1",
		0,
		"C",
		false,
		0,
		"",
	)

	if err := pdf.Output(w); err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusNotExtended,
		}
	}
	return nil
}
