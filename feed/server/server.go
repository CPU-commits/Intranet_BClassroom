package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CPU-commits/Intranet_BClassroom/controllers"
	"github.com/CPU-commits/Intranet_BClassroom/middlewares"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("No .env file found")
	}
}

func Init() {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.String(http.StatusInternalServerError, fmt.Sprintf("Server Internal Error: %s", err))
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, res.Response{
			Success: false,
			Message: "Server Internal Error",
		})
	}))
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"*"},
		AllowHeaders: []string{"*"},
	}))
	// Validators
	InitValidators()
	// Routes
	teacherRol := []string{models.TEACHER}
	studentRol := []string{models.STUDENT, models.STUDENT_DIRECTIVE}
	module := router.Group(
		"/api/c/classroom/modules",
		middlewares.JWTMiddleware(),
		middlewares.RolesMiddleware(teacherRol),
	)
	publication := router.Group(
		"/api/c/classroom/publications",
		middlewares.JWTMiddleware(),
		middlewares.RolesMiddleware(teacherRol),
	)
	form := router.Group(
		"/api/c/classroom/forms",
		middlewares.JWTMiddleware(),
		middlewares.RolesMiddleware(teacherRol),
	)
	grade := router.Group(
		"/api/c/classroom/grades",
		middlewares.JWTMiddleware(),
		middlewares.RolesMiddleware(teacherRol),
	)
	work := router.Group(
		"/api/c/classroom/works",
		middlewares.JWTMiddleware(),
	)
	{
		// Init controllers
		publicationController := new(controllers.PublicationController)
		moduleController := new(controllers.ModulesController)
		formController := new(controllers.FormController)
		gradesController := new(controllers.GradesController)
		worksController := new(controllers.WorkController)
		// Define routes
		// Module
		module.POST(
			"/new_sub_section/:idModule",
			middlewares.AuthorizedRouteModule(),
			moduleController.NewSubSection,
		)
		// Publication
		publication.POST(
			"/upload/:idModule",
			middlewares.AuthorizedRouteModule(),
			publicationController.NewPublication,
		)
		publication.PUT("/update/:idPublication", publicationController.UpdatePublication)
		publication.DELETE(
			"/delete/:idPublication/:idModule",
			middlewares.AuthorizedRouteModule(),
			publicationController.DeletePublication,
		)
		publication.DELETE(
			"/delete_attached/:idAttached/:idModule",
			middlewares.AuthorizedRouteModule(),
			publicationController.DeletePublicationAttached,
		)
		// Form
		form.POST("/upload_form", formController.UploadForm)
		form.PUT("/update_form/:idForm", formController.UpdateForm)
		form.DELETE("/delete_form/:idForm", formController.DeleteForm)
		// Grades
		grade.POST(
			"/upload_program/:idModule",
			middlewares.AuthorizedRouteModule(),
			gradesController.UploadProgramGrade,
		)
		grade.POST(
			"/upload_grade/:idModule/:idStudent",
			middlewares.AuthorizedRouteModule(),
			gradesController.UploadGrade,
		)
		grade.PUT(
			"/update_grade/:idModule/:idGrade",
			middlewares.AuthorizedRouteModule(),
			gradesController.UpdateGrade,
		)
		grade.DELETE(
			"/delete_program/:idModule/:idProgram",
			middlewares.AuthorizedRouteModule(),
			gradesController.DeleteGradeProgram,
		)
		// Works
		work.POST(
			"/upload_work/:idModule",
			middlewares.RolesMiddleware(teacherRol),
			middlewares.AuthorizedRouteModule(),
			worksController.UploadWork,
		)
		work.POST(
			"/save_answer/:idWork/:idQuestion",
			middlewares.RolesMiddleware(studentRol),
			middlewares.AuthorizedRouteModule(),
			worksController.SaveAnswer,
		)
		work.POST(
			"/upload_files/:idWork",
			middlewares.RolesMiddleware(studentRol),
			middlewares.AuthorizedRouteModule(),
			worksController.UploadFiles,
		)
		work.POST(
			"/finish_form/:idWork",
			middlewares.RolesMiddleware(studentRol),
			middlewares.AuthorizedRouteModule(),
			worksController.FinishForm,
		)
		work.POST(
			"/upload_points_question/:idWork/:idQuestion/:idStudent",
			middlewares.RolesMiddleware(teacherRol),
			middlewares.AuthorizedRouteModule(),
			worksController.UploadPointsQuestion,
		)
		work.POST(
			"/grade_form/:idWork",
			middlewares.RolesMiddleware(teacherRol),
			middlewares.AuthorizedRouteModule(),
			worksController.GradeForm,
		)
		work.POST(
			"/grade_files/:idWork",
			middlewares.RolesMiddleware(teacherRol),
			middlewares.AuthorizedRouteModule(),
			worksController.GradeFiles,
		)
		work.POST(
			"/upload_evaluate_files/:idWork/:idStudent",
			middlewares.RolesMiddleware(teacherRol),
			middlewares.AuthorizedRouteModule(),
			worksController.UploadEvaluateFiles,
		)
		work.POST(
			"/upload_reevaluate_files/:idWork/:idStudent",
			middlewares.RolesMiddleware(teacherRol),
			middlewares.AuthorizedRouteModule(),
			worksController.UploadReEvaluateFiles,
		)
		work.PUT(
			"/update_work/:idWork",
			middlewares.RolesMiddleware(teacherRol),
			middlewares.AuthorizedRouteModule(),
			worksController.UpdateWork,
		)
		work.DELETE(
			"/delete_work/:idWork",
			middlewares.RolesMiddleware(teacherRol),
			middlewares.AuthorizedRouteModule(),
			worksController.DeleteWork,
		)
		work.DELETE(
			"/delete_file_work/:idWork/:idFile",
			middlewares.RolesMiddleware(studentRol),
			middlewares.AuthorizedRouteModule(),
			worksController.DeleteFileClassroom,
		)
		work.DELETE(
			"/delete_attached/:idWork/:idAttached",
			middlewares.RolesMiddleware(teacherRol),
			middlewares.AuthorizedRouteModule(),
			worksController.DeleteAttached,
		)
		work.DELETE(
			"/delete_item_pattern/:idWork/:idItem",
			middlewares.RolesMiddleware(teacherRol),
			middlewares.AuthorizedRouteModule(),
			worksController.DeleteItemPattern,
		)
	}
	// No route
	router.NoRoute(func(ctx *gin.Context) {
		ctx.JSON(404, res.Response{
			Success: false,
			Message: "Not found",
		})
	})
	// Init server
	if err := router.Run(); err != nil {
		log.Fatalf("Error init server")
	}
}
