package controllers

import (
	"net/http"
	"strconv"

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
// GetModule godoc
// @Summary     Get module
// @Description Get module
// @Tags        modules
// @Tags        classroom
// @Tags        roles.teacher
// @Tags        roles.student
// @Tags        roles.student_directive
// @Accept      json
// @Produce     json
// @Param       idModule path     string true "MongoID"
// @Success     200      {object} res.Response{body=smaps.ModuleMap}
// @Failure     401      {object} res.Response{} "Unauthorized"
// @Failure     401      {object} res.Response{} "Unauthorized role"
// @Failure     404      {object} res.Response{} "Not found"
// @Failure     503      {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /modules/get_module/{idModule} [get]
func (modules *ModulesController) GetModule(c *gin.Context) {
	idModule := c.Param("idModule")
	// Get module
	moduleData, err := moduleService.GetModule(idModule)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, res.Response{
			Success: false,
			Message: err.Err.Error(),
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

// GetModules godoc
// GetModule godoc
// @Summary     Get modules
// @Description Get user modules
// @Tags        modules
// @Tags        classroom
// @Tags        roles.teacher
// @Tags        roles.student
// @Tags        roles.student_directive
// @Accept      json
// @Produce     json
// @Success     200 {object} res.Response{body=smaps.ModulesMap}
// @Failure     401 {object} res.Response{} "Unauthorized"
// @Failure     401 {object} res.Response{} "Unauthorized role"
// @Failure     403 {object} res.Response{} "No estás asignado a ningún curso"
// @Failure     503 {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /modules/get_modules [get]
func (modules *ModulesController) GetModules(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)

	courses, err := services.FindCourses(claims)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	modulesData, err := moduleService.GetModules(courses, claims.UserType, false)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, res.Response{
			Success: false,
			Message: err.Err.Error(),
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

// GetModulesHistory godoc
// @Tags    modules
// @Tags    roles.student
// @Tags    roles.student_directive
// @Accept  json
// @Produce json
// @Param   total query boolean false "Get total?"
// @Param   limit query integer false "Limit"
// @Param   skip  query integer false "Skip"
// @Sucess  200 {object} res.Response{body=smaps.ModulesHistoryMap}
// @Failure 400 {object} res.Response{} "Bad query param"
// @Failure 401 {object} res.Response{} "Unauthorized"
// @Failure 401 {object} res.Response{} "Unauthorized role"
// @Failure 503 {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /modules/get_modules_history [get]
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
	modules, totalModules, errRes := moduleService.GetModulesHistory(
		claims.ID,
		limitNum,
		skipNum,
		total == "true",
		false,
		"",
	)
	if errRes != nil {
		c.AbortWithStatusJSON(errRes.StatusCode, &res.Response{
			Success: false,
			Message: errRes.Err.Error(),
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

// Search godoc
// @Summary     search
// @Description search in all module content
// @Tags        modules
// @Tags        classroom
// @Tags        roles.student
// @Tags        roles.student_directive
// @Accept      json
// @Produce     json
// @Param       idModule path     string true  "Desc"
// @Param       search   query    string false "Search"
// @Success     200      {object} res.Response{body=smaps.SearchHitsMap}
// @Failure     401      {object} res.Response{} "Unauthorized"
// @Failure     401      {object} res.Response{} "Unauthorized role"
// @Failure     503      {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /modules/search/{idModule} [get]
func (modules *ModulesController) Search(c *gin.Context) {
	idModule := c.Param("idModule")
	search := c.DefaultQuery("search", "")
	hits, err := moduleService.Search(idModule, search)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, res.Response{
			Success: false,
			Message: err.Err.Error(),
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

// DownloadFileModule godoc
// @Summary     Download file module
// @Description Download file of module
// @Tags        modules
// @Tags        classroom
// @Tags        roles.teacher
// @Tags        roles.student
// @Tags        roles.student_directive
// @Accept      json
// @Produce     json
// @Param       idModule path     string true "MongoID"
// @Param       idFile   path     string true "MongoID"
// @Success     200      {object} res.Response{body=smaps.TokenMap}
// @Failure     400      {object} res.Response{} "Bad path params"
// @Failure     401      {object} res.Response{} "Unauthorized"
// @Failure     401      {object} res.Response{} "No tienes acceso a este archivo"
// @Failure     401      {object} res.Response{} "Unauthorized role"
// @Failure     503      {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router      /modules/download_file/{idFile}/{idModule} [get]
func (modules *ModulesController) DownloadFile(c *gin.Context) {
	idModule := c.Param("idModule")
	idFile := c.Param("idFile")

	clamis, _ := services.NewClaimsFromContext(c)
	// Download file
	tokens, err := moduleService.DownloadModuleFile(idModule, idFile, clamis)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
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
