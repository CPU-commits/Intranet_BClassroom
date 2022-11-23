package smaps

import (
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/services"
)

type FormsMap struct {
	Forms []models.Form `json:"forms"`
}

type FormMap struct {
	Form []models.FormWLookup `json:"form"`
}

type ProgramGradeMap struct {
	Programs []models.GradesProgram `json:"programs"`
}

type StudentsGradesMap struct {
	Students []services.StudentGrade `json:"students"`
}

type StudentGradesMap struct {
	Grades []*services.OrderedGrade `json:"grades"`
}

type IdInsertedMap struct {
	ID string `json:"_id"`
}

type InsertedIdMap struct {
	ID string `json:"inserted_id"`
}

type ModuleMap struct {
	Module *models.ModuleWithLookup `json:"module"`
}

type ModulesMap struct {
	Modules []models.ModuleWithLookup `json:"modules"`
}

type ModulesHistoryMap struct {
	Modules []models.ModuleWithLookup `json:"modules"`
	Total   int                       `json:"total"`
}

type SearchHitsMap struct {
	Hits interface{} `json:"hits"`
}

type TokenMap struct {
	Token string `json:"token"`
}

type PublicationsMap struct {
	Publications []*services.PublicationsRes `json:"publications"`
	Total        int64                       `json:"total"`
}

type PublicationMap struct {
	Publication *services.PublicationsRes `json:"publication"`
}

type NewPublicationMap struct {
	ID          string   `json:"_id"`
	AttachedIds []string `json:"attached_ids"`
}

type ModulesWorksMap struct {
	Works []services.WorkStatus `json:"works"`
}

type WorksMap struct {
	Works []models.WorkWLookup `json:"works"`
}

type WorkMap struct {
	Work          *models.WorkWLookupNFiles           `json:"work"`
	Grade         models.GradeWLookup                 `json:"grade"`
	FormHasPoints bool                                `json:"form_has_points"`
	FormAccess    *models.FormAccess                  `json:"form_access" extensions:"x-student"`
	FileUploaded  models.FileUploadedClassroomWLookup `json:"files_uploaded" extensions:"x-student"`
}

type FormWorkMap struct {
	Form    models.FormWLookup    `json:"form"`
	Answers []*services.AnswerRes `json:"answers"`
	Work    struct {
		Wtime     bool      `json:"wtime"`
		DateLimit time.Time `json:"date_limit"`
		Status    string    `json:"status"`
		Points    struct {
			MaxPoints   int `json:"max_points"`
			TotalPoints int `json:"total_points"`
		} `json:"points"`
	} `json:"work"`
}

type FormStudentMap struct {
	Form    *models.FormWLookup  `json:"form"`
	Answers []services.AnswerRes `json:"answers"`
}

type StudentsStatusMap struct {
	Students    []services.Student `json:"students"`
	TotalPoints int                `json:"total_points"`
}
