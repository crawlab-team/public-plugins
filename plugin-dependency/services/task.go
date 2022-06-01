package services

import (
	"github.com/crawlab-team/crawlab-core/controllers"
	mongo2 "github.com/crawlab-team/crawlab-db/mongo"
	"github.com/crawlab-team/plugin-dependency/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strconv"
)

type TaskService struct {
	parent *Service
	api    *gin.Engine
}

func (svc *TaskService) Init() {
	svc.api.GET("/tasks", svc.getList)
	svc.api.GET("/tasks/:id/logs", svc.getLogs)
}

func (svc *TaskService) getList(c *gin.Context) {
	// filter
	query, _ := controllers.GetFilterQuery(c)

	// all
	all, _ := strconv.ParseBool(c.Query("all"))

	// pagination
	pagination := controllers.MustGetPagination(c)

	// options
	opts := &mongo2.FindOptions{
		Sort: bson.D{{"_id", -1}},
	}
	if !all {
		opts.Skip = (pagination.Page - 1) * pagination.Size
		opts.Limit = pagination.Size
	}

	// tasks
	var tasks []models.Task
	if err := svc.parent.colT.Find(query, opts).All(&tasks); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// total
	total, err := svc.parent.colT.Count(query)
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	controllers.HandleSuccessWithListData(c, tasks, total)
}

func (svc *TaskService) getLogs(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		controllers.HandleErrorBadRequest(c, err)
		return
	}

	var logList []models.Log
	if err := svc.parent.colL.Find(bson.M{"task_id": id}, nil).All(&logList); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	controllers.HandleSuccessWithData(c, logList)
}

func NewTaskService(parent *Service) (svc *TaskService) {
	svc = &TaskService{
		parent: parent,
		api:    parent.GetApi(),
	}

	return svc
}
