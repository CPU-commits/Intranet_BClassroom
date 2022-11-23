package controllers

import (
	"io"

	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
)

// Services
var workService = services.NewWorksService()

type WorkController struct{}

// Query
// GetModulesWorks godoc
// @Summary Get modules works
// @Desc    Get student modules works
// @Tags    works
// @Tags    classroom
// @Tags    roles.student
// @Tags    roles.student_directive
// @Accept  json
// @Produce json
// @Success 200 {object} res.Response{body=smaps.ModulesWorksMap}
// @Failure 401 {object} res.Response{} "Unauthorized"
// @Failure 401 {object} res.Response{} "Unauthorized role"
// @Failure 503 {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/get_modules_works [get]
func (w *WorkController) GetModulesWorks(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	// Get
	works, err := workService.GetModulesWorks(*claims)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["works"] = works
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

// GetWorks godoc
// @Summary Get works
// @Desc    Get module works
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Tags    roles.student
// @Tags    roles.student_directive
// @Accept  json
// @Produce json
// @Param   idModule path     string true "MongoID"
// @Success 200      {object} res.Response{body=smaps.WorksMap}
// @Failure 401      {object} res.Response{} "Unauthorized"
// @Failure 401      {object} res.Response{} "Unauthorized role"
// @Failure 503      {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/get_works/{idModule} [get]
func (w *WorkController) GetWorks(c *gin.Context) {
	idModule := c.Param("idModule")
	// Get
	works, err := workService.GetWorks(idModule)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["works"] = works
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

// GetWork godoc
// @Summary Get work
// @Desc    Get work
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Tags    roles.student
// @Tags    roles.student_directive
// @Accept  json
// @Produce json
// @Param   idWork path     string true "MongoID"
// @Success 200    {object} res.Response{smaps.WorkMap}
// @Failure 401    {object} res.Response{} "Unauthorized"
// @Failure 401    {object} res.Response{} "Unauthorized role"
// @Failure 503    {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/get_work/{idWork} [get]
func (w *WorkController) GetWork(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	idWork := c.Param("idWork")
	response, err := workService.GetWork(idWork, claims)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

// GetForm godoc
// @Summary Get form
// @Desc    Get form work
// @Tags    works
// @Tags    classroom
// @Tags    roles.student
// @Tags    roles.student_directive
// @Accept  json
// @Produce json
// @Param   idWork path     string true "MongoID"
// @Success 200    {object} res.Response{body=smaps.FormWorkMap}
// @Failure 400    {object} res.Response{} "Bad path param"
// @Failure 400    {object} res.Response{} "Este trabajo no es de tipo formulario"
// @Failure 400    {object} res.Response{} "No accediste al formulario, no hay respuestas a revisar"
// @Failure 401    {object} res.Response{} "No se puede acceder a este trabajo todavía"
// @Failure 401    {object} res.Response{} "Unauthorized"
// @Failure 401    {object} res.Response{} "Unauthorized role"
// @Failure 503    {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/get_form/{idWork} [get]
func (w *WorkController) GetForm(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	idWork := c.Param("idWork")
	// Get
	response, err := workService.GetForm(idWork, claims.ID)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	// Response
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

// GetFormStudent godoc
// @Summary Get form student
// @Desc    Get form student
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idWork    path     string true "MongoID"
// @Param   idStudent path     string true "MongoID"
// @Success 200       {object} res.Response{smaps.FormStudentMap}
// @Failure 400       {object} res.Response{} "Bad path param"
// @Failure 400       {object} res.Response{} "Este trabajo no es de tipo formulario"
// @Failure 400       {object} res.Response{} "Este alumno no ha tiene respuestas, ya que no abrió el formulario"
// @Failure 401       {object} res.Response{} "Este formulario todavía no se puede evalua"
// @Failure 401       {object} res.Response{} "Unauthorized"
// @Failure 401       {object} res.Response{} "Unauthorized role"
// @Failure 503       {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/get_form_student/{idWork}/{idStudent} [get]
func (w *WorkController) GetFormStudent(c *gin.Context) {
	idWork := c.Param("idWork")
	idStudent := c.Param("idStudent")
	// Get
	form, answers, err := workService.GetFormStudent(idWork, idStudent)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["form"] = form
	response["answers"] = answers
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

// GetStudentsStatus godoc
// @Summary Get students status
// @Desc    Get students status
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idModule path     string true "MongoID"
// @Param   idWork   path     string true "MongoID"
// @Success 200      {object} res.Response{body=smaps.StudentsStatusMap}
// @Failure 400      {object} res.Response{} "Bad path param"
// @Failure 400      {object} res.Response{} "Ningún estudiante pertenece a este trabajo"
// @Failure 401      {object} res.Response{} "Unauthorized"
// @Failure 401      {object} res.Response{} "Unauthorized role"
// @Failure 503      {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/get_students_status/{idModule}/{idWork} [get]
func (w *WorkController) GetStudentsStatus(c *gin.Context) {
	idModule := c.Param("idModule")
	idWork := c.Param("idWork")

	// Get
	students, totalPoints, err := workService.GetStudentsStatus(idModule, idWork)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}
	// Response
	response := make(map[string]interface{})
	response["students"] = students
	response["total_points"] = totalPoints
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

// DownloadFilesWorkStudent godoc
// @Summary Download files work student
// @Desc    Download files work student
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce octet-stream
// @Param   idStudent path     string         true "MongoID"
// @Param   idWork    path     string         true "MongoID"
// @Success 200       {file}   binary         "Zip file"
// @Failure 400       {object} res.Response{} "Bad path param"
// @Failure 401       {object} res.Response{} "Unauthorized"
// @Failure 401       {object} res.Response{} "Unauthorized role"
// @Failure 503       {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/download_files_work_student/{idWork}/{idStudent} [get]
func (w *WorkController) DownloadFilesWorkStudent(c *gin.Context) {
	idStudent := c.Param("idStudent")
	idWork := c.Param("idWork")
	c.Writer.Header().Set("Content-type", "application/octet-stream")
	c.Stream(func(w io.Writer) bool {
		// Download Files
		ar, err := workService.DownloadFilesWorkStudent(
			idWork,
			idStudent,
			w,
		)
		if err != nil {
			c.AbortWithStatusJSON(err.StatusCode, &res.Response{
				Success: false,
				Message: err.Err.Error(),
			})
			return false
		}
		c.Writer.Header().Set(
			"Content-Disposition",
			"attachment; filename='filename.zip'",
		)
		ar.Close()
		return false
	})
}
