package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/CPU-commits/Intranet_BClassroom/res"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var publicationService *PublicationService

type PublicationService struct{}

func (publication *PublicationService) GetPublicationsFromIdModule(
	idModule,
	section string,
	skip,
	limit int,
	total bool,
) ([]*PublicationsRes, int64, *res.ErrorRes) {
	// Recovery if close channel
	defer func() {
		recovery := recover()
		if recovery != nil {
			fmt.Printf("A channel closed")
		}
	}()

	sectionInt, err := strconv.Atoi(section)
	if err != nil {
		return nil, 0, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get sub section ID
	module, err := moduleService.GetModuleFromID(idModule)
	if err != nil {
		return nil, 0, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if len(module.SubSections) <= sectionInt {
		return nil, 0, &res.ErrorRes{
			Err:        fmt.Errorf("no existe esta sección"),
			StatusCode: http.StatusNotFound,
		}
	}
	// Match
	match := bson.D{
		{
			Key: "$match",
			Value: bson.M{
				"sub_section": module.SubSections[sectionInt].ID,
			},
		},
	}
	// Sort
	sortBson := bson.D{
		{
			Key: "$sort",
			Value: bson.M{
				"upload_date": -1,
			},
		},
	}
	// Skip
	skipBson := bson.D{
		{
			Key:   "$skip",
			Value: skip,
		},
	}
	// Limit
	limitBson := bson.D{
		{
			Key:   "$limit",
			Value: limit,
		},
	}
	// Get publications
	var publications []*models.Publication
	cursor, err := publicationModel.Aggreagate(mongo.Pipeline{match, sortBson, skipBson, limitBson})
	if err != nil {
		return nil, 0, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if err = cursor.All(db.Ctx, &publications); err != nil {
		return nil, 0, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Get publications content
	publicationsRes := make([]*PublicationsRes, len(publications))

	es, err := db.NewConnectionEs()
	if err != nil {
		return nil, 0, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	var errRes res.ErrorRes
	var wg sync.WaitGroup
	c := make(chan (int), 5)

	for i, publication := range publications {
		wg.Add(1)
		c <- 1

		go func(publication *models.Publication, i int, retErr *res.ErrorRes, wg *sync.WaitGroup) {
			defer wg.Done()
			response, err := es.Get(models.PUBLICATIONS_INDEX, publication.ID.Hex())
			if err != nil {
				*retErr = res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusServiceUnavailable,
				}
				close(c)
				return
			}
			// Decode data
			var mapRes map[string]interface{}
			if err := json.NewDecoder(response.Body).Decode(&mapRes); err != nil {
				retErr = &res.ErrorRes{
					Err:        err,
					StatusCode: http.StatusInternalServerError,
				}
				return
			}
			// Get files
			var attacheds []AttachedRes

			for _, attached := range publication.Attached {
				var file *models.File

				if attached.Type == "file" {
					file, err = fileModel.GetFileByID(attached.File)
					if err != nil {
						retErr = &res.ErrorRes{
							Err:        err,
							StatusCode: http.StatusServiceUnavailable,
						}
						return
					}
				}

				attacheds = append(attacheds, AttachedRes{
					ID:    attached.ID.Hex(),
					Type:  attached.Type,
					Link:  attached.Link,
					Title: attached.Title,
					File:  file,
				})
			}
			// Add response
			publicationsRes[i] = &PublicationsRes{
				Attached:   attacheds,
				Content:    mapRes["_source"],
				ID:         publication.ID.Hex(),
				UploadDate: publication.UploadDate,
				UpdateDate: publication.UpdateDate,
			}
			// Close body
			response.Body.Close()

			<-c
		}(publication, i, &errRes, &wg)
	}
	wg.Wait()
	if errRes.Err != nil {
		return nil, 0, &errRes
	}
	// Sort
	sort.Slice(publicationsRes, func(i, j int) bool {
		return publicationsRes[i].UploadDate > publicationsRes[j].UploadDate
	})
	// Get total
	var totalData int64
	if total {
		totalData, err = publicationModel.Use().CountDocuments(db.Ctx, bson.M{})
		if err != nil {
			return nil, 0, &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusServiceUnavailable,
			}
		}
	}
	return publicationsRes, totalData, nil
}

func (p *PublicationService) GetPublication(idModule, idPublication string) (*PublicationsRes, error) {
	idObjModule, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return nil, err
	}
	idObjPublication, err := primitive.ObjectIDFromHex(idPublication)
	if err != nil {
		return nil, err
	}
	// Match
	match := bson.D{
		{
			Key: "$match",
			Value: bson.M{
				"_id": idObjPublication,
			},
		},
	}
	// Get publication
	var publication []*models.Publication
	cursor, err := publicationModel.Aggreagate(mongo.Pipeline{match})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(db.Ctx, &publication); err != nil {
		return nil, err
	}

	if len(publication) == 0 {
		return nil, fmt.Errorf("no existe esta publicación")
	}
	// Get module
	var module *models.Module
	cursorM := moduleModel.GetByID(idObjModule)
	if err := cursorM.Decode(&module); err != nil {
		return nil, err
	}

	var flag bool
	for _, subSection := range module.SubSections {
		if subSection.ID == publication[0].SubSection {
			flag = true
			break
		}
	}
	if !flag {
		return nil, fmt.Errorf("esta publicación no pertenece a este módulo")
	}
	// Get publications content
	var publicationsRes *PublicationsRes

	es, err := db.NewConnectionEs()
	if err != nil {
		return nil, err
	}
	res, err := es.Get(models.PUBLICATIONS_INDEX, publication[0].ID.Hex())
	if err != nil {
		return nil, err
	}
	// Close body
	defer res.Body.Close()
	// Decode data
	var mapRes map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&mapRes); err != nil {
		return nil, err
	}
	// Get files
	var attacheds []AttachedRes

	for _, attached := range publication[0].Attached {
		var file *models.File

		if attached.Type == "file" {
			file, err = fileModel.GetFileByID(attached.File)
			if err != nil {
				return nil, err
			}
		}

		attacheds = append(attacheds, AttachedRes{
			ID:    attached.ID.Hex(),
			Type:  attached.Type,
			Link:  attached.Link,
			Title: attached.Title,
			File:  file,
		})
	}
	// Add response
	publicationsRes = &PublicationsRes{
		Attached:   attacheds,
		Content:    mapRes["_source"],
		ID:         publication[0].ID.Hex(),
		UploadDate: publication[0].UploadDate,
		UpdateDate: publication[0].UpdateDate,
	}
	return publicationsRes, nil
}

func (publication *PublicationService) NewPublication(
	publicationData *forms.PublicationForm,
	claims *Claims,
	section,
	idModule string,
) (map[string]interface{}, *res.ErrorRes) {
	userIdObj, _ := primitive.ObjectIDFromHex(claims.ID)
	sectionInt, err := strconv.Atoi(section)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get sub section ID
	module, err := moduleService.GetModuleFromID(idModule)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	if len(module.SubSections) <= sectionInt {
		return nil, &res.ErrorRes{
			Err:        fmt.Errorf("no existe esta sección"),
			StatusCode: http.StatusBadRequest,
		}
	}
	// Insert publication mongoDB
	newPublicationModel, attachedIds, err := models.NewModelPublication(
		publicationData,
		userIdObj,
		module.SubSections[sectionInt].ID,
	)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	insertedPublication, err := publicationModel.NewDocument(newPublicationModel)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Insert publication ElasticSearch
	publicationEs := &models.ContentPublication{
		Content:   publicationData.Content,
		Author:    claims.Name,
		Published: time.Now().Round(time.Second).UTC(),
		IDModule:  idModule,
	}
	data, err := json.Marshal(publicationEs)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusInternalServerError,
		}
	}
	// Add item to the BulkIndexer
	oid, _ := insertedPublication.InsertedID.(primitive.ObjectID)
	bi, err := models.NewBulkPublication()
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	err = bi.Add(
		context.Background(),
		esutil.BulkIndexerItem{
			Action:     "index",
			DocumentID: oid.Hex(),
			Body:       bytes.NewReader(data),
		},
	)
	if err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if err := bi.Close(context.Background()); err != nil {
		return nil, &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Notification
	var titleOfNotification string
	for i := 0; i < len(publicationData.Content); i++ {
		titleOfNotification += string(publicationData.Content[i])
		if i == 19 {
			break
		}
	}
	titleOfNotification += "..."

	nats.PublishEncode("notify/classroom", res.NotifyClassroom{
		Title: titleOfNotification,
		Link: fmt.Sprintf(
			"/aula_virtual/clase/%s/publicacion/%s",
			idModule,
			insertedPublication.InsertedID.(primitive.ObjectID).Hex(),
		),
		Where: module.Subject.Hex(),
		Room:  module.Section.Hex(),
		Type:  res.PUBLICATION,
	})
	// Response
	response := make(map[string]interface{})
	response["_id"] = insertedPublication.InsertedID
	response["attached_ids"] = attachedIds
	return response, nil
}

