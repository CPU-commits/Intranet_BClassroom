package controllers

import (
	"net/http"

	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
)

// Services
var workService = services.NewWorksService()

type WorkController struct{}

// Feed
// UploadWork godoc
// @Summary Upload work
// @Desc    Upload module work
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idModule path     string         true "MongoID"
// @Param   work     body     forms.WorkForm true "Desc"
// @Success 201      {object} res.Response{}
// @Failure 400      {object} res.Response{} "Bad body"
// @Failure 400      {object} res.Response{} "Bad path param"
// @Failure 401      {object} res.Response{} "Unauthorized"
// @Failure 401      {object} res.Response{} "Unauthorized role"
// @Failure 503      {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/upload_work/{idModule} [post]
func (w *WorkController) UploadWork(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	idModule := c.Param("idModule")
	var work *forms.WorkForm
	if err := c.BindJSON(&work); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Insert
	if err := workService.UploadWork(work, idModule, claims); err != nil {
		c.AbortWithStatusJSON(err.StatusCode, &res.Response{
			Success: false,
			Message: err.Err.Error(),
		})
		return
	}

	c.JSON(201, &res.Response{
		Success: true,
	})
}

// SaveAnswer godoc
// @Summary Save answer
// @Desc    Save form answer
// @Tags    works
// @Tags    classroom
// @Tags    roles.student
// @Tags    roles.student_directive
// @Accept  json
// @Produce json
// @Param   idWork     path     string           true "MongoID"
// @Param   idQuestion path     string           true "MongoID"
// @Param   answer     body     forms.AnswerForm true "Desc"
// @Success 200        {object} res.Response{}
// @Failure 400        {object} res.Response{} "Bad body"
// @Failure 400        {object} res.Response{} "Bad path param"
// @Failure 400        {object} res.Response{} "El trabajo no es de tipo formulario"
// @Failure 401        {object} res.Response{} "Ya no puedes acceder al formulario"
// @Failure 401        {object} res.Response{} "Ya no se puede acceder al formulario"
// @Failure 401        {object} res.Response{} "Unauthorized"
// @Failure 401        {object} res.Response{} "Unauthorized role"
// @Failure 503        {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/save_answer/{idWork}/{idQuestion} [post]
func (w *WorkController) SaveAnswer(c *gin.Context) {
	var answer *forms.AnswerForm
	if err := c.BindJSON(&answer); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	idWork := c.Param("idWork")
	idQuestion := c.Param("idQuestion")
	claims, _ := services.NewClaimsFromContext(c)
	// Save
	err := workService.SaveAnswer(answer, idWork, idQuestion, claims.ID)
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

// UploadFiles godoc
// @Summary Upload files
// @Desc    Upload files to work
// @Tags    works
// @Tags    classroom
// @Tags    roles.student
// @Tags    roles.student_directive
// @Accept  mpfd
// @Produce json
// @Param   idWork  path     string true "MongoID"
// @Param   files[] formData []file true "Form Data in files[]"
// @Success 200     {object} res.Response{}
// @Failrue 400 {object} res.Response{} "Bad path param"
// @Failrue 400 {object} res.Response{} "Bad formData"
// @Failure 401 {object} res.Response{} "Todavía no se puede acceder a este trabajo"
// @Failure 401 {object} res.Response{} "Ya no se pueden subir archivos a este trabajo"
// @Failure 401 {object} res.Response{} "Unauthorized"
// @Failure 401 {object} res.Response{} "Unauthorized role"
// @Failure 413 {object} res.Response{} "Solo se puede subir hasta 3 archivos por trabajo"
// @Failure 503 {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/upload_files/{idWork} [post]
func (w *WorkController) UploadFiles(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	idWork := c.Param("idWork")
	form, err := c.MultipartForm()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	files := form.File["files[]"]
	if len(files) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: "Debe existir mín. 1 archivo a subir",
		})
		return
	}
	errRes := workService.UploadFiles(files, idWork, claims.ID)
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

