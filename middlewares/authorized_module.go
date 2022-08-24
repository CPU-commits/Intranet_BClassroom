package middlewares

import (
	"fmt"
	"net/http"

	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var workModel = models.NewWorkModel()

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
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, &res.Response{
				Success: false,
				Message: authorized.Error(),
			})
			return
		}
		ctx.Next()
	}
}
