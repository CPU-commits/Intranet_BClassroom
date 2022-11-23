package controllers

import (
	"net/http"

	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
)

type PublicationController struct{}

// Services
var publicationService = services.NewPublicationsService()

// Feed
// NewPublication godoc
// @Summary New publication
// @Desc    New module publication in sub-section (default: 0)
// @Tags    publications
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idModule path     string  true  "MongoID"
// @Param   section  query    integer false "Default: 0"
// @Success 200      {object} res.Response{body=smaps.NewPublicationMap}
// @Failure 400      {object} res.Response{} "Bad path param"
// @Failure 400      {object} res.Response{} "No existe esta sección"
// @Failure 401      {object} res.Response{} "Unauthorized"
// @Failure 401      {object} res.Response{} "Unauthorized role"
// @Failure 503      {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /publications/upload/{idModule} [post]
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
	response, errRes := publicationService.NewPublication(publicationData, claims, section, idModule)
	if errRes != nil {
		c.AbortWithStatusJSON(errRes.StatusCode, &res.Response{
			Success: false,
			Message: errRes.Err.Error(),
		})
		return
	}
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

// UpdatePublication godoc
// @Summary Update publication
// @Desc    Update module publication
// @Tags    publications
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idPublication path     string true "MongoID"
// @Success 200           {object} res.Response{}
// @Failure 400           {object} res.Response{} "Bad path param"
// @Failure 401           {object} res.Response{} "Unauthorized"
// @Failure 401           {object} res.Response{} "Unauthorized role"
// @Failure 401           {object} res.Response{} "No tienes acceso a esta publicación"
// @Failure 404           {object} res.Response{} "Publication not found"
// @Failure 503           {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /publications/update/{idPublication} [put]
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
	errRes := publicationService.UpdatePublication(content, idPublication, claims.ID)
	if errRes != nil {
		c.AbortWithStatusJSON(errRes.StatusCode, &res.Response{
			Success: false,
			Message: errRes.Err.Error(),
		})
		return
	}
	c.JSON(200, &res.Response{
		Success: true,
	})
}

// DeletePublication godoc
// @Summary Delete publication
// @Desc    Delete module publication
// @Tags    publications
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idModule      path     string true "MongoID"
// @Param   idPublication path     string true "MongoID"
// @Success 200           {object} res.Response{}
// @Failure 400           {object} res.Response{} "Bad path param"
// @Failure 401           {object} res.Response{} "Unauthorized"
// @Failure 401           {object} res.Response{} "Unauthorized role"
// @Failure 401           {object} res.Response{} "Not Access module"
// @Failure 404           {object} res.Response{} "Publication not found"
// @Failure 503           {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /publications/delete/{idPublication}/{idModule} [delete]
func (publication *PublicationController) DeletePublication(c *gin.Context) {
	idModule := c.Param("idModule")
	idPublication := c.Param("idPublication")
	claims, _ := services.NewClaimsFromContext(c)
	// Delete
	err := publicationService.DeletePublication(idModule, idPublication, *claims)
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

// DeletePublicationAttached godoc
// @Summary Delete publication attached
// @Desc    Delete publication attached
// @Tags    publications
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idModule   path     string true "MongoID"
// @Param   idAttached path     string true "MongoID"
// @Success 200        {object} res.Response{}
// @Failure 400        {object} res.Response{} "Esta publicación no tiene elementos adjuntos"
// @Failure 401        {object} res.Response{} "Unauthorized"
// @Failure 401        {object} res.Response{} "Not Access module"
// @Failure 401        {object} res.Response{} "Unauthorized role"
// @Failure 404        {object} res.Response{} "Publication not found"
// @Failure 404        {object} res.Response{} "No existe el elemento adjunto"
// @Failure 503        {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /publications/delete_attached/{idAttached}/{idModule} [delete]
func (publication *PublicationController) DeletePublicationAttached(c *gin.Context) {
	idModule := c.Param("idModule")
	idAttached := c.Param("idAttached")
	claims, _ := services.NewClaimsFromContext(c)
	// Delete
	err := publicationService.DeletePublicationAttached(idModule, idAttached, *claims)
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
