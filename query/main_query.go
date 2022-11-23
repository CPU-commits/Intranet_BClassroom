package main

import "github.com/CPU-commits/Intranet_BClassroom/query/server"

// @title          Classroom API
// @version        1.0
// @description    API Server Classroom service
// @termsOfService http://swagger.io/terms/

// @contact.name  API Support
// @contact.url   http://www.swagger.io/support
// @contact.email support@swagger.io

// lincense.name  Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @tag.name        classroom
// @tag.description Service of classroom

// @host     localhost:8080
// @BasePath /api/c/classroom

// @securityDefinitions.apikey ApiKeyAuth
// @in                         header
// @name                       Authorization
// @description                BearerJWTToken in Authorization Header

// @accept  json
// @produce json
// @product octet-stream
// @product application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @product application/pdf

// @schemes http https
func main() {
	server.Init()
}
