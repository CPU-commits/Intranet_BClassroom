package controllers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
)

// Services
var gradesService = services.NewGradesService()

type GradesController struct{}

// Query
// GetProgramGrade godoc
// @Summary     Get program grade
// @Description Get program grade by MongoId (Sort by program number)
// @Tags        grades
// @Tags        classroom
// @Tags        roles.all
// @Accept      json
// @Produce     json
// @Param       idModule path     string true "Mongo ID Form"
// @Success     200      {object} res.Response{body=smaps.ProgramGradeMap}
// @Failure     401      {object} res.Response{} "Unauthorized"
// @Failure     503      {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /grades/get_grade_programs/{idModule} [get]
func (g *GradesController) GetProgramGrade(c *gin.Context) {
	idModule := c.Param("idModule")
	programs, err := gradesService.GetGradePrograms(idModule)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["programs"] = programs
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

// GetStudentsGrades godoc
// @Summary     Get students grades
// @Description Get all students and their grades (in a module)
// @Tags        grades
// @Tags        classroom
// @Tags        roles.teacher
// @Tags        roles.director
// @Tags        roles.directive
// @Accept      json
// @Produce     json
// @Param       idModule path string true "Mongo ID Form"
// @Sucess      200 {object} res.Response={body=smaps.StudentsGradesMap}
// @Failure     401 {object} res.Response{} "Unauthorized"
// @Failure     401 {object} res.Response{} "Unauthorized role"
// @Failure     503 {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /grades/get_students_grades/{idModule} [get]
func (g *GradesController) GetStudentsGrades(c *gin.Context) {
	idModule := c.Param("idModule")
	// Get grades
	students, err := gradesService.GetStudentsGrades(idModule)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["students"] = students
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

// GetStudentGrades godoc
// @Summary     Get student grades
// @Description Get student grades (in a module)
// @Tags        grades
// @Tags        classroom
// @Tags        roles.student
// @Tags        roles.student_directive
// @Accept      json
// @Produce     json
// @Param       idModule path string true "Mongo ID Form"
// @Sucess      200 {object} res.Response={body=smaps.StudentGradesMap}
// @Failure     401 {object} res.Response{} "Unauthorized"
// @Failure     401 {object} res.Response{} "Unauthorized role"
// @Failure     503 {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /grades/get_student_grades/{idModule} [get]
func (g *GradesController) GetStudentGrades(c *gin.Context) {
	idModule := c.Param("idModule")
	claims, _ := services.NewClaimsFromContext(c)

	grades, err := gradesService.GetStudentGrades(idModule, claims.ID)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["grades"] = grades
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

// ExportGrades godoc
// @Summary     Export grades
// @Description Get student grades (in a module) -> export to Excel
// @Tags        grades
// @Tags        classroom
// @Tags        roles.teacher
// @Tags        roles.directive
// @Tags        roles.director
// @Accept      json
// @Produce     application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param       idModule path string true "Mongo ID Form"
// @Sucess      200 {file} io.Writer "Excel File"
// @Failure     401 {object} res.Response{} "Unauthorized"
// @Failure     401 {object} res.Response{} "Unauthorized role"
// @Failure     503 {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Failure     510 {object} res.Response{} "Buffer io.Writter"
// @Router      /grades/export_grades/{idModule} [get]
func (g *GradesController) ExportGrades(c *gin.Context) {
	idModule := c.Param("idModule")

	c.Writer.Header().Set(
		"Content-type",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	)

	c.Stream(func(w io.Writer) bool {
		file, err := gradesService.ExportGrades(idModule, w)
		if err != nil {
			c.AbortWithStatusJSON(err.StatusCode, &res.Response{
				Success: false,
				Message: err.Err.Error(),
			})
			return false
		}

		c.Writer.Header().Set(
			"Content-Disposition",
			"attachment; filename='filename.zip'",
		)
		if err := file.Close(); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
				Success: false,
				Message: err.Error(),
			})
			return false
		}
		return false
	})
}

// ExportGradesStudent godoc
// @Summary     Export student grades
// @Description Get student grades (in a semester) -> export to PDF
// @Tags        grades
// @Tags        classroom
// @Tags        roles.student
// @Tags        roles.student_directive
// @Accept      json
// @Produce     application/pdf
// @Param       semester query string true "MongoID"
// @Sucess      200 {file} binary "PDF File"
// @Failure     401 {object} res.Response{} "Unauthorized"
// @Failure     401 {object} res.Response{} "Unauthorized role"
// @Failure     503 {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Failure     510 {object} res.Response{} "Buffer io.Writter"
// @Router      /grades/download_grades [get]
func (g *GradesController) ExportGradesStudent(c *gin.Context) {
	semester := c.DefaultQuery("semester", "")

	claims, _ := services.NewClaimsFromContext(c)

	c.Stream(func(w io.Writer) bool {
		err := gradesService.ExportGradesStudent(claims, semester, w)
		if err != nil {
			c.AbortWithStatusJSON(err.StatusCode, &res.Response{
				Success: false,
				Message: err.Err.Error(),
			})
			return false
		}

		c.Writer.Header().Set(
			"Content-type",
			"application/pdf",
		)
		c.Writer.Header().Set(
			"Content-Disposition",
			fmt.Sprintf("attachment; filename='%s.zip'", claims.Name),
		)
		return false
	})
}
