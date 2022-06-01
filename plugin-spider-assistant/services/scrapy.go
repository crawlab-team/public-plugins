package services

import (
	"encoding/json"
	"github.com/crawlab-team/crawlab-core/controllers"
	"github.com/crawlab-team/crawlab-core/spider/fs"
	"github.com/crawlab-team/go-trace"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"os/exec"
	"path"
)

type ScrapyService struct {
	parent *Service
	api    *gin.Engine
}

func (svc *ScrapyService) Init() {
	svc.api.GET("/scrapy/:id", svc.get)
}

func (svc *ScrapyService) get(c *gin.Context) {
	// spider id
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		controllers.HandleErrorBadRequest(c, err)
		return
	}

	//// spider
	//s, err := svc.parent.getSpider(id)
	//if err != nil {
	//	controllers.HandleErrorInternalServerError(c, err)
	//	return
	//}

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

	// results
	res, err := svc._getResults(workspacePath)
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	controllers.HandleSuccessWithData(c, res)
}

func (svc *ScrapyService) _getResults(workspacePath string) (res bson.M, err error) {
	// arguments
	var args []string

	// script path
	scriptPath := path.Join("scripts", "scrapy.py")
	args = append(args, scriptPath)

	// directory path
	args = append(args, "-d")
	args = append(args, workspacePath)

	// command
	cmd := exec.Command("python", args...)

	// output
	data, err := cmd.Output()
	if err != nil {
		return nil, trace.TraceError(err)
	}

	// deserialize
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, trace.TraceError(err)
	}

	return res, nil
}

func NewScrapyService(parent *Service) (svc *ScrapyService) {
	svc = &ScrapyService{
		parent: parent,
		api:    parent.GetApi(),
	}
	return svc
}
