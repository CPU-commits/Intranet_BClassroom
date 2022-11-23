package services

import (
	"fmt"
	"net/http"
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var formService *FormService

type FormService struct{}

func (f *FormService) getLookupQuestions() bson.D {
	return bson.D{
		{
			Key: "$lookup",
			Value: bson.M{
				"from":         models.FORM_QUESTION_COLLECTION,
				"localField":   "items.questions",
				"foreignField": "_id",
				"as":           "items.questions",
			},
		},
	}
}

func (f *FormService) GetFormsUser(userId string) ([]models.Form, *res.ErrorRes) {
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get
	match := bson.D{
		{
			Key: "$match",
			Value: bson.M{
				"author": userObjId,
				"status": true,
			},
		},
	}
	sort := bson.D{
		{
			Key: "$sort",
			Value: bson.M{
				"upload_date": -1,
			},
		},
	}
	project := bson.D{
		{
			Key: "$project",
			Value: bson.M{
				"items": 0,
			},
		},
	}

	var forms []models.Form
	cursor, err := formModel.Aggreagate(mongo.Pipeline{
		match,
		sort,
		project,
	})
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if err = cursor.All(db.Ctx, &forms); err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}

	return forms, nil
}

func (f *FormService) GetFormById(idForm primitive.ObjectID) (*models.Form, error) {
	var form *models.Form
	cursor := formModel.GetByID(idForm)
	if err := cursor.Decode(&form); err != nil {
		return nil, err
	}
	return form, nil
}

func (f *FormService) GetForm(idForm, idUser string, onlyAuthor bool) ([]models.FormWLookup, *res.ErrorRes) {
	idFormObj, err := primitive.ObjectIDFromHex(idForm)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjUser, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get
	match := bson.D{}
	if onlyAuthor {
		match = append(match, primitive.E{
			Key: "$match",
			Value: bson.M{
				"_id":    idFormObj,
				"author": idObjUser,
			},
		})
	} else {
		match = append(match, primitive.E{
			Key: "$match",
			Value: bson.M{
				"_id": idFormObj,
			},
		})
	}
	unwindItems := bson.D{
		{
			Key:   "$unwind",
			Value: bson.M{"path": "$items"},
		},
	}
	group := bson.D{
		{
			Key: "$group",
			Value: bson.M{
				"_id": "$_id",
				"items": bson.M{
					"$push": "$items",
				},
			},
		},
	}
	lookupRoot := bson.D{
		{
			Key: "$lookup",
			Value: bson.M{
				"from":         models.FORM_COLLECTION,
				"localField":   "_id",
				"foreignField": "_id",
				"as":           "result",
				"pipeline": bson.A{bson.M{
					"$project": bson.M{
						"items":  0,
						"author": 0,
					},
				}},
			},
		},
	}
	unwindResult := bson.D{
		{
			Key: "$unwind",
			Value: bson.M{
				"path": "$result",
			},
		},
	}
	addFields := bson.D{
		{
			Key: "$addFields",
			Value: bson.M{
				"result.items": "$items",
			},
		},
	}
	replaceRoot := bson.D{
		{
			Key: "$replaceRoot",
			Value: bson.M{
				"newRoot": "$result",
			},
		},
	}

	var form []models.FormWLookup
	cursor, err := formModel.Aggreagate(mongo.Pipeline{
		match,
		unwindItems,
		f.getLookupQuestions(),
		group,
		lookupRoot,
		unwindResult,
		addFields,
		replaceRoot,
	})
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if err = cursor.All(db.Ctx, &form); err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	return form, nil
}

func (f *FormService) UploadForm(form *forms.FormForm, userId string) *res.ErrorRes {
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Insert questions
	var questionsIds [][]primitive.ObjectID

	for _, item := range form.Items {
		var individualPoints int

		if item.Type == "equal" {
			all := func(questions []forms.QuestionForm) int {
				count := 0
				for _, question := range questions {
					if question.Type != "alternatives" {
						count += 1
					}
				}
				return count
			}(item.Questions)
			individualPoints = item.Points / all
		}

		var questions []interface{}
		for _, question := range item.Questions {
			if item.Type == "custom" {
				individualPoints = question.Points
			}
			questionData := models.NewModelsFormQuestion(&question, &item, individualPoints)
			questions = append(questions, questionData)
		}
		inserts, err := formQuestionModel.Use().InsertMany(db.Ctx, questions)
		if err != nil {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}

		var ids []primitive.ObjectID
		for _, id := range inserts.InsertedIDs {
			ids = append(ids, id.(primitive.ObjectID))
		}
		questionsIds = append(questionsIds, ids)
	}

	formData := models.NewModelsForm(form, questionsIds, form.PointsType, userObjId)

	_, err = formModel.NewDocument(formData)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	return nil
}

