package controllers

import (
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

// Feed
// UploadProgramGrade godoc
// @Summary     Upload program grade
// @Description Upload program grade in module, ROLS=[teacher]
// @Tags        grades
// @Tags        classroom
// @Tags        roles.teacher
// @Accept      json
// @Produce     json
// @Param       idModule     path     string                 true "MongoID"
// @Param       programGrade body     forms.GradeProgramForm true "Desc"
// @Success     201          {object} res.Response{body=smaps.IdInsertedMap}
// @Failure     400          {object} res.Response{} "Bad body"
// @Failure     400          {object} res.Response{} "El porcentaje indicado superado el 100 por ciento. Queda %v por ciento libre"
// @Failure     400          {object} res.Response{} "El porcentaje indicado superado el 100 por ciento."
// @Failure     400          {object} res.Response{} "El porcentaje sumatorio de las calificaciones acumulativas debe ser exactamente 100 por cierto"
// @Failure     401          {object} res.Response{} "Unauthorized"
// @Failure     401          {object} res.Response{} "Unauthorized role"
// @Failure     409          {object} res.Response{} "Esta calificación ya está programada"
// @Failure     503          {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /grades/upload_program/{idModule} [post]
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
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}

	// Response
	response := make(map[string]interface{})
	response["_id"] = id.(primitive.ObjectID).Hex()

	c.JSON(201, &res.Response{
		Success: true,
		Data:    response,
	})
}

// UploadGrade godoc
// @Summary     Upload grade
// @Description Upload grade in module to student, ROLS=[teacher]
// @Tags        grades
// @Tags        classroom
// @Tags        roles.teacher
// @Accept      json
// @Produce     json
// @Param       idModule  path     string          true "MongoID"
// @Param       idStudent path     string          true "MongoID"
// @Param       grade     body     forms.GradeForm true "Desc"
// @Success     201       {object} res.Response{body=smaps.IdInsertedMap}
// @Failure     400       {object} res.Response{} "Bad body"
// @Failure     400       {object} res.Response{} "Calificación inválida. Mín: %v. Máx: %v"
// @Failure     401       {object} res.Response{} "Unauthorized"
// @Failure     401       {object} res.Response{} "Unauthorized role"
// @Failure     409       {object} res.Response{} "La calificación acumulativa no existe"
// @Failure     409       {object} res.Response{} "No se puede agregar una calificación ya subida"
// @Failure     503       {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /grades/upload_grade/{idModule}/{idStudent} [post]
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
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["_id"] = idInserted.(primitive.ObjectID).Hex()
	c.JSON(201, &res.Response{
		Success: true,
		Data:    response,
	})
}

// UpdateGrade godoc
// @Summary     Update grade
// @Description Update grade in module to student, ROLS=[teacher]
// @Tags        grades
// @Tags        classroom
// @Tags        roles.teacher
// @Accept      json
// @Param       idModule path     string                true "MongoID"
// @Param       idGrade  path     string                true "MongoID"
// @Param       grade    body     forms.UpdateGradeForm true "Desc"
// @Success     201      {object} res.Response{}
// @Failure     400      {object} res.Response{} "Bad body"
// @Failure     400      {object} res.Response{} "Calificación inválida. Mín: %v. Máx: %v"
// @Failure     401      {object} res.Response{} "Unauthorized"
// @Failure     401      {object} res.Response{} "Unauthorized role"
// @Failure     404      {object} res.Response{} "No existe la programación de calificación"
// @Failure     409      {object} res.Response{} "Esta calificación no pertenece al módulo"
// @Failure     503      {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"..
// @Router      /grades/update_grade/{idModule}/{idGrade} [put]
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
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	c.JSON(200, &res.Response{
		Success: true,
	})
}

// DeleteGradeProgram godoc
// @Summary     Delete grade program
// @Description Delete grade program in module, ROLS=[teacher]
// @Tags        grades
// @Tags        classroom
// @Tags        roles.teacher
// @Accept      json
// @Param       idModule  path     string true "MongoID"
// @Param       idProgram path     string true "MongoID"
// @Success     200       {object} res.Response{}
// @Failure     401       {object} res.Response{} "Unauthorized"
// @Failure     401       {object} res.Response{} "Unauthorized role"
// @Failure     404       {object} res.Response{} "No existe la programación de calificación"
// @Failure     409       {object} res.Response{} "Esta programación de calificación no pertenece al módulo indicado"
// @Failure     409       {object} res.Response{} "Esta programación está en uso en el trabajo %s"
// @Failure     409       {object} res.Response{} "Esta programación está en uso en alguna calificación"
// @Failure     503       {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"..
// @Router      /grades/delete_program/{idModule}/{idProgram} [delete]
func (g *GradesController) DeleteGradeProgram(c *gin.Context) {
	idModule := c.Param("idModule")
	idProgram := c.Param("idProgram")

	if err := gradesService.DeleteGradeProgram(idModule, idProgram); err != nil {
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	c.JSON(200, &res.Response{
		Success: true,
	})
}
