package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	controllers_feed "github.com/CPU-commits/Intranet_BClassroom/feed/controllers"
	"github.com/CPU-commits/Intranet_BClassroom/middlewares"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/settings"
	ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/secure"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
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
	// Log file
	logEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	fileCore := zapcore.NewCore(logEncoder, zapcore.AddSync(&lumberjack.Logger{
		Filename:   "logs/app_feed.log",
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     7,
	}), zap.InfoLevel)
	// Log console
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig())
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.InfoLevel)
	// Combine cores for multi-output logging
	teeCore := zapcore.NewTee(fileCore, consoleCore)
	zapLogger := zap.New(teeCore)

	router.Use(ginzap.GinzapWithConfig(zapLogger, &ginzap.Config{
		TimeFormat: time.RFC3339,
		UTC:        true,
		SkipPaths:  []string{"/api/annoucements/swagger"},
	}))
	router.Use(ginzap.RecoveryWithZap(zapLogger, true))

	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.String(http.StatusInternalServerError, fmt.Sprintf("Server Internal Error: %s", err))
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, res.Response{
			Success: false,
			Message: "Server Internal Error",
		})
	}))
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
		middlewares.MaxSizePerFile(MAX_FILE_SIZE, MAX_FILE_SIZE_STR, MAX_FILES, "files[]"),
	)
	{
		// Init controllers
		publicationController := new(controllers_feed.PublicationController)
		moduleController := new(controllers_feed.ModulesController)
		formController := new(controllers_feed.FormController)
		gradesController := new(controllers_feed.GradesController)
		worksController := new(controllers_feed.WorkController)
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
