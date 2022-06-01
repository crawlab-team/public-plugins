package services

import (
	"github.com/crawlab-team/crawlab-core/interfaces"
	"github.com/crawlab-team/crawlab-core/models/models"
	"github.com/crawlab-team/crawlab-core/utils"
	plugin "github.com/crawlab-team/crawlab-plugin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	*plugin.Internal

	spiderSvc *SpiderService
	scrapySvc *ScrapyService
}

func (svc *Service) Init() (err error) {
	svc.spiderSvc.Init()
	svc.scrapySvc.Init()
	return nil
}

func (svc *Service) Start() (err error) {
	svc.StartApi()
	utils.DefaultWait()
	return nil
}

func (svc *Service) Stop() (err error) {
	return nil
}

func (svc *Service) getSpider(id primitive.ObjectID) (s *models.Spider, err error) {
	spiderSvc, err := svc.GetModelService().NewBaseServiceDelegate(interfaces.ModelIdSpider)
	if err != nil {
		return nil, err
	}
	doc, err := spiderSvc.GetById(id)
	if err != nil {
		return nil, err
	}
	s = doc.(*models.Spider)
	return s, nil
}

func NewService() *Service {
	svc := &Service{
		Internal: plugin.NewInternal(),
	}

	svc.spiderSvc = NewSpiderService(svc)
	svc.scrapySvc = NewScrapyService(svc)

	if err := svc.Init(); err != nil {
		panic(err)
	}

	return svc
}
