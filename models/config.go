package models

import (
	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/settings"
)

var settingsData = settings.GetSettings()

// MongoDB
var DbConnect = db.NewConnection(
	settingsData.MONGO_HOST,
	settingsData.MONGO_DB,
)
