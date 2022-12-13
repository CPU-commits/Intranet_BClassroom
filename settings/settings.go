package settings

import (
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

var lock = &sync.Mutex{}
var singleSettingsInstace *settings

type settings struct {
	JWT_SECRET_KEY      string
	MONGO_DB            string
	MONGO_ROOT_USERNAME string
	MONGO_ROOT_PASSWORD string
	MONGO_HOST          string
	MONGO_CONNECTION    string
	NATS_HOST           string
	AWS_BUCKET          string
	AWS_REGION          string
	ELS_HOST            string
	ELS_PASSWORD        string
	ELS_PORT            int
	ELS_USERNAME        string
	COLLEGE_NAME        string
	CLIENT_URL          string
	NODE_ENV            string
}

func newSettings() *settings {
	elsPort, err := strconv.Atoi(os.Getenv("ELS_PORT"))
	if err != nil {
		panic(err)
	}
	return &settings{
		JWT_SECRET_KEY:      os.Getenv("JWT_SECRET_KEY"),
		MONGO_DB:            os.Getenv("MONGO_DB"),
		MONGO_ROOT_USERNAME: os.Getenv("MONGO_ROOT_USERNAME"),
		MONGO_ROOT_PASSWORD: os.Getenv("MONGO_ROOT_PASSWORD"),
		MONGO_HOST:          os.Getenv("MONGO_HOST"),
		MONGO_CONNECTION:    os.Getenv("MONGO_CONNECTION"),
		NATS_HOST:           os.Getenv("NATS_HOST"),
		ELS_HOST:            os.Getenv("ELS_HOST"),
		ELS_PORT:            elsPort,
		ELS_PASSWORD:        os.Getenv("ELS_PASSWORD"),
		ELS_USERNAME:        os.Getenv("ELS_USERNAME"),
		AWS_BUCKET:          os.Getenv("AWS_BUCKET"),
		AWS_REGION:          os.Getenv("AWS_REGION"),
		COLLEGE_NAME:        os.Getenv("COLLEGE_NAME"),
		CLIENT_URL:          os.Getenv("CLIENT_URL"),
		NODE_ENV:            os.Getenv("NODE_ENV"),
	}
}

func init() {
	if os.Getenv("NODE_ENV") != "prod" {
		if err := godotenv.Load(); err != nil {
			log.Fatalf("No .env file found")
		}
	}
}

func GetSettings() *settings {
	if singleSettingsInstace == nil {
		lock.Lock()
		defer lock.Unlock()
		singleSettingsInstace = newSettings()
	}
	return singleSettingsInstace
}
