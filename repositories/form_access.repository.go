package repositories

import (
	"net/http"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"go.mongodb.org/mongo-driver/bson"
)

type FormAccessRepository struct{}

func (f *FormAccessRepository) GetFormAccess(filters bson.D) (*models.FormAccess, *res.ErrorRes) {
	var formAccess *models.FormAccess

	cursor := formAccessModel.GetOne(filters)
	if err := cursor.Decode(&formAccess); err != nil && err.Error() != db.NO_SINGLE_DOCUMENT {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	return formAccess, nil
}

func NewFormAccessRepository() *FormAccessRepository {
	return &FormAccessRepository{}
}
