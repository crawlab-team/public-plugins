package services

import (
	"github.com/crawlab-team/crawlab-core/controllers"
	mongo2 "github.com/crawlab-team/crawlab-db/mongo"
	"github.com/crawlab-team/plugin-dependency/constants"
	"github.com/crawlab-team/plugin-dependency/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type SettingService struct {
	parent *Service
	api    *gin.Engine
	col    *mongo2.Col // dependency settings
}

func (svc *SettingService) Init() {
	svc.api.GET("/settings", svc.getSettingList)
	svc.api.GET("/settings/:id", svc.getSetting)
	svc.api.PUT("/settings", svc.putSetting)
	svc.api.POST("/settings/:id", svc.postSetting)
	svc.api.DELETE("/settings/:id", svc.deleteSetting)
	svc.api.POST("/settings/:id/enable", svc.enableSetting)
	svc.api.POST("/settings/:id/disable", svc.disableSetting)
}

func (svc *SettingService) getSettingList(c *gin.Context) {
	// params
	pagination := controllers.MustGetPagination(c)
	query := controllers.MustGetFilterQuery(c)
	sort := controllers.MustGetSortOption(c)

	// get list
	var list []models.Setting
	if err := svc.col.Find(query, &mongo2.FindOptions{
		Sort:  sort,
		Skip:  pagination.Size * (pagination.Page - 1),
		Limit: pagination.Size,
	}).All(&list); err != nil {
		if err.Error() == mongo.ErrNoDocuments.Error() {
			controllers.HandleSuccessWithListData(c, nil, 0)
		} else {
			controllers.HandleErrorInternalServerError(c, err)
		}
		return
	}

	// total count
	total, err := svc.col.Count(query)
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	controllers.HandleSuccessWithListData(c, list, total)
}

func (svc *SettingService) getSetting(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		controllers.HandleErrorBadRequest(c, err)
		return
	}

	var s models.Setting
	if err := svc.col.FindId(id).One(&s); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	controllers.HandleSuccessWithData(c, s)
}

func (svc *SettingService) putSetting(c *gin.Context) {
	var s models.Setting
	if err := c.ShouldBindJSON(&s); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	s.Id = primitive.NewObjectID()
	if _, err := svc.col.Insert(s); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	controllers.HandleSuccessWithData(c, s)
}

func (svc *SettingService) postSetting(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		controllers.HandleErrorBadRequest(c, err)
		return
	}

	var s models.Setting
	if err := svc.col.FindId(id).One(&s); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	if err := c.ShouldBindJSON(&s); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}
	s.Id = id

	if err := svc.col.ReplaceId(id, s); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	controllers.HandleSuccessWithData(c, s)
}

func (svc *SettingService) deleteSetting(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		controllers.HandleErrorBadRequest(c, err)
		return
	}

	if err := svc.col.DeleteId(id); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	controllers.HandleSuccess(c)
}

func (svc *SettingService) enableSetting(c *gin.Context) {
	svc._toggleSettingFunc(true)(c)
}

func (svc *SettingService) disableSetting(c *gin.Context) {
	svc._toggleSettingFunc(false)(c)
}

func (svc *SettingService) _toggleSettingFunc(value bool) func(c *gin.Context) {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			controllers.HandleErrorBadRequest(c, err)
			return
		}
		var s models.Setting
		if err := svc.col.FindId(id).One(&s); err != nil {
			controllers.HandleErrorInternalServerError(c, err)
			return
		}
		s.Enabled = value
		if err := svc.col.ReplaceId(id, s); err != nil {
			controllers.HandleErrorInternalServerError(c, err)
			return
		}
		controllers.HandleSuccess(c)
	}
}

func NewSettingService(parent *Service) (svc *SettingService) {
	svc = &SettingService{
		parent: parent,
		api:    parent.GetApi(),
		col:    mongo2.GetMongoCol(constants.DependencySettingsColName),
	}

	return svc
}
