package services

import (
	"github.com/crawlab-team/crawlab-core/controllers"
	"github.com/crawlab-team/crawlab-core/spider/fs"
	"github.com/crawlab-team/crawlab-core/utils"
	"github.com/crawlab-team/plugin-scrapy/constants"
	"github.com/crawlab-team/plugin-scrapy/entity"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"path"
)

type SpiderService struct {
	parent *Service
	api    *gin.Engine
}

func (svc *SpiderService) Init() {
	svc.api.GET("/spiders/:id", svc.get)
}

func (svc *SpiderService) get(c *gin.Context) {
	// spider id
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		controllers.HandleErrorBadRequest(c, err)
		return
	}

	// spider fs service
	fsSvc, err := fs.NewSpiderFsService(id)
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// sync to workspace
	if err := fsSvc.GetFsService().SyncToWorkspace(); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// workspace path
	workspacePath := fsSvc.GetWorkspacePath()

	// spider info
	info := entity.SpiderInfo{}
	info.Framework = svc._getFramework(workspacePath)

	controllers.HandleSuccessWithData(c, info)
}

func (svc *SpiderService) _getFramework(workspacePath string) (t string) {
	if utils.Exists(path.Join(workspacePath, constants.ScrapyCfgFileName)) {
		return constants.SpiderTypeScrapy
	}
	return ""
}

func NewSpiderService(parent *Service) (svc *SpiderService) {
	svc = &SpiderService{
		parent: parent,
		api:    parent.GetApi(),
	}
	return svc
}
