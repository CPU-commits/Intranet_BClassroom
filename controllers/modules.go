package controllers

import (
	"net/http"
	"strconv"

	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
)

// Services
var moduleService = services.NewModulesService()

type ModulesController struct{}

// Nats
func init() {
	getCourses()
}

func getCourses() {
	moduleService.GetCourses()
}

// Query
func (modules *ModulesController) GetModule(c *gin.Context) {
	idModule := c.Param("idModule")
	// Get module
	moduleData, err := moduleService.GetModule(idModule)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	if moduleData == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, res.Response{
			Success: false,
			Message: "Not found",
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["module"] = moduleData
	c.JSON(200, res.Response{
		Success: true,
		Data:    response,
	})
}

func (modules *ModulesController) GetModules(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)

	courses, err := services.FindCourses(claims)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	modulesData, err := moduleService.GetModules(courses, claims.UserType, false)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["modules"] = modulesData
	c.JSON(200, res.Response{
		Success: true,
		Data:    response,
	})
}

func (module *ModulesController) GetModulesHistory(c *gin.Context) {
	total := c.DefaultQuery("total", "false")
	limit := c.DefaultQuery("limit", "20")
	skip := c.DefaultQuery("skip", "0")
	claims, _ := services.NewClaimsFromContext(c)

	limitNum, err := strconv.Atoi(limit)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: "Limit must be a number",
		})
		return
	}
	skipNum, err := strconv.Atoi(skip)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: "Skip must be a number",
		})
		return
	}
	modules, totalModules, err := moduleService.GetModulesHistory(
		claims.ID,
		limitNum,
		skipNum,
		total == "true",
		false,
		"",
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
	response["modules"] = modules
	response["total"] = totalModules
	c.JSON(200, res.Response{
		Success: true,
		Data:    response,
	})
}

func (modules *ModulesController) Search(c *gin.Context) {
	idModule := c.Param("idModule")
	search := c.DefaultQuery("search", "")
	hits, err := moduleService.Search(idModule, search)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["hits"] = hits
	c.JSON(200, res.Response{
		Success: true,
		Data:    response,
	})
}

func (modules *ModulesController) DownloadFile(c *gin.Context) {
	idModule := c.Param("idModule")
	idFile := c.Param("idFile")

	clamis, _ := services.NewClaimsFromContext(c)
	// Download file
	tokens, err := moduleService.DownloadModuleFile(idModule, idFile, clamis)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["token"] = tokens[0]
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

// Feed
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
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["inserted_id"] = insertedID
	c.JSON(200, res.Response{
		Success: true,
		Data:    response,
	})
}
