package main

import (
	"github.com/CPU-commits/Intranet_BClassroom/feed/server"
)

// @title          Classroom Feed API
// @version        1.0
// @description    API Server For feed requests of classroom service
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
func main() {
	server.Init()
}
