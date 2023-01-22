package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/gin-gonic/gin"
)

func MaxSizePerFile(maxSize float64, maxSizeStr string, maxFiles int, properties ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if strings.HasPrefix(ctx.Request.Header.Get("Content-Type"), "multipart/form-data") {
			form, err := ctx.MultipartForm()
			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, &res.Response{
					Message: "Body must be a multipart/form-data",
					Success: false,
				})
				return
			}
			// Get file count and maxSize
			countFiles := 0
			for _, property := range properties {
				files := form.File[property]
				// Count files
				lenFiles := len(files)
				countFiles += lenFiles
				// Validate count files
				if countFiles > maxFiles {
					ctx.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, &res.Response{
						Message: fmt.Sprintf("Too many files - Max %v", maxFiles),
						Success: false,
					})
					return
				}
				if lenFiles > 0 {
					for _, file := range files {
						if file.Size > int64(maxSize) {
							ctx.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, &res.Response{
								Message: fmt.Sprintf("File %v too large - Max %v", file.Filename, maxSizeStr),
								Success: false,
							})
							return
						}
					}
				}
			}
			fmt.Printf("countFiles: %v\n", countFiles)
		}
		ctx.Next()
		return
	}
}
