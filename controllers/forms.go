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

// Query
func (f *FormController) GetForms(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	// Get
	forms, err := formService.GetFormsUser(claims.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
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

func (f *FormController) GetForm(c *gin.Context) {
	idForm := c.Param("idForm")
	claims, _ := services.NewClaimsFromContext(c)
	form, err := formService.GetForm(idForm, claims.ID, true)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	if len(form) == 0 {
		c.AbortWithStatusJSON(http.StatusNotFound, &res.Response{
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

// Feed
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
			Message: err.Error(),
		})
		return
	}
	c.JSON(200, &res.Response{
		Success: true,
	})
}

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

func (f *FormController) DeleteForm(c *gin.Context) {
	idForm := c.Param("idForm")
	claims, _ := services.NewClaimsFromContext(c)

	// Delete
	err := formService.DeleteForm(idForm, claims.ID)
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
