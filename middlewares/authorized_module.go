package middlewares

import (
	"fmt"
	"net/http"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var workModel = models.NewWorkModel()
var moduleHistoryModel = models.NewModuleHistoryModel()

func FindIfIsHistoryModule(idModule, idStudent primitive.ObjectID) error {
	var moduleHistory *models.ModuleHistory

	cursor := moduleHistoryModel.GetOne(bson.D{
		{
			Key:   "module",
			Value: idModule,
		},
		{
			Key: "students",
			Value: bson.M{
				"$in": bson.A{idStudent},
			},
		},
	})
	if err := cursor.Decode(&moduleHistory); err != nil {
		return err
	}
	return nil
}

func AuthorizedRouteModule() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		claims, _ := services.NewClaimsFromContext(ctx)
		if claims.UserType == models.DIRECTOR || claims.UserType == models.DIRECTIVE {
			ctx.Next()
			return
		}

		idModule := ctx.Param("idModule")
		if idModule == "" && ctx.Param("idWork") != "" {
			idObjWork, err := primitive.ObjectIDFromHex(ctx.Param("idWork"))
			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
					Success: false,
					Message: err.Error(),
				})
				return
			}

			var work *models.Work
			cursor := workModel.GetByID(idObjWork)
			if err := cursor.Decode(&work); err != nil {
				ctx.AbortWithStatusJSON(http.StatusNotFound, &res.Response{
					Success: false,
					Message: fmt.Sprintf("No existe el trabajo indicado"),
				})
				return
			}
			idModule = work.Module.Hex()
		}

		authorized := services.AuthorizedRouteFromIdModule(idModule, claims)
		if authorized != nil {
			if claims.UserType == models.STUDENT || claims.UserType == models.STUDENT_DIRECTIVE {
				idObjModule, err := primitive.ObjectIDFromHex(idModule)
				if err != nil {
					ctx.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
						Message: err.Error(),
					})
					return
				}
				idObjStudent, err := primitive.ObjectIDFromHex(claims.ID)
				if err != nil {
					ctx.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
						Message: err.Error(),
					})
					return
				}

				err = FindIfIsHistoryModule(idObjModule, idObjStudent)
				if err == nil {
					ctx.Next()
					return
				} else {
					var message string
					var status int

					if err.Error() == db.NO_SINGLE_DOCUMENT {
						status = http.StatusUnauthorized
						message = "No tienes acceso a esta ruta"
					} else {
						status = http.StatusServiceUnavailable
						message = "Service Unavailable"
					}
					ctx.AbortWithStatusJSON(status, &res.Response{
						Message: message,
					})
					return
				}
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, &res.Response{
				Success: false,
				Message: authorized.Error(),
			})
			return
		}
		ctx.Next()
	}
}
