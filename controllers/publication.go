package controllers

import (
	"net/http"
	"strconv"

	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
)

type PublicationController struct{}

// Services
var publicationService = services.NewPublicationsService()

// Query
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

	publications, totalData, err := publicationService.GetPublicationsFromIdModule(
		idModule,
		section,
		skipNumber,
		limitNumber,
		totalBool,
	)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
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

// Feed
func (publication *PublicationController) NewPublication(c *gin.Context) {
	idModule := c.Param("idModule")
	section := c.DefaultQuery("section", "0")
	claims, _ := services.NewClaimsFromContext(c)
	var publicationData *forms.PublicationForm

	if err := c.ShouldBindJSON(&publicationData); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Insert
	response, err := publicationService.NewPublication(publicationData, claims, section, idModule)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

func (publication *PublicationController) UpdatePublication(c *gin.Context) {
	var content *forms.PublicationUpdateForm
	idPublication := c.Param("idPublication")
	claims, _ := services.NewClaimsFromContext(c)

	if err := c.BindJSON(&content); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Update
	err := publicationService.UpdatePublication(content, idPublication, claims.ID)
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

func (publication *PublicationController) DeletePublication(c *gin.Context) {
	idModule := c.Param("idModule")
	idPublication := c.Param("idPublication")
	claims, _ := services.NewClaimsFromContext(c)
	// Delete
	err := publicationService.DeletePublication(idModule, idPublication, *claims)
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

func (publication *PublicationController) DeletePublicationAttached(c *gin.Context) {
	idModule := c.Param("idModule")
	idAttached := c.Param("idAttached")
	claims, _ := services.NewClaimsFromContext(c)
	// Delete
	err := publicationService.DeletePublicationAttached(idModule, idAttached, *claims)
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
