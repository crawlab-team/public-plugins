package services

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"github.com/cenkalti/backoff/v4"
	"github.com/crawlab-team/crawlab-core/interfaces"
	models2 "github.com/crawlab-team/crawlab-core/models/models"
	"github.com/crawlab-team/crawlab-core/node/config"
	mongo2 "github.com/crawlab-team/crawlab-db/mongo"
	grpc "github.com/crawlab-team/crawlab-grpc"
	plugin "github.com/crawlab-team/crawlab-plugin"
	"github.com/crawlab-team/go-trace"
	"github.com/crawlab-team/plugin-dependency/constants"
	"github.com/crawlab-team/plugin-dependency/entity"
	"github.com/crawlab-team/plugin-dependency/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/dig"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Service struct {
	*plugin.Internal

	// internals
	colS        *mongo2.Col // dependency settings
	colD        *mongo2.Col // dependencies
	colT        *mongo2.Col // dependency tasks
	colL        *mongo2.Col // dependency logs
	cfgSvc      interfaces.NodeConfigService
	currentNode interfaces.Node
	masterNode  interfaces.Node
	msgStream   grpc.MessageService_ConnectClient

	// sub services
	settingSvc *SettingService
	taskSvc    *TaskService
	pythonSvc  *PythonService
	nodeSvc    *NodeService
	spiderSvc  *SpiderService
}

func (svc *Service) Init() (err error) {
	// initialize sub services
	svc.settingSvc.Init()
	svc.taskSvc.Init()
	svc.pythonSvc.Init()
	svc.nodeSvc.Init()
	svc.spiderSvc.Init()

	return nil
}

func (svc *Service) Start() (err error) {
	if svc.cfgSvc.IsMaster() {
		// initialize data
		if err := svc.initData(); err != nil {
			return err
		}

		// initialize indexes
		if err := svc.initIndexes(); err != nil {
			return err
		}

		// start api
		go svc.StartApi()

		// start python service
		go svc.pythonSvc.Start()
	}

	// get current node
	if err := svc.getCurrentNode(); err != nil {
		return err
	}

	// get master node
	if err := svc.getMasterNode(); err != nil {
		return err
	}

	// handle stream messages
	svc.handleStreamMessages()

	// wait
	svc.Wait()

	return nil
}

func (svc *Service) Stop() (err error) {
	svc.StopApi()
	return nil
}

func (svc *Service) initData() (err error) {
	total, err := svc.colS.Count(nil)
	if err != nil {
		return err
	}
	if total > 0 {
		return nil
	}

	// data to initialize
	settings := []models.Setting{
		{
			Id:          primitive.NewObjectID(),
			Key:         constants.DependencyTypePython,
			Name:        "Python",
			Description: "settings.description.python",
			Cmd:         "pip",
			Enabled:     true,
		},
		{
			Id:          primitive.NewObjectID(),
			Key:         constants.DependencyTypeNode,
			Name:        "Node.js",
			Cmd:         "npm",
			Description: "settings.description.node",
			Enabled:     true,
		},
	}
	var data []interface{}
	for _, s := range settings {
		data = append(data, s)
	}
	_, err = svc.colS.InsertMany(data)
	if err != nil {
		return err
	}
	return nil
}

func (svc *Service) initIndexes() (err error) {
	// settings
	optsColS := &options.IndexOptions{}
	optsColS.SetUnique(true)
	_ = svc.colS.CreateIndexes([]mongo.IndexModel{
		{
			Keys: bson.D{
				{"type", 1},
				{"node_id", 1},
				{"name", 1},
			},
			Options: optsColS,
		},
	})

	// tasks
	optsColT := &options.IndexOptions{}
	optsColT.SetExpireAfterSeconds(60 * 60 * 24)
	_ = svc.colT.CreateIndexes([]mongo.IndexModel{
		{
			Keys: bson.D{
				{"update_ts", 1},
			},
			Options: optsColT,
		},
	})

	// logs
	optsColL := &options.IndexOptions{}
	optsColL.SetExpireAfterSeconds(60 * 60 * 24)
	_ = svc.colL.CreateIndexes([]mongo.IndexModel{
		{
			Keys: bson.D{{"task_id", 1}},
		},
		{
			Keys:    bson.D{{"update_ts", 1}},
			Options: optsColL,
		},
	})

	return nil
}

