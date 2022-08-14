package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

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
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	// Routes
	defaultRoles := []string{
		models.STUDENT,
		models.STUDENT_DIRECTIVE,
		models.TEACHER,
	}
	modules := router.Group(
		"/api/classroom/modules",
		middlewares.JWTMiddleware(),
		middlewares.RolesMiddleware(defaultRoles),
	)
	publications := router.Group(
		"/api/classroom/publications",
		middlewares.JWTMiddleware(),
		middlewares.RolesMiddleware(defaultRoles),
	)
	forms := router.Group(
		"/api/classroom/forms",
		middlewares.JWTMiddleware(),
		middlewares.RolesMiddleware(defaultRoles),
	)
	grade := router.Group(
		"/api/classroom/grades",
		middlewares.JWTMiddleware(),
	)
	work := router.Group(
		"/api/classroom/works",
		middlewares.JWTMiddleware(),
	)
	{
		// Init controllers
		modulesController := new(controllers.ModulesController)
		publicationsController := new(controllers.PublicationController)
		formsController := new(controllers.FormController)
		gradesController := new(controllers.GradesController)
		worksController := new(controllers.WorkController)
		// Define routes
		// Modules
		modules.GET(
			"/get_module/:idModule",
			middlewares.AuthorizedRouteModule(),
			modulesController.GetModule,
		)
		modules.GET("/get_modules", modulesController.GetModules)
		modules.GET(
			"/download_file/:idFile/:idModule",
			middlewares.AuthorizedRouteModule(),
			modulesController.DownloadFile,
		)
		modules.GET(
			"/search/:idModule",
			middlewares.AuthorizedRouteModule(),
			modulesController.Search,
		)
		// Publications
		publications.GET(
			"/get_publications/:idModule",
			middlewares.AuthorizedRouteModule(),
			publicationsController.GetPublications,
		)
		publications.GET(
			"/get_publication/:idModule/:idPublication",
			middlewares.AuthorizedRouteModule(),
			publicationsController.GetPublication,
		)
		// Forms
		forms.GET("/get_forms", formsController.GetForms)
		forms.GET("/get_form/:idForm", formsController.GetForm)
		// Grades
		grade.GET(
			"/get_grade_programs/:idModule",
			middlewares.RolesMiddleware(defaultRoles),
			middlewares.AuthorizedRouteModule(),
			gradesController.GetProgramGrade,
		)
		grade.GET(
			"/get_students_grades/:idModule",
			middlewares.RolesMiddleware([]string{models.TEACHER}),
			middlewares.AuthorizedRouteModule(),
			gradesController.GetStudentsGrades,
		)
		grade.GET(
			"/get_student_grades/:idModule",
			middlewares.RolesMiddleware([]string{models.STUDENT, models.STUDENT_DIRECTIVE}),
			middlewares.AuthorizedRouteModule(),
			gradesController.GetStudentGrades,
		)
		grade.GET(
			"/export_grades/:idModule",
			middlewares.RolesMiddleware([]string{models.TEACHER}),
			gradesController.ExportGrades,
		)
		grade.GET(
			"/download_grades",
			middlewares.RolesMiddleware([]string{models.STUDENT, models.STUDENT_DIRECTIVE}),
			gradesController.ExportGradesStudent,
		)
		// Works
		work.GET(
			"/get_modules_works",
			middlewares.RolesMiddleware([]string{models.STUDENT, models.STUDENT_DIRECTIVE}),
			worksController.GetModulesWorks,
		)
		work.GET(
			"/get_works/:idModule",
			middlewares.RolesMiddleware(defaultRoles),
			middlewares.AuthorizedRouteModule(),
			worksController.GetWorks,
		)
		work.GET(
			"/get_work/:idWork",
			middlewares.RolesMiddleware(defaultRoles),
			middlewares.AuthorizedRouteModule(),
			worksController.GetWork,
		)
		work.GET(
			"/get_form/:idWork",
			middlewares.RolesMiddleware([]string{models.STUDENT, models.STUDENT_DIRECTIVE}),
			middlewares.AuthorizedRouteModule(),
			worksController.GetForm,
		)
		work.GET(
			"/get_students_status/:idModule/:idWork",
			middlewares.RolesMiddleware([]string{models.TEACHER}),
			middlewares.AuthorizedRouteModule(),
			worksController.GetStudentsStatus,
		)
		work.GET(
			"/get_form_student/:idWork/:idStudent",
			middlewares.RolesMiddleware([]string{models.TEACHER}),
			middlewares.AuthorizedRouteModule(),
			worksController.GetFormStudent,
		)
		work.GET(
			"/download_files_work_student/:idWork/:idStudent",
			middlewares.RolesMiddleware([]string{models.TEACHER}),
			middlewares.AuthorizedRouteModule(),
			worksController.DownloadFilesWorkStudent,
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
