package core

import (
	"encoding/json"
	"fmt"
	"github.com/apex/log"
	"github.com/crawlab-team/crawlab-core/controllers"
	"github.com/crawlab-team/crawlab-core/entity"
	"github.com/crawlab-team/crawlab-core/interfaces"
	mongo2 "github.com/crawlab-team/crawlab-db/mongo"
	grpc "github.com/crawlab-team/crawlab-grpc"
	plugin "github.com/crawlab-team/crawlab-plugin"
	"github.com/crawlab-team/go-trace"
	parser "github.com/crawlab-team/template-parser"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"io"
	"os"
	"strings"
	"time"
)

type Service struct {
	*plugin.Internal
	col *mongo2.Col // notification settings
}

func (svc *Service) Init() (err error) {
	// handle events
	go svc.handleEvents()

	// api
	api := svc.GetApi()
	//api.POST("/send", svc.send)
	api.GET("/triggers", svc.getTriggerList)
	api.GET("/settings", svc.getSettingList)
	api.GET("/settings/:id", svc.getSetting)
	api.PUT("/settings", svc.putSetting)
	api.POST("/settings/:id", svc.postSetting)
	api.DELETE("/settings/:id", svc.deleteSetting)
	api.POST("/settings/:id/enable", svc.enableSetting)
	api.POST("/settings/:id/disable", svc.disableSetting)

	return nil
}

func (svc *Service) Start() (err error) {
	// initialize data
	if err := svc.initData(); err != nil {
		return err
	}

	// start api
	svc.StartApi()

	return nil
}

func (svc *Service) Stop() (err error) {
	svc.StopApi()
	return nil
}

func (svc *Service) initData() (err error) {
	total, err := svc.col.Count(nil)
	if err != nil {
		return err
	}
	if total > 0 {
		return nil
	}

	// data to initialize
	settings := []NotificationSetting{
		{
			Id:          primitive.NewObjectID(),
			Type:        NotificationTypeMail,
			Enabled:     true,
			Name:        "Task Change (Mail)",
			Description: "This is the default mail notification. You can edit it with your own settings",
			Triggers: []string{
				"model:tasks:change",
			},
			Title: "[Crawlab] Task Update: {{$.status}}",
			Template: `Dear {{$.user.username}},

Please find the task data as below.

|Key|Value|
|:-:|:--|
|Task Status|{{$.status}}|
|Task Priority|{{$.priority}}|
|Task Mode|{{$.mode}}|
|Task Command|{{$.cmd}}|
|Task Params|{{$.params}}|
|Error Message|{{$.error}}|
|Node|{{$.node.name}}|
|Spider|{{$.spider.name}}|
|Project|{{$.spider.project.name}}|
|Schedule|{{$.schedule.name}}|
|Result Count|{{$.:task_stat.result_count}}|
|Wait Duration (sec)|{#{{$.:task_stat.wait_duration}}/1000#}|
|Runtime Duration (sec)|{#{{$.:task_stat.runtime_duration}}/1000#}|
|Total Duration (sec)|{#{{$.:task_stat.total_duration}}/1000#}|
|Result Count|{{$.:task_stat.result_count}}|
|Avg Results / Sec|{#{{$.:task_stat.result_count}}/({{$.:task_stat.total_duration}}/1000)#}|
`,
			Mail: NotificationSettingMail{
				Server:         "smtp.163.com",
				Port:           "465",
				User:           os.Getenv("CRAWLAB_PLUGIN_NOTIFICATION_MAIL_USER"),
				Password:       os.Getenv("CRAWLAB_PLUGIN_NOTIFICATION_MAIL_PASSWORD"),
				SenderEmail:    os.Getenv("CRAWLAB_PLUGIN_NOTIFICATION_MAIL_SENDER_EMAIL"),
				SenderIdentity: os.Getenv("CRAWLAB_PLUGIN_NOTIFICATION_MAIL_SENDER_IDENTITY"),
				To:             "{{$.user[create].email}}",
				Cc:             os.Getenv("CRAWLAB_PLUGIN_NOTIFICATION_MAIL_CC"),
			},
		},
		{
			Id:          primitive.NewObjectID(),
			Type:        NotificationTypeMobile,
			Enabled:     true,
			Name:        "Task Change (Mobile)",
			Description: "This is the default mobile notification. You can edit it with your own settings",
			Triggers: []string{
				"model:tasks:change",
			},
			Title: "[Crawlab] Task Update: {{$.status}}",
			Template: `Dear {{$.user.username}},

Please find the task data as below.

- **Task Status**: {{$.status}}
- **Task Priority**: {{$.priority}}
- **Task Mode**: {{$.mode}}
- **Task Command**: {{$.cmd}}
- **Task Params**: {{$.params}}
- **Error Message**: {{$.error}}
- **Node**: {{$.node.name}}
- **Spider**: {{$.spider.name}}
- **Project**: {{$.spider.project.name}}
- **Schedule**: {{$.schedule.name}}
- **Result Count**: {{$.:task_stat.result_count}}
- **Wait Duration (sec)**: {#{{$.:task_stat.wait_duration}}/1000#}
- **Runtime Duration (sec)**: {#{{$.:task_stat.runtime_duration}}/1000#}
- **Total Duration (sec)**: {#{{$.:task_stat.total_duration}}/1000#}
- **Result Count**: {{$.:task_stat.result_count}}
- **Avg Results / Sec**: {#{{$.:task_stat.result_count}}/({{$.:task_stat.total_duration}}/1000)#}`,
			Mobile: NotificationSettingMobile{
				Webhook: os.Getenv("CRAWLAB_PLUGIN_NOTIFICATION_MOBILE_WEBHOOK"),
			},
		},
	}
	var data []interface{}
	for _, s := range settings {
		data = append(data, s)
	}
	_, err = svc.col.InsertMany(data)
	if err != nil {
		return err
	}
	return nil
}