// FinishForm godoc
// @Summary Finish form
// @Desc    Finish form
// @Tags    works
// @Tags    classroom
// @Tags    roles.student
// @Tags    roles.student_directive
// @Accept  json
// @Produce json
// @Param   idWork  path     string            true "MongoID"
// @Param   answers body     forms.AnswersForm true "Desc"
// @Success 200     {object} res.Response{}
// @Failrue 400 {object} res.Response{} "Bad path param"
// @Failrue 400 {object} res.Response{} "Bad body"
// @Failure 401 {object} res.Response{} "Ya no se pueden modificar las respuestas de este formulario"
// @Failure 401 {object} res.Response{} "Unauthorized"
// @Failure 401 {object} res.Response{} "Unauthorized role"
// @Failure 503 {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/finish_form/{idWork} [post]
func (w *WorkController) FinishForm(c *gin.Context) {
	var answers *forms.AnswersForm
	idWork := c.Param("idWork")
	claims, _ := services.NewClaimsFromContext(c)
	if err := c.BindJSON(&answers); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Finish
	errRes := workService.FinishForm(answers, idWork, claims.ID)
	if errRes != nil {
		c.AbortWithStatusJSON(errRes.StatusCode, &res.Response{
			Message: errRes.Err.Error(),
			Success: false,
		})
		return
	}
	c.JSON(200, &res.Response{
		Success: true,
	})
}

