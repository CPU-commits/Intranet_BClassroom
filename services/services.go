package services

import (
	"github.com/CPU-commits/Intranet_BClassroom/aws_s3"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/stack"
)

// Models
var workModel = models.NewWorkModel()
var formAccessModel = models.NewFormAccessModel()
var answerModel = models.NewAnswerModel()
var evaluatedAnswersModel = models.NewEvaluatedAnswersModel()
var gradeModel = models.NewGradesModel()
var workGradeModel = models.NewWorkGradesModel()
var teacherModel = models.NewTeacherModel()
var studentModel = models.NewStudentModel()
var formModel = models.NewFormModel()
var formQuestionModel = models.NewFormQuestionModel()
var gradeProgramModel = models.NewGradesProgramModel()
var moduleModel = models.NewModuleModel()
var publicationModel = models.NewPublicationModel()
var fileModel = models.NewFileModel()
var fileUCModel = models.NewFileUCModel()

// Packages
var nats = stack.NewNats()
var aws = aws_s3.NewAWSS3()
