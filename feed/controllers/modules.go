package controllers

import (
	"net/http"

	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
)

// Services
var moduleService = services.NewModulesService()

type ModulesController struct{}

// Feed
// NewSubSection godoc
// @Summary     New sub section
// @Description New sub section to module
// @Tags        modules
// @Tags        classroom
// @Tags        roles.teacher
// @Accept      json
// @Produce     json
// @Param       idModule path     string true "MongoID"
// @Success     201      {object} res.Response{body=smaps.InsertedIdMap}
// @Failure     400      {object} res.Response{} "Bad path params"
// @Failure     401      {object} res.Response{} "Unauthorized"
// @Failure     401      {object} res.Response{} "Unauthorized role"
// @Router      /modules/new_sub_section/{idModule} [post]
func (modules *ModulesController) NewSubSection(c *gin.Context) {
	var subSectionData *forms.SubSectionData
	idModule := c.Param("idModule")
	// Binding
	if err := c.BindJSON(&subSectionData); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Insert
	insertedID, err := moduleService.NewSubSection(subSectionData, idModule)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["inserted_id"] = insertedID
	c.JSON(201, res.Response{
		Success: true,
		Data:    response,
	})
}