func (f *FormService) UpdateForm(form *forms.FormForm, userId, idForm string) *res.ErrorRes {
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idFormObj, err := primitive.ObjectIDFromHex(idForm)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get form
	var formData *models.Form
	cursor := formModel.GetByID(idFormObj)
	if err := cursor.Decode(&formData); err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if !formData.Status {
		return &res.ErrorRes{
			Err:        fmt.Errorf("Este formulario no está disponible"),
			StatusCode: http.StatusForbidden,
		}
	}
	if formData.Author != userObjId {
		return &res.ErrorRes{
			Err:        fmt.Errorf("No tienes acceso para editar este formulario"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	// Get work
	var work []models.Work
	cursorW, err := workModel.GetAll(bson.D{{
		Key:   "form",
		Value: idFormObj,
	}}, &options.FindOptions{})
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if err := cursorW.All(db.Ctx, &work); err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	for _, work := range work {
		if work.IsRevised {
			return &res.ErrorRes{
				Err:        fmt.Errorf("Este formulario está asignado a un trabajo revisado, no se puede editar"),
				StatusCode: http.StatusForbidden,
			}
		}
	}
	// Update form
	var newItems []models.FormItem

	for _, item := range form.Items {
		newItem := models.FormItem{
			Title:      item.Title,
			PointsType: item.Type,
		}

		idObjItem, err := primitive.ObjectIDFromHex(item.ID)

		if err != nil && item.ID != "" {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusBadRequest,
			}
		}
		var questionsIds []primitive.ObjectID
		// Individual points question
		var individualPoints int
		if item.Type == "equal" {
			all := func(questions []forms.QuestionForm) int {
				count := 0
				for _, question := range questions {
					if question.Type != "alternatives" {
						count += 1
					}
				}
				return count
			}(item.Questions)
			individualPoints = item.Points / all
		}
		// Insert or update
		if idObjItem.IsZero() && item.ID == "" {
			// Insert questions
			var questions []interface{}
			for _, question := range item.Questions {
				if item.Type == "custom" {
					individualPoints = question.Points
				}
				questionData := models.NewModelsFormQuestion(&question, &item, individualPoints)
				questions = append(questions, questionData)
			}
			inserts, err := formQuestionModel.Use().InsertMany(db.Ctx, questions)
			if err != nil {
				return &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
			}

			var ids []primitive.ObjectID
			for _, id := range inserts.InsertedIDs {
				ids = append(ids, id.(primitive.ObjectID))
			}
			questionsIds = ids
		} else {
			newItem.ID = idObjItem
			// Evaluate questions
			for _, question := range item.Questions {
				idObjQuestion, err := primitive.ObjectIDFromHex(question.ID)

				if err != nil && question.ID != "" {
					return &res.ErrorRes{
						Err:        err,
						StatusCode: http.StatusBadRequest,
					}
				}
				// Insert or update
				if item.Type == "custom" {
					individualPoints = question.Points
				}
				if idObjQuestion.IsZero() {
					newId := primitive.NewObjectID()
					questionData := models.ItemQuestion{
						ID:       newId,
						Type:     question.Type,
						Answers:  question.Answers,
						Question: question.Question,
					}
					if question.Type == "alternatives_correct" {
						questionData.Correct = *question.Correct
					}
					if question.Type != "alternatives" {
						questionData.Points = individualPoints
					}
					_, err := formQuestionModel.NewDocument(questionData)
					if err != nil {
						return &res.ErrorRes{
							Err:        err,
							StatusCode: http.StatusServiceUnavailable,
						}
					}

					questionsIds = append(questionsIds, newId)
				} else {
					updateInfo := bson.M{
						"question": question.Question,
						"type":     question.Type,
						"answers":  question.Answers,
					}
					unsetInfo := bson.M{}
					// Set or unset
					if question.Type == "alternatives_correct" {
						updateInfo["correct"] = question.Correct
					} else {
						unsetInfo["correct"] = ""
					}
					if question.Type != "alternatives" && form.PointsType == "true" {
						updateInfo["points"] = individualPoints
					} else {
						unsetInfo["points"] = ""
					}

					_, err := formQuestionModel.Use().UpdateByID(
						db.Ctx,
						idObjQuestion,
						bson.D{
							{
								Key:   "$set",
								Value: updateInfo,
							},
							{
								Key:   "$unset",
								Value: unsetInfo,
							},
						},
					)
					if err != nil {
						return &res.ErrorRes{
							Err:        err,
							StatusCode: http.StatusServiceUnavailable,
						}
					}
					questionsIds = append(questionsIds, idObjQuestion)
				}
			}
		}
		newItem.Questions = questionsIds
		newItems = append(newItems, newItem)
	}

	_, err = formModel.Use().UpdateByID(db.Ctx, idFormObj, bson.D{
		{
			Key: "$set",
			Value: bson.M{
				"title":       form.Title,
				"has_points":  form.PointsType == "true",
				"update_date": primitive.NewDateTimeFromTime(time.Now()),
				"items":       newItems,
			},
		},
	})
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	return nil
}

func (f *FormService) DeleteForm(idForm, idUser string) *res.ErrorRes {
	idObjForm, err := primitive.ObjectIDFromHex(idForm)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idObjUser, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get form
	var form models.Form

	cursor := formModel.GetByID(idObjForm)
	if err := cursor.Decode(&form); err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if !form.Status {
		return &res.ErrorRes{
			Err:        fmt.Errorf("Este formulario ya está eliminado"),
			StatusCode: http.StatusForbidden,
		}
	}
	if form.Author != idObjUser {
		return &res.ErrorRes{
			Err:        fmt.Errorf("No estás autorizado a eliminar este formulario"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	// Delete
	_, err = formModel.Use().UpdateByID(db.Ctx, idObjForm, bson.D{
		{
			Key: "$set",
			Value: bson.M{
				"status": false,
			},
		},
	})
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	return nil
}

func NewFormService() *FormService {
	if formService == nil {
		formService = &FormService{}
	}
	return formService
}
