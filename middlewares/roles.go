package middlewares

import (
	"net/http"

	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/CPU-commits/Intranet_BClassroom/services"
	"github.com/gin-gonic/gin"
)

func RolesMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		claims, _ := services.NewClaimsFromContext(ctx)
		if claims.UserType != models.TEACHER && claims.UserType != models.STUDENT && claims.UserType != models.STUDENT_DIRECTIVE {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, &res.Response{
				Success: false,
				Message: "Unauthorized",
			})
			return
		}
		ctx.Next()
	}
}
