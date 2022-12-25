package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/middlewares"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	controllers_query "github.com/CPU-commits/Intranet_BClassroom/query/controllers"
	"github.com/CPU-commits/Intranet_BClassroom/query/docs"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/settings"
	ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/secure"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

func keyFunc(c *gin.Context) string {
	return c.ClientIP()
}

func ErrorHandler(c *gin.Context, info ratelimit.Info) {
	c.JSON(http.StatusTooManyRequests, &res.Response{
		Success: false,
		Message: "Too many requests. Try again in" + time.Until(info.ResetTime).String(),
	})
}

var settingsData = settings.GetSettings()

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("No .env file found")
	}
}

func Init() {
	router := gin.New()
	// Proxies
	router.SetTrustedProxies([]string{"localhost"})
	// Zap looger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	router.Use(ginzap.GinzapWithConfig(logger, &ginzap.Config{
		TimeFormat: time.RFC3339,
		UTC:        true,
		SkipPaths:  []string{"/api/c/classroom/swagger"},
	}))
	router.Use(ginzap.RecoveryWithZap(logger, true))

	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.String(http.StatusInternalServerError, fmt.Sprintf("Server Internal Error: %s", err))
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, res.Response{
			Success: false,
			Message: "Server Internal Error",
		})
	}))
	// Docs
	docs.SwaggerInfo.BasePath = "/api/c/classroom"
	docs.SwaggerInfo.Version = "v1"
	docs.SwaggerInfo.Host = "localhost:8080"
	// CORS
	httpOrigin := "http://" + settingsData.CLIENT_URL
	httpsOrigin := "https://" + settingsData.CLIENT_URL
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{httpOrigin, httpsOrigin},
		AllowMethods:     []string{"GET", "OPTIONS", "PUT", "DELETE", "POST"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
		AllowWebSockets:  false,
		MaxAge:           12 * time.Hour,
	}))
	// Secure
	sslUrl := "ssl." + settingsData.CLIENT_URL
	secureConfig := secure.Config{
		SSLHost:              sslUrl,
		STSSeconds:           315360000,
		STSIncludeSubdomains: true,
		FrameDeny:            true,
		ContentTypeNosniff:   true,
		BrowserXssFilter:     true,
		IENoOpen:             true,
		ReferrerPolicy:       "strict-origin-when-cross-origin",
		SSLProxyHeaders: map[string]string{
			"X-Fowarded-Proto": "https",
		},
	}
	/*if settingsData.NODE_ENV == "prod" {
		secureConfig.AllowedHosts = []string{
			settingsData.CLIENT_URL,
			sslUrl,
		}
	}*/
	router.Use(secure.New(secureConfig))
	// Rate limit
	store := ratelimit.InMemoryStore(&ratelimit.InMemoryOptions{
		Rate:  time.Second,
		Limit: 7,
	})
	mw := ratelimit.RateLimiter(store, &ratelimit.Options{
		ErrorHandler: ErrorHandler,
		KeyFunc:      keyFunc,
	})
	router.Use(mw)
	// Routes
	defaultRoles := []string{
		models.STUDENT,
		models.STUDENT_DIRECTIVE,
		models.TEACHER,
	}
	modules := router.Group(
		"/api/c/classroom/modules",
		middlewares.JWTMiddleware(),
		middlewares.RolesMiddleware(defaultRoles),
	)
	publications := router.Group(
		"/api/c/classroom/publications",
		middlewares.JWTMiddleware(),
		middlewares.RolesMiddleware(defaultRoles),
	)
	forms := router.Group(
		"/api/c/classroom/forms",
		middlewares.JWTMiddleware(),
		middlewares.RolesMiddleware(defaultRoles),
	)
	grade := router.Group(
		"/api/c/classroom/grades",
		middlewares.JWTMiddleware(),
	)
	work := router.Group(
		"/api/c/classroom/works",
		middlewares.JWTMiddleware(),
	)
	{
		// Init controllers
		modulesController := new(controllers_query.ModulesController)
		publicationsController := new(controllers_query.PublicationController)
		formsController := new(controllers_query.FormController)
		gradesController := new(controllers_query.GradesController)
		worksController := new(controllers_query.WorkController)
		// Define routes
		// Modules
		modules.GET(
			"/get_module/:idModule",
			middlewares.AuthorizedRouteModule(),
			modulesController.GetModule,
		)
		modules.GET("/get_modules", modulesController.GetModules)
		modules.GET("/get_modules_history", modulesController.GetModulesHistory)
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
			middlewares.RolesMiddleware([]string{
				models.TEACHER,
				models.STUDENT,
				models.STUDENT_DIRECTIVE,
				models.DIRECTIVE,
				models.DIRECTOR,
			}),
			middlewares.AuthorizedRouteModule(),
			gradesController.GetProgramGrade,
		)
		grade.GET(
			"/get_students_grades/:idModule",
			middlewares.RolesMiddleware([]string{
				models.TEACHER,
				models.DIRECTOR,
				models.DIRECTIVE,
			}),
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
			middlewares.RolesMiddleware([]string{
				models.TEACHER,
				models.DIRECTIVE,
				models.DIRECTOR,
			}),
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
	// Route docs
	router.GET("/api/c/classroom/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// Route healthz
	router.GET("/api/c/classroom/healthz", func(ctx *gin.Context) {
		ctx.JSON(200, &res.Response{
			Success: true,
		})
	})
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
