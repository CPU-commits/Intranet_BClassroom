package controllers

import (
	"net/http"

	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
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
	if err := gradesService.UploadProgram(programGrade, idModule); err != nil {
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
