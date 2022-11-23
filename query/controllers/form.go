package controllers

import (
	"net/http"

	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
)

type FormController struct{}

// Services
var formService = services.NewFormService()

// GetForms godoc
// @Summary     Get forms of user, ROLES=[teacher]
// @Description Get forms generated for user
// @Tags        forms
// @Tags        classroom
// @Tags        roles.teacher
// @Accept      json
// @Product     json
// @Success     200 {object} res.Response{body=smaps.FormsMap}
// @Failure     401 {object} res.Response{} "Unauthorized"
// @Failure     401 {object} res.Response{} "Unauthorized role"
// @Failure     503 {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /forms/get_forms [get]
func (f *FormController) GetForms(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	// Get
	forms, err := formService.GetFormsUser(claims.ID)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["forms"] = forms
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

// GetForm godoc
// @Summary     Get a single form
// @Description Get a single user form, ROLS=[teacher]
// @Tags        forms
// @Tags        classroom
// @Tags        roles.teacher
// @Accept      json
// @Produce     json
// @Success     200 {object} res.Response{body=smaps.FormMap}
// @Failure     401 {object} res.Response{} "Unauthorized"
// @Failure     401 {object} res.Response{} "Unauthorized role"
// @Failure     404 {object} res.Response{} "No existe este formulario"
// @Failure     503 {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /forms/get_form/{idForm} [get]
func (f *FormController) GetForm(c *gin.Context) {
	idForm := c.Param("idForm")
	claims, _ := services.NewClaimsFromContext(c)
	form, err := formService.GetForm(idForm, claims.ID, true)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	if len(form) == 0 {
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: "No existe este formulario",
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["form"] = form[0]
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}