func (svc *Service) sendMail(s *NotificationSetting, entity bson.M) (err error) {
	// to
	to, err := parser.Parse(s.Mail.To, entity)
	if err != nil {
		log.Warnf("parsing 'to' error: %v", err)
	}
	if to == "" {
		return nil
	}

	// cc
	cc, err := parser.Parse(s.Mail.Cc, entity)
	if err != nil {
		log.Warnf("parsing 'cc' error: %v", err)
	}

	// title
	title, err := parser.Parse(s.Title, entity)
	if err != nil {
		log.Warnf("parsing 'title' error: %v", err)
	}

	// content
	content, err := parser.Parse(s.Template, entity)
	if err != nil {
		log.Warnf("parsing 'content' error: %v", err)
	}

	// send mail
	if err := SendMail(s, to, cc, title, content); err != nil {
		return err
	}

	return nil
}

func (svc *Service) sendMobile(s *NotificationSetting, entity bson.M) (err error) {
	// webhook
	webhook, err := parser.Parse(s.Mobile.Webhook, entity)
	if err != nil {
		log.Warnf("parsing 'webhook' error: %v", err)
	}
	if webhook == "" {
		return nil
	}

	// title
	title, err := parser.Parse(s.Title, entity)
	if err != nil {
		log.Warnf("parsing 'title' error: %v", err)
	}

	// content
	content, err := parser.Parse(s.Template, entity)
	if err != nil {
		log.Warnf("parsing 'content' error: %v", err)
	}

	// send
	if err := SendMobileNotification(webhook, title, content); err != nil {
		return err
	}

	return nil
}