func (publication *PublicationService) UpdatePublication(
	content *forms.PublicationUpdateForm,
	idPublication,
	idUser string,
) *res.ErrorRes {
	idPublicationObj, err := primitive.ObjectIDFromHex(idPublication)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idUserObj, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get Publication
	var publicationData *models.Publication
	cursor := publicationModel.GetByID(idPublicationObj)
	err = cursor.Decode(&publicationData)
	if err != nil {
		if err.Error() == db.NO_SINGLE_DOCUMENT {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusNotFound,
			}
		}
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Verify user
	if idUserObj != publicationData.Author {
		return &res.ErrorRes{
			Err:        fmt.Errorf("no tienes acceso a esta publicación"),
			StatusCode: http.StatusUnauthorized,
		}
	}
	// Update
	// Update content
	data, err := json.Marshal(content)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusInternalServerError,
		}
	}
	bi, err := models.NewBulkPublication()
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	err = bi.Add(
		context.Background(),
		esutil.BulkIndexerItem{
			Action:     "update",
			DocumentID: idPublication,
			Body:       bytes.NewReader([]byte(fmt.Sprintf(`{"doc":%s}`, data))),
		},
	)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if err := bi.Close(context.Background()); err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Update date
	_, err = publicationModel.Use().UpdateByID(db.Ctx, idPublicationObj, bson.D{
		{
			Key: "$set",
			Value: bson.M{
				"update_date": primitive.DateTime(primitive.NewDateTimeFromTime(time.Now())),
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

func (publication *PublicationService) DeletePublication(
	idModule,
	idPublication string,
	claims Claims,
) *res.ErrorRes {
	idModuleObj, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idPublicationObj, err := primitive.ObjectIDFromHex(idPublication)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Get publication
	var publicationData *models.Publication
	cursor := publicationModel.GetByID(idPublicationObj)
	err = cursor.Decode(&publicationData)
	if err != nil {
		if err.Error() == db.NO_SINGLE_DOCUMENT {
			return &res.ErrorRes{
				Err:        err,
				StatusCode: http.StatusNotFound,
			}
		}
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Verify access
	err = hasAccessFromIdModuleNSubSection(idModuleObj, publicationData.SubSection)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusUnauthorized,
		}
	}
	// Delete publication
	// ElasticSearch
	bi, err := models.NewBulkPublication()
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	err = bi.Add(
		context.Background(),
		esutil.BulkIndexerItem{
			Action:     "delete",
			DocumentID: idPublication,
		},
	)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if err := bi.Close(context.Background()); err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Mongodb
	_, err = publicationModel.Use().DeleteOne(db.Ctx, bson.M{
		"_id": idPublicationObj,
	})
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	// Delete notifications
	nats.Publish("delete_notification", []byte(idPublication))
	return nil
}

func (publication *PublicationService) DeletePublicationAttached(
	idModule,
	idAttached string,
	claims Claims,
) *res.ErrorRes {
	idModuleObj, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	idAttachedObj, err := primitive.ObjectIDFromHex(idAttached)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusBadRequest,
		}
	}
	// Match
	match := bson.D{
		{
			Key: "$match",
			Value: bson.M{
				"attached": bson.M{
					"$elemMatch": bson.M{
						"_id": idAttachedObj,
					},
				},
			},
		},
	}
	// Get publication
	var publicationData []models.Publication

	cursor, err := publicationModel.Aggreagate(mongo.Pipeline{
		match,
	})
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if err = cursor.All(db.Ctx, &publicationData); err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusServiceUnavailable,
		}
	}
	if publicationData == nil {
		return &res.ErrorRes{
			Err:        fmt.Errorf("no existe el elemento adjunto"),
			StatusCode: http.StatusNotFound,
		}
	}
	if len(publicationData[0].Attached) == 0 {
		return &res.ErrorRes{
			Err:        fmt.Errorf("esta publicación no tiene elementos adjuntos"),
			StatusCode: http.StatusBadRequest,
		}
	}
	// Verify access
	err = hasAccessFromIdModuleNSubSection(idModuleObj, publicationData[0].SubSection)
	if err != nil {
		return &res.ErrorRes{
			Err:        err,
			StatusCode: http.StatusUnauthorized,
		}
	}
	// Delete attached
	_, err = publicationModel.Use().UpdateByID(db.Ctx, publicationData[0].ID, bson.D{
		{
			Key: "$pull",
			Value: bson.M{
				"attached": bson.M{
					"_id": idAttachedObj,
				},
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

func hasAccessFromIdModuleNSubSection(idModule, idSubSection primitive.ObjectID) error {
	match := bson.D{
		{
			Key: "$match",
			Value: bson.M{
				"_id": idModule,
				"sub_sections": bson.M{
					"$elemMatch": bson.M{
						"_id": idSubSection,
					},
				},
			},
		},
	}
	project := bson.D{
		{
			Key: "$project",
			Value: bson.M{
				"_id": 1,
			},
		},
	}
	cursorAll, err := moduleModel.Aggreagate(mongo.Pipeline{match, project})
	if err != nil {
		return err
	}
	var modules []models.Module
	if err = cursorAll.All(db.Ctx, &modules); err != nil {
		return err
	}
	if len(modules) == 0 {
		return fmt.Errorf("no tienes acceso a esta publicación")
	}
	return nil
}

func NewPublicationsService() *PublicationService {
	if publicationService == nil {
		publicationService = &PublicationService{}
	}
	return publicationService
}