// UploadPointsQuestion godoc
// @Summary Upload points question
// @Desc    Upload points question
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idWork     path     string                 true "MongoID"
// @Param   idQuestion path     string                 true "MongoID"
// @Param   idStudent  path     string                 true "MongoID"
// @Param   points     body     forms.EvaluateQuestion true "Desc"
// @Success 200        {object} res.Response{}
// @Failrue 400 {object} res.Response{} "Bad path param"
// @Failrue 400 {object} res.Response{} "Bad body"
// @Failure 400 {object} res.Response{} "No se puede evaluar un formulario sin puntos"
// @Failure 400 {object} res.Response{} "Puntaje fuera de rango. Debe ser entre cero y máx %v"
// @Failure 401 {object} res.Response{} "Unauthorized"
// @Failure 401 {object} res.Response{} "Unauthorized role"
// @Failure 401 {object} res.Response{} "Todavía no se pueden evaluar preguntas en este formulario"
// @Failure 404 {object} res.Response{} "No existe la programación de calificación"
// @Failure 503 {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/upload_points_question/{idWork}/{idQuestion}/{idStudent} [post]
func (w *WorkController) UploadPointsQuestion(c *gin.Context) {
	var points *forms.EvaluateQuestion
	idWork := c.Param("idWork")
	idQuestion := c.Param("idQuestion")
	idStudent := c.Param("idStudent")
	claims, _ := services.NewClaimsFromContext(c)

	if err := c.BindJSON(&points); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Upload
	err := workService.UploadPointsStudent(*points.Points, claims.ID, idWork, idQuestion, idStudent)
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

// UploadEvaluateFiles godoc
// @Summary Upload evaluate files
// @Desc    Upload evaluate files
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idWork    path     string                    true "MongoID"
// @Param   idStudent path     string                    true "MongoID"
// @Param   evaluate  body     []forms.EvaluateFilesForm true "Desc"
// @Failure 400       {object} res.Response{}            "Bad body"
// @Failure 400       {object} res.Response{}            "Este trabajo no es de tipo archivos"
// @Failure 400       {object} res.Response{}            "Bad body"
// @Failure 400       {object} res.Response{}            "Los puntos evaluados superan el máx. del item"
// @Failure 401       {object} res.Response{}            "Unauthorized"
// @Failure 401       {object} res.Response{}            "Todavía no se puede evaluar el trabajo"
// @Failure 401       {object} res.Response{}            "Ya no se puede actualizar el puntaje del alumno"
// @Failure 401       {object} res.Response{}            "Unauthorized role"
// @Failure 404       {object} res.Response{}            "No existe la programación de calificación"
// @Failure 404       {object} res.Response{}            "No existe el item #%s en este trabajo"
// @Failure 503       {object} res.Response{}            "Service Unavailable - NATS || DB Service Unavailable"
// @Failure 503       {object} res.Response{}            "No se encontraron archivos subidos por parte del alumno"
// @Success 200       {object} res.Response{}
// @Router  /works/upload_evaluate_files/{idWork}/{idStudent} [post]
func (w *WorkController) UploadEvaluateFiles(c *gin.Context) {
	var evaluate []forms.EvaluateFilesForm
	claims, _ := services.NewClaimsFromContext(c)
	idWork := c.Param("idWork")
	idStudent := c.Param("idStudent")

	if err := c.BindJSON(&evaluate); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Upload
	err := workService.UploadEvaluateFiles(evaluate, idWork, claims.ID, idStudent, false)
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

// UploadReEvaluateFiles godoc
// @Summary Upload reevaluate files
// @Desc    Upload reevaluate files
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idWork    path     string                    true "MongoID"
// @Param   idStudent path     string                    true "MongoID"
// @Param   evaluate  body     []forms.EvaluateFilesForm true "Desc"
// @Failure 400       {object} res.Response{}            "Bad body"
// @Failure 400       {object} res.Response{}            "Este trabajo no es de tipo archivos"
// @Failure 400       {object} res.Response{}            "Bad path param"
// @Failure 400       {object} res.Response{}            "Los puntos evaluados superan el máx. del item"
// @Failure 401       {object} res.Response{}            "Unauthorized"
// @Failure 401       {object} res.Response{}            "Todavía no se puede evaluar el trabajo"
// @Failure 401       {object} res.Response{}            "Ya no se puede actualizar el puntaje del alumno"
// @Failure 401       {object} res.Response{}            "Unauthorized role"
// @Failure 404       {object} res.Response{}            "No existe la programación de calificación"
// @Failure 404       {object} res.Response{}            "No existe el item #%s en este trabajo"
// @Failure 503       {object} res.Response{}            "Service Unavailable - NATS || DB Service Unavailable"
// @Failure 503       {object} res.Response{}            "No se encontraron archivos subidos por parte del alumno"
// @Success 200       {object} res.Response{}
// @Router  /works/upload_reevaluate_files/{idWork}/{idStudent} [post]
func (w *WorkController) UploadReEvaluateFiles(c *gin.Context) {
	var evaluate []forms.EvaluateFilesForm
	idWork := c.Param("idWork")
	idStudent := c.Param("idStudent")
	claims, _ := services.NewClaimsFromContext(c)
	if err := c.BindJSON(&evaluate); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Upload
	err := workService.UploadEvaluateFiles(evaluate, idWork, claims.ID, idStudent, true)
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

// UploadEvaluateInperson godoc
// @Summary Upload evaluate Inperson
// @Desc    Upload evaluate Inperson
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idWork    path     string                    true "MongoID"
// @Param   idStudent path     string                    true "MongoID"
// @Param   evaluate  body     forms.EvaluateInperson true "Desc"
// @Failure 400       {object} res.Response{}            "Bad body"
// @Failure 400       {object} res.Response{}            "Este trabajo no es de tipo presencial"
// @Failure 400       {object} res.Response{}            "Bad body"
// @Failure 401       {object} res.Response{}            "Unauthorized"
// @Failure 401       {object} res.Response{}            "Todavía no se puede evaluar el trabajo"
// @Failure 401       {object} res.Response{}            "Ya no se puede actualizar el puntaje del alumno"
// @Failure 401       {object} res.Response{}            "Unauthorized role"
// @Failure 404       {object} res.Response{}            "No existe la programación de calificación"
// @Failure 503       {object} res.Response{}            "Service Unavailable - NATS || DB Service Unavailable"
// @Failure 503       {object} res.Response{}            "No se encontraron archivos subidos por parte del alumno"
// @Success 200       {object} res.Response{}
// @Router  /works/upload_evaluate_inperson/{idWork}/{idStudent} [post]
func (w *WorkController) UploadEvaluateInperson(c *gin.Context) {
	var evaluate *forms.EvaluateInperson
	claims, _ := services.NewClaimsFromContext(c)
	idWork := c.Param("idWork")
	idStudent := c.Param("idStudent")

	if err := c.BindJSON(&evaluate); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Upload
	err := workService.UploadEvaluateInperson(evaluate, idWork, claims.ID, idStudent, false)
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

// UploadReEvaluatePerson godoc
// @Summary Upload reevaluate Person
// @Desc    Upload reevaluate Person
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idWork    path     string                    true "MongoID"
// @Param   idStudent path     string                    true "MongoID"
// @Param   evaluate  body     forms.EvaluateInperson true "Desc"
// @Failure 400       {object} res.Response{}            "Bad body"
// @Failure 400       {object} res.Response{}            "Este trabajo no es de tipo archivos"
// @Failure 400       {object} res.Response{}            "Bad path param"
// @Failure 400       {object} res.Response{}            "Los puntos evaluados superan el máx. del item"
// @Failure 401       {object} res.Response{}            "Unauthorized"
// @Failure 401       {object} res.Response{}            "Todavía no se puede evaluar el trabajo"
// @Failure 401       {object} res.Response{}            "Ya no se puede actualizar el puntaje del alumno"
// @Failure 401       {object} res.Response{}            "Unauthorized role"
// @Failure 404       {object} res.Response{}            "No existe la programación de calificación"
// @Failure 404       {object} res.Response{}            "No existe el item #%s en este trabajo"
// @Failure 503       {object} res.Response{}            "Service Unavailable - NATS || DB Service Unavailable"
// @Failure 503       {object} res.Response{}            "No se encontraron archivos subidos por parte del alumno"
// @Success 200       {object} res.Response{}
// @Router  /works/upload_reevaluate_files/{idWork}/{idStudent} [post]
func (w *WorkController) UploadReEvaluateInperson(c *gin.Context) {
	var evaluate *forms.EvaluateInperson
	idWork := c.Param("idWork")
	idStudent := c.Param("idStudent")
	claims, _ := services.NewClaimsFromContext(c)
	if err := c.BindJSON(&evaluate); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Upload
	err := workService.UploadEvaluateInperson(evaluate, idWork, claims.ID, idStudent, true)
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

// UpdateWork godoc
// @Summary Update work
// @Desc    Update work
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idWork path     string               true "MongoID"
// @Param   work   body     forms.UpdateWorkForm true "Desc"
// @Success 200    {object} res.Response{}
// @Failure 400    {object} res.Response{} "Bad path param"
// @Failure 400    {object} res.Response{} "Este formulario está eliminado"
// @Failure 400    {object} res.Response{} "Un trabajo evaluado no puede tener un formulario sin puntaje"
// @Failure 400    {object} res.Response{} "La fecha y hora de inicio es mayor a la limite"
// @Failure 400    {object} res.Response{} "La fecha y hora de inicio es mayor a la limite registrada"
// @Failure 400    {object} res.Response{} "La fecha y hora de inicio registrada es mayor a la limite"
// @Failure 401    {object} res.Response{} "Unauthorized"
// @Failure 401    {object} res.Response{} "Este formulario no te pertenece"
// @Failure 401    {object} res.Response{} "Ya no se puede editar este trabajo"
// @Failure 401    {object} res.Response{} "Unauthorized role"
// @Failure 404    {object} res.Response{} "No se puede actualizar un item que no está registrado"
// @Failure 503    {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/update_work/{idWork} [put]
func (w *WorkController) UpdateWork(c *gin.Context) {
	var work *forms.UpdateWorkForm
	idWork := c.Param("idWork")
	claims, _ := services.NewClaimsFromContext(c)
	if err := c.BindJSON(&work); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Update work
	err := workService.UpdateWork(work, idWork, claims.ID)
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

// GradeForm godoc
// @Summary Grade form
// @Desc    Grade form
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idWork path     string true "MongoID"
// @Success 200    {object} res.Response{}
// @Failure 400    {object} res.Response{} "Bad path param"
// @Failure 400    {object} res.Response{} "Este formulario no puede ser calificado, el formulario no tiene puntos"
// @Failure 400    {object} res.Response{} "El trabajo no es de tipo formulario"
// @Failure 400    {object} res.Response{} "No existen alumnos a evaluar en este trabajo"
// @Failure 401    {object} res.Response{} "Este formulario todavía no se puede calificar"
// @Failure 401    {object} res.Response{} "Unauthorized"
// @Failure 401    {object} res.Response{} "Unauthorized role"
// @Failure 403    {object} res.Response{} "Este trabajo ya está evaluado"
// @Failure 503    {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/grade_form/{idWork} [post]
func (w *WorkController) GradeForm(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	idWork := c.Param("idWork")
	// Grade
	err := workService.GradeWork(idWork, claims.ID, "form")
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

// GradeFiles godoc
// @Summary Grade files
// @Desc    Grade files
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idWork path     string true "MongoID"
// @Success 200    {object} res.Response{}
// @Failure 400    {object} res.Response{} "Bad path param"
// @Failure 400    {object} res.Response{} "El trabajo no es de tipo archivos"
// @Failure 400    {object} res.Response{} "No existen alumnos a evaluar en este trabajo"
// @Failure 401    {object} res.Response{} "Este trabajo todavía no se puede calificar"
// @Failure 401    {object} res.Response{} "Unauthorized"
// @Failure 401    {object} res.Response{} "Unauthorized role"
// @Failure 403    {object} res.Response{} "Este trabajo ya está evaluado"
// @Failure 503    {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/grade_files/{idWork} [post]
func (w *WorkController) GradeFiles(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	idWork := c.Param("idWork")
	// Grade
	err := workService.GradeWork(idWork, claims.ID, "files")
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

// GradeInperson godoc
// @Summary Grade inperson work
// @Desc    Grade inperson work
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idWork path     string true "MongoID"
// @Success 200    {object} res.Response{}
// @Failure 400    {object} res.Response{} "Bad path param"
// @Failure 400    {object} res.Response{} "El trabajo no es de tipo presencial"
// @Failure 400    {object} res.Response{} "No existen alumnos a evaluar en este trabajo"
// @Failure 401    {object} res.Response{} "Este trabajo todavía no se puede calificar"
// @Failure 401    {object} res.Response{} "Unauthorized"
// @Failure 401    {object} res.Response{} "Unauthorized role"
// @Failure 403    {object} res.Response{} "Este trabajo ya está evaluado"
// @Failure 503    {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/grade_inperson/{idWork} [post]
func (*WorkController) GradeInperson(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	idWork := c.Param("idWork")
	// Grade
	err := workService.GradeWork(idWork, claims.ID, "in-person")
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

// DeleteWork godoc
// @Summary Delete work
// @Desc    Delete work
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idWork path     string true "MongoID"
// @Success 200    {object} res.Response{}
// @Failure 400    {object} res.Response{} "Bad path param"
// @Failure 400    {object} res.Response{} "No se puede eliminar un trabajo calificado"
// @Failure 401    {object} res.Response{} "Unauthorized"
// @Failure 401    {object} res.Response{} "Unauthorized role"
// @Failure 503    {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/delete_work/{idWork} [delete]
func (w *WorkController) DeleteWork(c *gin.Context) {
	idWork := c.Param("idWork")
	// Delete work
	err := workService.DeleteWork(idWork)
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

// DeleteFileClassroom godoc
// @Summary Delete file classroom
// @Desc    Delete file classroom
// @Tags    works
// @Tags    classroom
// @Tags    roles.student
// @Tags    roles.student_directive
// @Accept  json
// @Produce json
// @Param   idWork path     string true "MongoID"
// @Param   idFile path     string true "MongoID"
// @Success 200    {object} res.Response{}
// @Failure 400    {object} res.Response{} "Bad path param"
// @Failure 400    {object} res.Response{} "No se puede eliminar un trabajo calificado"
// @Failure 401    {object} res.Response{} "Unauthorized"
// @Failure 401    {object} res.Response{} "Unauthorized role"
// @Failure 404    {object} res.Response{} "No se encontró el archivo a eliminar en este trabajo"
// @Failure 503    {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/delete_file_work/{idWork}/{idFile} [delete]
func (w *WorkController) DeleteFileClassroom(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	idWork := c.Param("idWork")
	idFile := c.Param("idFile")
	// Delete file
	err := workService.DeleteFileClassroom(idWork, idFile, claims.ID)
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

// DeleteAttached godoc
// @Summary Delete Attached
// @Desc    Delete Attached
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idWork     path     string true "MongoID"
// @Param   idAttached path     string true "MongoID"
// @Success 200        {object} res.Response{}
// @Failure 400        {object} res.Response{} "Bad path param"
// @Failure 401        {object} res.Response{} "Ya no se puede editar este trabajo"
// @Failure 401        {object} res.Response{} "Unauthorized"
// @Failure 401        {object} res.Response{} "Unauthorized role"
// @Failure 404        {object} res.Response{} "No existe este elemento adjunto al trabajo"
// @Failure 503        {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/delete_attached/{idWork}/{idAttached} [delete]
func (w *WorkController) DeleteAttached(c *gin.Context) {
	idWork := c.Param("idWork")
	idAttached := c.Param("idAttached")
	// Delete
	err := workService.DeleteAttached(idWork, idAttached)
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

// DeleteItemPattern godoc
// @Summary Delete Item Pattern
// @Desc    Delete Item Pattern
// @Tags    works
// @Tags    classroom
// @Tags    roles.teacher
// @Accept  json
// @Produce json
// @Param   idWork path     string true "MongoID"
// @Param   idItem path     string true "MongoID"
// @Success 200    {object} res.Response{}
// @Failure 400    {object} res.Response{} "Este no es un trabajo de archivos"
// @Failure 400    {object} res.Response{} "Bad path param"
// @Failure 401    {object} res.Response{} "Unauthorized"
// @Failure 401    {object} res.Response{} "Este trabajo ya no se puede editar"
// @Failure 401    {object} res.Response{} "Unauthorized role"
// @Failure 404    {object} res.Response{} "No existe el item a eliminar"
// @Failure 503    {object} res.Response{} "Service Unavailable - NATS || DB Service Unavailable"
// @Router  /works/delete_item_pattern/{idWork}/{idItem} [delete]
func (w *WorkController) DeleteItemPattern(c *gin.Context) {
	idWork := c.Param("idWork")
	idItem := c.Param("idItem")
	// Delete
	err := workService.DeleteItemPattern(idWork, idItem)
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
