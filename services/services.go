package services

import (
	"encoding/json"

	"github.com/CPU-commits/Intranet_BClassroom/aws_s3"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/repositories"
	"github.com/CPU-commits/Intranet_BClassroom/settings"
	"github.com/CPU-commits/Intranet_BClassroom/stack"
	"github.com/google/uuid"
)

// Models
var (
	workModel             = models.NewWorkModel()
	averageModel          = models.NewAveragesModel()
	formAccessModel       = models.NewFormAccessModel()
	answerModel           = models.NewAnswerModel()
	evaluatedAnswersModel = models.NewEvaluatedAnswersModel()
	gradeModel            = models.NewGradesModel()
	workGradeModel        = models.NewWorkGradesModel()
	teacherModel          = models.NewTeacherModel()
	parentModel           = models.NewParentsModel()
	studentModel          = models.NewStudentModel()
	formModel             = models.NewFormModel()
	formQuestionModel     = models.NewFormQuestionModel()
	gradeProgramModel     = models.NewGradesProgramModel()
	moduleModel           = models.NewModuleModel()
	moduleHistoryModel    = models.NewModuleHistoryModel()
	publicationModel      = models.NewPublicationModel()
	fileModel             = models.NewFileModel()
	fileUCModel           = models.NewFileUCModel()
	sessionModel          = models.NewSessionModel()
)

// Repositories
var (
	workRepository = repositories.NewWorkRepository()
)

// Packages
var nats = stack.NewNats()
var aws = aws_s3.NewAWSS3()

// Settings
var settingsData = settings.GetSettings()

func formatRequestToNestjsNats(data interface{}) ([]byte, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	request := make(map[string]interface{})
	request["id"] = id.String()
	if data != nil {
		request["data"] = data
	}
	jsonMarshal, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	return jsonMarshal, nil
}
