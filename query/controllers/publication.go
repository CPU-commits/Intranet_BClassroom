package controllers

import (
	"net/http"
	"strconv"

	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
)

type PublicationController struct{}

// Services
var publicationService = services.NewPublicationsService()

// Query
// GetPublications godoc
// @Summary     Get publications
// @Description Get module publications
// @Tags        publications
// @Tags        classroom
// @Tags        roles.teacher
// @Tags        roles.student
// @Tags        roles.student_directive
// @Accept      json
// @Produce     json
// @Param       idModule path     string  true  "MongoID"
// @Param       section  query    integer false "Section"
// @Param       skip     query    integer false "Skip"
// @Param       limit    query    integer false "Limit"
// @Param       total    query    bool    false "Get total?"
// @Success     200      {object} res.Response{body=smaps.PublicationsMap}
// @Failure     401      {object} res.Response{} "Unauthorized"
// @Failure     401      {object} res.Response{} "Unauthorized role"
// @Failure     404      {object} res.Response{} "No existe esta sección"
// @Failure     503      {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /publications/get_publications/{idModule} [get]
func (publication *PublicationController) GetPublications(c *gin.Context) {
	idModule := c.Param("idModule")
	section := c.DefaultQuery("section", "0")
	skip := c.DefaultQuery("skip", "0")
	limit := c.DefaultQuery("limit", "20")
	total := c.DefaultQuery("total", "false")

	skipNumber, err := strconv.Atoi(skip)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	limitNumber, err := strconv.Atoi(limit)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	totalBool := total == "true"

	publications, totalData, errRes := publicationService.GetPublicationsFromIdModule(
		idModule,
		section,
		skipNumber,
		limitNumber,
		totalBool,
	)
	if err != nil {
		c.AbortWithStatusJSON(errRes.StatusCode, &res.Response{
			Success: false,
			Message: errRes.Err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["publications"] = publications
	response["total"] = totalData
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

// GetPublication godoc
// @Summary     Get publication
// @Description Get module publication
// @Tags        publications
// @Tags        classroom
// @Tags        roles.teacher
// @Tags        roles.student
// @Tags        roles.student_directive
// @Accept      json
// @Produce     json
// @Param       idModule      path     string true "MongoID"
// @Param       idPublication path     string true "MongoID"
// @Success     200           {object} res.Response{body=smaps.PublicationMap}
// @Failure     401           {object} res.Response{} "Unauthorized"
// @Failure     401           {object} res.Response{} "Unauthorized role"
// @Failure     503           {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /publications/get_publication/{idModule}/{idPublication} [get]
func (p *PublicationController) GetPublication(c *gin.Context) {
	idModule := c.Param("idModule")
	idPublication := c.Param("idPublication")
	// Get
	publication, err := publicationService.GetPublication(idModule, idPublication)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["publication"] = publication
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}
