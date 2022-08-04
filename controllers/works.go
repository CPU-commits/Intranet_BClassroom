package controllers

import (
	"io"
	"net/http"

	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
)

// Services
var workService = services.NewWorksService()

type WorkController struct{}

// Query
func (w *WorkController) GetModulesWorks(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	// Get
	works, err := workService.GetModulesWorks(*claims)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
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

func (w *WorkController) GetWorks(c *gin.Context) {
	idModule := c.Param("idModule")
	// Get
	works, err := workService.GetWorks(idModule)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
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

func (w *WorkController) GetWork(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	idWork := c.Param("idWork")
	response, err := workService.GetWork(idWork, claims)
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

func (w *WorkController) GetForm(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	idWork := c.Param("idWork")
	// Get
	response, err := workService.GetForm(idWork, claims.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	// Response
	c.JSON(200, &res.Response{
		Success: true,
		Data:    response,
	})
}

func (w *WorkController) GetFormStudent(c *gin.Context) {
	idWork := c.Param("idWork")
	idStudent := c.Param("idStudent")
	// Get
	form, answers, err := workService.GetFormStudent(idWork, idStudent)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
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

func (w *WorkController) GetStudentsStatus(c *gin.Context) {
	idModule := c.Param("idModule")
	idWork := c.Param("idWork")

	// Get
	students, totalPoints, err := workService.GetStudentsStatus(idModule, idWork)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
			Success: false,
			Message: err.Error(),
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
			c.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
				Success: false,
				Message: err.Error(),
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

// Feed
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
	err = workService.UploadFiles(files, idWork, claims.ID)
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
	workService.FinishForm(answers, idWork, claims.ID)
	c.JSON(200, &res.Response{
		Success: true,
	})
}

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

func (w *WorkController) GradeForm(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	idWork := c.Param("idWork")
	// Grade
	err := workService.GradeForm(idWork, claims.ID)
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

func (w *WorkController) GradeFiles(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	idWork := c.Param("idWork")
	// Grade
	err := workService.GradeFiles(idWork, claims.ID)
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

func (w *WorkController) DeleteWork(c *gin.Context) {
	idWork := c.Param("idWork")
	// Delete work
	err := workService.DeleteWork(idWork)
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

func (w *WorkController) DeleteFileClassroom(c *gin.Context) {
	claims, _ := services.NewClaimsFromContext(c)
	idWork := c.Param("idWork")
	idFile := c.Param("idFile")
	// Delete file
	err := workService.DeleteFileClassroom(idWork, idFile, claims.ID)
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

func (w *WorkController) DeleteAttached(c *gin.Context) {
	idWork := c.Param("idWork")
	idAttached := c.Param("idAttached")
	// Delete
	err := workService.DeleteAttached(idWork, idAttached)
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

func (w *WorkController) DeleteItemPattern(c *gin.Context) {
	idWork := c.Param("idWork")
	idItem := c.Param("idItem")
	// Delete
	err := workService.DeleteItemPattern(idWork, idItem)
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
