package controllers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Services
var gradesService = services.NewGradesService()

type GradesController struct{}

// Query
func (g *GradesController) GetProgramGrade(c *gin.Context) {
	idModule := c.Param("idModule")
	programs, err := gradesService.GetGradePrograms(idModule)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
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

func (g *GradesController) GetStudentsGrades(c *gin.Context) {
	idModule := c.Param("idModule")
	// Get grades
	students, err := gradesService.GetStudentsGrades(idModule)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
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

func (g *GradesController) GetStudentGrades(c *gin.Context) {
	idModule := c.Param("idModule")
	claims, _ := services.NewClaimsFromContext(c)

	grades, err := gradesService.GetStudentGrades(idModule, claims.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
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

func (g *GradesController) ExportGrades(c *gin.Context) {
	idModule := c.Param("idModule")

	c.Writer.Header().Set(
		"Content-type",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	)

	c.Stream(func(w io.Writer) bool {
		file, err := gradesService.ExportGrades(idModule, w)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
				Success: false,
				Message: err.Error(),
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

func (g *GradesController) ExportGradesStudent(c *gin.Context) {
	semester := c.DefaultQuery("semester", "")

	claims, _ := services.NewClaimsFromContext(c)
	c.Writer.Header().Set(
		"Content-type",
		"application/pdf",
	)

	c.Stream(func(w io.Writer) bool {
		err := gradesService.ExportGradesStudent(claims, semester, w)
		if err != nil {
			fmt.Printf("err.Error(): %v\n", err.Error())
			c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
				Success: false,
				Message: err.Error(),
			})
			return false
		}

		c.Writer.Header().Set(
			"Content-Disposition",
			fmt.Sprintf("attachment; filename='%s.zip'", claims.Name),
		)
		return false
	})
}

// Feed
func (g *GradesController) UploadProgramGrade(c *gin.Context) {
	idModule := c.Param("idModule")
	var programGrade *forms.GradeProgramForm
	if err := c.BindJSON(&programGrade); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Upload
	id, err := gradesService.UploadProgram(programGrade, idModule)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// Response
	response := make(map[string]interface{})
	response["_id"] = id.(primitive.ObjectID).Hex()

	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

func (g *GradesController) UploadGrade(c *gin.Context) {
	var grade *forms.GradeForm
	idModule := c.Param("idModule")
	idStudent := c.Param("idStudent")
	claims, _ := services.NewClaimsFromContext(c)

	if err := c.BindJSON(&grade); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	idInserted, err := gradesService.UploadGrade(grade, idModule, idStudent, claims.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["_id"] = idInserted.(primitive.ObjectID).Hex()
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

func (g *GradesController) UpdateGrade(c *gin.Context) {
	var grade *forms.UpdateGradeForm
	idModule := c.Param("idModule")
	idGrade := c.Param("idGrade")

	if err := c.BindJSON(&grade); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	err := gradesService.UpdateGrade(grade, idModule, idGrade)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	c.JSON(200, &res.Response{
		Success: true,
	})
}

func (g *GradesController) DeleteGradeProgram(c *gin.Context) {
	idModule := c.Param("idModule")
	idProgram := c.Param("idProgram")

	if err := gradesService.DeleteGradeProgram(idModule, idProgram); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	c.JSON(200, &res.Response{
		Success: true,
	})
}