func (svc *Service) handleStreamMessages() {
	if err := svc.connect(); err != nil {
		panic(err)
	}

	for {
		msg, err := svc.msgStream.Recv()
		if err == io.EOF {
			return
		}

		if err != nil {
			trace.PrintError(err)
			time.Sleep(1 * time.Second)
			_ = svc.connect()
			continue
		}

		var msgData entity.MessageData
		if err := json.Unmarshal(msg.Data, &msgData); err != nil {
			trace.PrintError(err)
			time.Sleep(1 * time.Second)
			_ = svc.connect()
			continue
		}

		switch msgData.Code {
		case constants.MessageCodeUpdateTask:
			go svc.updateTask(msg, msgData)
		case constants.MessageCodeInsertLogs:
			go svc.insertLogs(msg, msgData)

		case constants.MessageCodePythonUpdate:
			go svc.pythonSvc.updateDependencyList(msg, msgData)
		case constants.MessageCodePythonSave:
			go svc.pythonSvc._saveDependencyList(msg, msgData)
		case constants.MessageCodePythonInstall:
			go svc.pythonSvc.installDependency(msg, msgData)
		case constants.MessageCodePythonUninstall:
			go svc.pythonSvc.uninstallDependency(msg, msgData)

		case constants.MessageCodeNodeUpdate:
			go svc.nodeSvc.updateDependencyList(msg, msgData)
		case constants.MessageCodeNodeSave:
			go svc.nodeSvc._saveDependencyList(msg, msgData)
		case constants.MessageCodeNodeInstall:
			go svc.nodeSvc.installDependency(msg, msgData)
		case constants.MessageCodeNodeUninstall:
			go svc.nodeSvc.uninstallDependency(msg, msgData)
		}
	}
}

func (svc *Service) connect() (err error) {
	if err := backoff.Retry(svc._connect, backoff.NewExponentialBackOff()); err != nil {
		return err
	}
	return nil
}

func (svc *Service) _connect() (err error) {
	stream, err := svc.GetGrpcClient().GetMessageClient().Connect(context.Background())
	if err != nil {
		return err
	}
	msg := &grpc.StreamMessage{
		Code:    grpc.StreamMessageCode_CONNECT,
		NodeKey: svc.currentNode.GetKey(),
		Key:     "plugin:" + svc.currentNode.GetKey(),
	}
	if err := stream.Send(msg); err != nil {
		return err
	}
	svc.msgStream = stream
	return nil
}

func (svc *Service) getCurrentNode() (err error) {
	nodeModelSvc, err := svc.GetModelService().NewBaseServiceDelegate(interfaces.ModelIdNode)
	if err != nil {
		return err
	}
	res, err := nodeModelSvc.Get(bson.M{"key": svc.cfgSvc.GetNodeKey()}, nil)
	if err != nil {
		return err
	}
	n, ok := res.(interfaces.Node)
	if !ok {
		return errors.New("invalid type")
	}
	svc.currentNode = n
	return nil
}

func (svc *Service) getMasterNode() (err error) {
	nodeModelSvc, err := svc.GetModelService().NewBaseServiceDelegate(interfaces.ModelIdNode)
	if err != nil {
		return trace.TraceError(err)
	}
	doc, err := nodeModelSvc.Get(bson.M{"is_master": true}, nil)
	if err != nil {
		return trace.TraceError(err)
	}
	n, ok := doc.(interfaces.Node)
	if !ok {
		err := errors.New("invalid type")
		return trace.TraceError(err)
	}
	svc.masterNode = n
	return nil
}

func (svc *Service) updateTask(msg *grpc.StreamMessage, msgData entity.MessageData) {
	var taskMsg entity.TaskMessage
	if err := json.Unmarshal(msgData.Data, &taskMsg); err != nil {
		trace.PrintError(err)
		return
	}
	update := bson.M{
		"$set": bson.M{
			"status": taskMsg.Status,
			"error":  taskMsg.Error,
		},
	}
	if err := svc.colT.UpdateId(taskMsg.TaskId, update); err != nil {
		trace.PrintError(err)
		return
	}
}

func (svc *Service) insertLogs(msg *grpc.StreamMessage, msgData entity.MessageData) {
	var logsMsg entity.LogsMessage
	if err := json.Unmarshal(msgData.Data, &logsMsg); err != nil {
		trace.PrintError(err)
		return
	}
	l := &models.Log{
		Id:       primitive.NewObjectID(),
		TaskId:   logsMsg.TaskId,
		Content:  strings.Join(logsMsg.Lines, "\n"),
		UpdateTs: time.Now(),
	}
	if _, err := svc.colL.Insert(l); err != nil {
		trace.PrintError(err)
		return
	}
}

func (svc *Service) _configureLogging(taskId primitive.ObjectID, cmd *exec.Cmd) {
	var logLines []string
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		scannerStdout := bufio.NewScanner(stdout)
		for scannerStdout.Scan() {
			line := scannerStdout.Text()
			logLines = append(logLines, line)
			if len(logLines)%10 == 0 && len(logLines) > 0 {
				svc._sendLogs(taskId, logLines)
				logLines = []string{}
			}
		}
		wg.Done()
	}()
	go func() {
		scannerStderr := bufio.NewScanner(stderr)
		for scannerStderr.Scan() {
			line := scannerStderr.Text()
			logLines = append(logLines, line)
			if len(logLines)%10 == 0 && len(logLines) > 0 {
				svc._sendLogs(taskId, logLines)
				logLines = []string{}
			}
		}
		wg.Done()
	}()
	go func() {
		wg.Wait()
		if len(logLines) > 0 {
			svc._sendLogs(taskId, logLines)
			logLines = []string{}
		}
	}()
}

