package controllers

import (
	"net/http"

	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
)

type FormController struct{}

// Services
var formService = services.NewFormService()

// Feed
// UploadForm godoc
// @Summary     Upload form to user
// @Description Upload a form to teacher, ROLS=[teacher]
// @Tags        classroom
// @Tags        forms
// @Tags        roles.teacher
// @Accept      json
// @Produce     json
// @Param       form body     forms.FormForm true "Add form"
// @Success     201  {object} res.Response{}
// @Failure     400  {object} res.Response{} "Bad request - Bad body"
// @Failure     401  {object} res.Response{} "Unauthorized"
// @Failure     401  {object} res.Response{} "Unauthorized role"
// @Failure     503  {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /forms/upload_form [post]
func (f *FormController) UploadForm(c *gin.Context) {
	var form *forms.FormForm
	claims, _ := services.NewClaimsFromContext(c)

	if err := c.BindJSON(&form); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// Insert
	err := formService.UploadForm(form, claims.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	c.JSON(http.StatusCreated, &res.Response{
		Success: true,
	})
}

// UpdateForm godoc
// @Summary     Update form to user
// @Description Update a form to teacher, ROLS=[teacher]
// @Tags        classroom
// @Tags        forms
// @Tags        roles.teacher
// @Accept      json
// @Produce     json
// @Param       form   body     forms.FormForm true "Update form"
// @Param       idForm path     string         true "Mongo ID Form"
// @Success     200    {object} res.Response{}
// @Failure     400    {object} res.Response{} "Bad request - Bad body"
// @Failure     401    {object} res.Response{} "Unauthorized"
// @Failure     401    {object} res.Response{} "No tienes acceso para editar este formulario"
// @Failure     401    {object} res.Response{} "Unauthorized role"
// @Failure     403    {object} res.Response{} "Este formulario no está disponible"
// @Failure     403    {object} res.Response{} "Este formulario está asignado a un trabajo revisado, no se puede editar"
// @Failure     503    {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /forms/update_form/{idForm} [put]
func (f *FormController) UpdateForm(c *gin.Context) {
	idForm := c.Param("idForm")
	var form *forms.FormForm
	claims, _ := services.NewClaimsFromContext(c)

	if err := c.BindJSON(&form); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// Update
	err := formService.UpdateForm(form, claims.ID, idForm)
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

// DeleteForm godoc
// @Summary     Delete form of user
// @Description Form own deletes(soft) a form, ROLS=[teacher]
// @Tags        classroom
// @Tags        forms
// @Tags        roles.teacher
// @Accept      json
// @Produce     json
// @Param       form   body     forms.FormForm true "Update form"
// @Param       idForm path     string         true "Mongo ID Form"
// @Success     200    {object} res.Response{}
// @Failure     400    {object} res.Response{} "Bad request - Bad body"
// @Failure     401    {object} res.Response{} "Unauthorized"
// @Failure     401    {object} res.Response{} "Unauthorized role"
// @Failure     401    {object} res.Response{} "No estás autorizado a eliminar este formulario"
// @Failure     403    {object} res.Response{} "Este formulario ya está eliminado"
// @Failure     403    {object} res.Response{} "Este formulario está asignado a un trabajo revisado, no se puede editar"
// @Failure     503    {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /forms/delete_form/{idForm} [delete]
func (f *FormController) DeleteForm(c *gin.Context) {
	idForm := c.Param("idForm")
	claims, _ := services.NewClaimsFromContext(c)

	// Delete
	err := formService.DeleteForm(idForm, claims.ID)
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
