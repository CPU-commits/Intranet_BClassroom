package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/CPU-commits/Intranet_BClassroom/models"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var publicationService *PublicationService

type PublicationService struct{}

// Responses type
type AttachedRes struct {
	ID    string       `json:"_id"`
	Type  string       `json:"type"`
	File  *models.File `json:"file"`
	Link  string       `json:"link"`
	Title string       `json:"title"`
}

type PublicationsRes struct {
	ID         string             `json:"_id" bson:"_id,omitempty"`
	Attached   []AttachedRes      `json:"attached" bson:"attached"`
	Content    interface{}        `json:"content"`
	UploadDate primitive.DateTime `json:"upload_date" bson:"upload_date"`
	UpdateDate primitive.DateTime `json:"update_date" bson:"update_date"`
}

func (publication *PublicationService) GetPublicationsFromIdModule(
	idModule,
	section string,
	skip,
	limit int,
	total bool,
) ([]*PublicationsRes, int64, error) {
	sectionInt, err := strconv.Atoi(section)
	if err != nil {
		return nil, 0, err
	}
	// Get sub section ID
	module, err := moduleService.GetModuleFromID(idModule)
	if err != nil {
		return nil, 0, err
	}
	if len(module.SubSections) <= sectionInt {
		return nil, 0, fmt.Errorf("No existe esta sección")
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
		return nil, 0, err
	}
	if err = cursor.All(db.Ctx, &publications); err != nil {
		return nil, 0, err
	}
	// Get publications content
	var publicationsRes []*PublicationsRes

	es, err := db.NewConnectionEs()
	if err != nil {
		return nil, 0, err
	}
	var newErr error
	var wg sync.WaitGroup
	for _, publication := range publications {
		wg.Add(1)
		go func(publication *models.Publication, retErr *error, wg *sync.WaitGroup) {
			defer wg.Done()
			res, err := es.Get(models.PUBLICATIONS_INDEX, publication.ID.Hex())
			if err != nil {
				*retErr = err
				return
			}
			// Decode data
			var mapRes map[string]interface{}
			if err := json.NewDecoder(res.Body).Decode(&mapRes); err != nil {
				*retErr = err
				return
			}
			// Get files
			var attacheds []AttachedRes

			for _, attached := range publication.Attached {
				var file *models.File

				if attached.Type == "file" {
					file, err = fileModel.GetFileByID(attached.File)
					if err != nil {
						*retErr = err
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
			publicationsRes = append(publicationsRes, &PublicationsRes{
				Attached:   attacheds,
				Content:    mapRes["_source"],
				ID:         publication.ID.Hex(),
				UploadDate: publication.UploadDate,
				UpdateDate: publication.UpdateDate,
			})
			// Close body
			res.Body.Close()
		}(publication, &newErr, &wg)
	}
	wg.Wait()
	if newErr != nil {
		return nil, 0, newErr
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
			return nil, 0, err
		}
	}
	return publicationsRes, totalData, nil
}

func (publication *PublicationService) NewPublication(
	publicationData *forms.PublicationForm,
	claims *Claims,
	section,
	idModule string,
) (map[string]interface{}, error) {
	userIdObj, _ := primitive.ObjectIDFromHex(claims.ID)
	sectionInt, err := strconv.Atoi(section)
	if err != nil {
		return nil, err
	}
	// Get sub section ID
	module, err := moduleService.GetModuleFromID(idModule)
	if err != nil {
		return nil, err
	}
	if len(module.SubSections) <= sectionInt {
		return nil, fmt.Errorf("No existe esta sección")
	}
	// Insert publication mongoDB
	newPublicationModel, attachedIds, err := models.NewModelPublication(
		publicationData,
		userIdObj,
		module.SubSections[sectionInt].ID,
	)
	insertedPublication, err := publicationModel.NewDocument(newPublicationModel)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	// Add item to the BulkIndexer
	oid, _ := insertedPublication.InsertedID.(primitive.ObjectID)
	bi, err := models.NewBulkPublication()
	if err != nil {
		return nil, err
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
		return nil, err
	}
	if err := bi.Close(context.Background()); err != nil {
		return nil, err
	}
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
) error {
	idPublicationObj, err := primitive.ObjectIDFromHex(idPublication)
	if err != nil {
		return err
	}
	idUserObj, err := primitive.ObjectIDFromHex(idUser)
	if err != nil {
		return err
	}
	// Get Publication
	var publicationData *models.Publication
	cursor := publicationModel.GetByID(idPublicationObj)
	err = cursor.Decode(&publicationData)
	if err != nil {
		return err
	}
	// Verify user
	if idUserObj != publicationData.Author {
		return fmt.Errorf("No tienes acceso a esta publicación")
	}
	// Update
	// Update content
	data, err := json.Marshal(content)
	bi, err := models.NewBulkPublication()
	if err != nil {
		return err
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
		return err
	}
	if err := bi.Close(context.Background()); err != nil {
		return err
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
		return err
	}
	return nil
}

func (publication *PublicationService) DeletePublication(
	idModule,
	idPublication string,
	claims Claims,
) error {
	idModuleObj, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return err
	}
	idPublicationObj, err := primitive.ObjectIDFromHex(idPublication)
	if err != nil {
		return err
	}
	// Get publication
	var publicationData *models.Publication
	cursor := publicationModel.GetByID(idPublicationObj)
	err = cursor.Decode(&publicationData)
	if err != nil {
		return err
	}
	// Verify access
	err = hasAccessFromIdModuleNSubSection(idModuleObj, publicationData.SubSection)
	if err != nil {
		return err
	}
	// Delete publication
	// ElasticSearch
	bi, err := models.NewBulkPublication()
	if err != nil {
		return err
	}
	err = bi.Add(
		context.Background(),
		esutil.BulkIndexerItem{
			Action:     "delete",
			DocumentID: idPublication,
		},
	)
	if err != nil {
		return err
	}
	if err := bi.Close(context.Background()); err != nil {
		return err
	}
	// Mongodb
	_, err = publicationModel.Use().DeleteOne(db.Ctx, bson.M{
		"_id": idPublicationObj,
	})
	if err != nil {
		return err
	}
	return nil
}

func (publication *PublicationService) DeletePublicationAttached(
	idModule,
	idAttached string,
	claims Claims,
) error {
	idModuleObj, err := primitive.ObjectIDFromHex(idModule)
	if err != nil {
		return err
	}
	idAttachedObj, err := primitive.ObjectIDFromHex(idAttached)
	if err != nil {
		return err
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
		return err
	}
	if err = cursor.All(db.Ctx, &publicationData); err != nil {
		return err
	}
	if publicationData == nil {
		return fmt.Errorf("No existe el elemento adjunto")
	}
	if len(publicationData[0].Attached) == 0 {
		return fmt.Errorf("Esta publicación no tiene elementos adjuntos")
	}
	// Verify access
	err = hasAccessFromIdModuleNSubSection(idModuleObj, publicationData[0].SubSection)
	if err != nil {
		return err
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
		return err
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
		return fmt.Errorf("No tienes acceso a esta publicación")
	}
	return nil
}

func NewPublicationsService() *PublicationService {
	if publicationService == nil {
		publicationService = &PublicationService{}
	}
	return publicationService
}