func (svc *Service) _sendLogs(taskId primitive.ObjectID, lines []string) {
	// logs message
	logsMsg := &entity.LogsMessage{
		TaskId: taskId,
		Lines:  lines,
	}

	// data
	data, _ := json.Marshal(logsMsg)

	// message data
	msgDataObj := &entity.MessageData{
		Code: constants.MessageCodeInsertLogs,
		Data: data,
	}
	msgData, _ := json.Marshal(msgDataObj)

	// stream message
	msg := &grpc.StreamMessage{
		Code:    grpc.StreamMessageCode_SEND,
		NodeKey: svc.currentNode.GetKey(),
		From:    "plugin:" + svc.currentNode.GetKey(),
		To:      "plugin:" + svc.masterNode.GetKey(),
		Data:    msgData,
	}

	// send message
	if err := svc.msgStream.Send(msg); err != nil {
		trace.PrintError(err)
		return
	}
}

func (svc *Service) _sendTaskStatus(taskId primitive.ObjectID, status string, err error) {
	// logs message
	logsMsg := &entity.TaskMessage{
		TaskId: taskId,
		Status: status,
	}

	// error
	if err != nil {
		logsMsg.Error = err.Error()
	}

	// data
	data, _ := json.Marshal(logsMsg)

	// message data
	msgDataObj := &entity.MessageData{
		Code: constants.MessageCodeUpdateTask,
		Data: data,
	}
	msgData, _ := json.Marshal(msgDataObj)

	// stream message
	msg := &grpc.StreamMessage{
		Code:    grpc.StreamMessageCode_SEND,
		NodeKey: svc.currentNode.GetKey(),
		From:    "plugin:" + svc.currentNode.GetKey(),
		To:      "plugin:" + svc.masterNode.GetKey(),
		Data:    msgData,
	}

	// send message
	if err := svc.msgStream.Send(msg); err != nil {
		trace.PrintError(err)
		return
	}
}

func (svc *Service) _getNodes(query bson.M) (nodes []models2.Node, err error) {
	// node model service
	nodeModelSvc, err := svc.GetModelService().NewBaseServiceDelegate(interfaces.ModelIdNode)
	if err != nil {
		return nil, err
	}

	// nodes
	list, err := nodeModelSvc.GetList(query, nil)
	if err != nil {
		return nil, err
	}
	for _, d := range list.Values() {
		n, ok := d.(models2.Node)
		if !ok {
			return nil, errors.New("invalid type")
		}
		nodes = append(nodes, n)
	}

	return nodes, nil
}

func (svc *Service) _updateDependencyList(
	msg *grpc.StreamMessage,
	msgData entity.MessageData,
	msgSaveCode string,
	getDependenciesFunc func(params entity.UpdateParams) (deps []models.Dependency, err error),
) {
	// params
	var params entity.UpdateParams
	if err := json.Unmarshal(msgData.Data, &params); err != nil {
		trace.PrintError(err)
		return
	}

	// installed dependencies
	deps, err := getDependenciesFunc(params)
	if err != nil {
		trace.PrintError(err)
		return
	}

	// data
	data, err := json.Marshal(deps)
	if err != nil {
		trace.PrintError(err)
		return
	}

	// message data
	msgDataObj := &entity.MessageData{
		Code: msgSaveCode,
		Data: data,
	}
	msgDataBytes, err := json.Marshal(msgDataObj)
	if err != nil {
		trace.PrintError(err)
		return
	}

	// stream message
	msg = &grpc.StreamMessage{
		Code:    grpc.StreamMessageCode_SEND,
		NodeKey: svc.currentNode.GetKey(),
		From:    "plugin:" + svc.currentNode.GetKey(),
		To:      "plugin:" + svc.masterNode.GetKey(),
		Data:    msgDataBytes,
	}

	// send message
	if err := svc.msgStream.Send(msg); err != nil {
		trace.PrintError(err)
		return
	}
}

func NewService() *Service {
	// service
	svc := &Service{
		Internal: plugin.NewInternal(),
		colS:     mongo2.GetMongoCol(constants.DependencySettingsColName),
		colD:     mongo2.GetMongoCol(constants.DependenciesColName),
		colT:     mongo2.GetMongoCol(constants.DependencyTasksColName),
		colL:     mongo2.GetMongoCol(constants.DependencyLogsColName),
	}

	// dependency injection
	c := dig.New()
	if err := c.Provide(config.NewNodeConfigService); err != nil {
		panic(err)
	}
	if err := c.Invoke(func(
		cfgSvc interfaces.NodeConfigService,
	) {
		svc.cfgSvc = cfgSvc
	}); err != nil {
		panic(err)
	}

	// sub services
	svc.settingSvc = NewSettingService(svc)
	svc.taskSvc = NewTaskService(svc)
	svc.pythonSvc = NewPythonService(svc)
	svc.nodeSvc = NewNodeService(svc)
	svc.spiderSvc = NewSpiderService(svc)

	// initialize
	if err := svc.Init(); err != nil {
		panic(err)
	}

	return svc
}