func (svc *Service) getTriggerList(c *gin.Context) {
	modelList := []string{
		interfaces.ModelColNameTag,
		interfaces.ModelColNameNode,
		interfaces.ModelColNameProject,
		interfaces.ModelColNameSpider,
		interfaces.ModelColNameTask,
		interfaces.ModelColNameJob,
		interfaces.ModelColNameSchedule,
		interfaces.ModelColNameUser,
		interfaces.ModelColNameSetting,
		interfaces.ModelColNameToken,
		interfaces.ModelColNameVariable,
		interfaces.ModelColNameTaskStat,
		interfaces.ModelColNamePlugin,
		interfaces.ModelColNameSpiderStat,
		interfaces.ModelColNameDataSource,
		interfaces.ModelColNameDataCollection,
		interfaces.ModelColNamePasswords,
	}
	actionList := []string{
		interfaces.ModelDelegateMethodAdd,
		interfaces.ModelDelegateMethodChange,
		interfaces.ModelDelegateMethodDelete,
		interfaces.ModelDelegateMethodSave,
	}

	var triggers []string
	for _, m := range modelList {
		for _, a := range actionList {
			triggers = append(triggers, fmt.Sprintf("model:%s:%s", m, a))
		}
	}

	controllers.HandleSuccessWithListData(c, triggers, len(triggers))
}

func (svc *Service) getSettingList(c *gin.Context) {
	// params
	pagination := controllers.MustGetPagination(c)
	query := controllers.MustGetFilterQuery(c)
	sort := controllers.MustGetSortOption(c)

	// get list
	var list []NotificationSetting
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

func (svc *Service) getSetting(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		controllers.HandleErrorBadRequest(c, err)
		return
	}

	var s NotificationSetting
	if err := svc.col.FindId(id).One(&s); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	controllers.HandleSuccessWithData(c, s)
}

func (svc *Service) putSetting(c *gin.Context) {
	var s NotificationSetting
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

func (svc *Service) postSetting(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		controllers.HandleErrorBadRequest(c, err)
		return
	}

	var s NotificationSetting
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

func (svc *Service) deleteSetting(c *gin.Context) {
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

func (svc *Service) enableSetting(c *gin.Context) {
	svc._toggleSettingFunc(true)(c)
}

func (svc *Service) disableSetting(c *gin.Context) {
	svc._toggleSettingFunc(false)(c)
}

func (svc *Service) handleEvents() {
	log.Infof("start handling events")

	// get stream
	log.Infof("attempt to obtain grpc stream...")
	var stream grpc.PluginService_SubscribeClient
	for {
		stream = svc.Internal.GetEventService().GetStream()
		if stream == nil {
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}
	log.Infof("obtained grpc stream, start receiving messages...")

	for {
		// receive stream message
		msg, err := stream.Recv()

		if err != nil {
			// end
			if strings.HasSuffix(err.Error(), io.EOF.Error()) {
				// TODO: implement
				log.Infof("received EOF signal, re-connecting...")
				//svc.GetGrpcClient().Restart()
			}

			trace.PrintError(err)
			time.Sleep(1 * time.Second)
			continue
		}

		var data entity.GrpcEventServiceMessage
		switch msg.Code {
		case grpc.StreamMessageCode_SEND_EVENT:
			// data
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				return
			}
			if len(data.Events) < 1 {
				continue
			}

			// event name
			eventName := data.Events[0]

			// settings
			var settings []NotificationSetting
			if err := svc.col.Find(bson.M{
				"enabled":  true,
				"triggers": eventName,
			}, nil).All(&settings); err != nil || len(settings) == 0 {
				continue
			}

			// handle events
			if err := svc._handleEventModel(settings, data.Data); err != nil {
				trace.PrintError(err)
			}
		default:
			continue
		}
	}
}

func (svc *Service) _handleEventModel(settings []NotificationSetting, data []byte) (err error) {
	var doc bson.M
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	for _, s := range settings {
		switch s.Type {
		case NotificationTypeMail:
			err = svc.sendMail(&s, doc)
		case NotificationTypeMobile:
			// TODO: implement
			err = svc.sendMobile(&s, doc)
		}
		if err != nil {
			trace.PrintError(err)
		}
	}

	return nil
}

func (svc *Service) _toggleSettingFunc(value bool) func(c *gin.Context) {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			controllers.HandleErrorBadRequest(c, err)
			return
		}
		var s NotificationSetting
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

func NewService() *Service {
	// service
	svc := &Service{
		Internal: plugin.NewInternal(),
		col:      mongo2.GetMongoCol(NotificationSettingsColName),
	}

	if err := svc.Init(); err != nil {
		panic(err)
	}

	return svc
}
