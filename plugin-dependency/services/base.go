package services

import (
	"encoding/json"
	"errors"
	"github.com/blang/semver/v4"
	constants2 "github.com/crawlab-team/crawlab-core/constants"
	"github.com/crawlab-team/crawlab-core/controllers"
	"github.com/crawlab-team/crawlab-core/interfaces"
	models2 "github.com/crawlab-team/crawlab-core/models/models"
	"github.com/crawlab-team/crawlab-core/spider/fs"
	"github.com/crawlab-team/crawlab-db/mongo"
	grpc "github.com/crawlab-team/crawlab-grpc"
	"github.com/crawlab-team/go-trace"
	"github.com/crawlab-team/plugin-dependency/constants"
	"github.com/crawlab-team/plugin-dependency/entity"
	"github.com/crawlab-team/plugin-dependency/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongo2 "go.mongodb.org/mongo-driver/mongo"
	"strconv"
	"sync"
	"time"
)

type baseService struct {
	svc        DependencyService
	parent     *Service
	api        *gin.Engine
	chMap      sync.Map
	s          models.Setting
	key        string
	codes      entity.MessageCodes
	vCache     sync.Map
	defaultCmd string
}

func (svc *baseService) Start() {
	// wait for message stream to be ready
	for {
		if svc.parent.msgStream != nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	// update
	if err := svc._update(); err != nil {
		trace.PrintError(err)
	}
}

func (svc *baseService) getList(c *gin.Context) {
	installed, _ := strconv.ParseBool(c.Query("installed"))
	if installed {
		svc._getInstalledList(c)
	} else {
		svc.svc.GetRepoList(c)
	}
}

func (svc *baseService) update(c *gin.Context) {
	if err := svc._update(); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}
	controllers.HandleSuccess(c)
}

func (svc *baseService) install(c *gin.Context) {
	// payload
	var payload entity.InstallPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// setting
	if err := svc._getSetting(); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// nodes
	query := bson.M{}
	if payload.Mode == constants.InstallModeAll {
		query["active"] = true
	} else {
		query["node_id"] = bson.M{"$in": payload.NodeIds}
	}
	nodes, err := svc.parent._getNodes(query)
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// wait group
	wg := sync.WaitGroup{}
	wg.Add(len(nodes))

	// iterate nodes
	for _, n := range nodes {
		// task
		t := &models.Task{
			Id:        primitive.NewObjectID(),
			Status:    constants2.TaskStatusRunning,
			SettingId: svc.s.Id,
			Type:      svc.key,
			NodeId:    n.Id,
			DepNames:  payload.Names,
			Action:    constants.ActionInstall,
			UpdateTs:  time.Now(),
		}
		if _, err := svc.parent.colT.Insert(t); err != nil {
			controllers.HandleErrorInternalServerError(c, err)
			return
		}

		// params
		params := &entity.InstallParams{
			TaskId:    t.Id,
			Upgrade:   payload.Upgrade,
			Names:     payload.Names,
			Proxy:     svc.s.Proxy,
			Cmd:       svc._getCmd(),
			UseConfig: payload.UseConfig,
			SpiderId:  payload.SpiderId,
		}

		// message data
		data, _ := json.Marshal(params)
		msgDataObj := &entity.MessageData{
			Code: svc.codes.Install,
			Data: data,
		}
		msgData, _ := json.Marshal(msgDataObj)

		// stream message
		msg := &grpc.StreamMessage{
			Code:    grpc.StreamMessageCode_SEND,
			NodeKey: svc.parent.currentNode.GetKey(),
			From:    "plugin:" + svc.parent.currentNode.GetKey(),
			To:      "plugin:" + n.GetKey(),
			Data:    msgData,
		}

		// send message
		if err := svc.parent.msgStream.Send(msg); err != nil {
			trace.PrintError(err)
			wg.Done()
			return
		}
	}

	controllers.HandleSuccess(c)
}

func (svc *baseService) uninstall(c *gin.Context) {
	// payload
	var payload entity.UninstallPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// setting
	if err := svc._getSetting(); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// node model service
	nodeModelSvc, err := svc.parent.GetModelService().NewBaseServiceDelegate(interfaces.ModelIdNode)
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// dependencies
	var deps []models.Dependency
	query := bson.M{
		"type": svc.key,
		"name": bson.M{"$in": payload.Names},
	}
	if err := svc.parent.colD.Find(query, nil).All(&deps); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// nodeMap
	nodeMap := map[primitive.ObjectID]interfaces.Node{}

	// dependencies by node id
	depNamesNodeMap := map[interfaces.Node][]string{}
	for _, d := range deps {
		// node
		n, ok := nodeMap[d.NodeId]
		if !ok {
			doc, err := nodeModelSvc.GetById(d.NodeId)
			if err != nil {
				controllers.HandleErrorInternalServerError(c, err)
				return
			}
			n, _ = doc.(interfaces.Node)
		}

		// skip if not active
		if !n.GetActive() {
			continue
		}

		// add to map
		_, ok = depNamesNodeMap[n]
		if !ok {
			depNamesNodeMap[n] = []string{}
		}
		depNamesNodeMap[n] = append(depNamesNodeMap[n], d.Name)
	}

	// iterate map
	for n, depNames := range depNamesNodeMap {
		// task
		t := &models.Task{
			Id:        primitive.NewObjectID(),
			Status:    constants2.TaskStatusRunning,
			SettingId: svc.s.Id,
			Type:      svc.key,
			NodeId:    n.GetId(),
			DepNames:  depNames,
			Action:    constants.ActionUninstall,
			UpdateTs:  time.Now(),
		}
		if _, err := svc.parent.colT.Insert(t); err != nil {
			controllers.HandleErrorInternalServerError(c, err)
			return
		}

		// params
		params := &entity.UninstallParams{
			TaskId: t.Id,
			Cmd:    svc._getCmd(),
			Names:  depNames,
		}

		// data
		data, err := json.Marshal(params)
		if err != nil {
			controllers.HandleErrorInternalServerError(c, err)
			return
		}

		// message data
		msgDataObj := &entity.MessageData{
			Code: svc.codes.Uninstall,
			Data: data,
		}
		msgData, err := json.Marshal(msgDataObj)
		if err != nil {
			controllers.HandleErrorInternalServerError(c, err)
			return
		}

		// stream message
		msg := &grpc.StreamMessage{
			Code:    grpc.StreamMessageCode_SEND,
			NodeKey: svc.parent.currentNode.GetKey(),
			From:    "plugin:" + svc.parent.currentNode.GetKey(),
			To:      "plugin:" + n.GetKey(),
			Data:    msgData,
		}

		// send message
		if err := svc.parent.msgStream.Send(msg); err != nil {
			controllers.HandleErrorInternalServerError(c, err)
			return
		}
	}
}

func (svc *baseService) _getInstalledList(c *gin.Context) {
	// params
	searchQuery := c.Query("query")
	pagination := controllers.MustGetPagination(c)

	// query
	query := bson.M{}
	query["type"] = svc.key
	if searchQuery != "" {
		query["name"] = primitive.Regex{
			Pattern: searchQuery,
			Options: "i",
		}
	}

	// base pipelines
	basePipelines := mongo2.Pipeline{
		{{
			"$match",
			query,
		}},
		{{
			"$group",
			bson.M{
				"_id": "$name",
				"node_ids": bson.M{
					"$push": "$node_id",
				},
				"versions": bson.M{
					"$addToSet": "$version",
				},
				"latest_version": bson.M{
					"$first": "$latest_version",
				},
			},
		}},
		{{
			"$project",
			bson.M{
				"name":           "$_id",
				"node_ids":       "$node_ids",
				"versions":       "$versions",
				"latest_version": "$latest_version",
			},
		}},
		{{"$sort", bson.D{{"name", 1}}}},
	}

	// dependency results
	var depsResults []entity.DependencyResult
	pipelines := basePipelines
	pipelines = append(pipelines, bson.D{{"$skip", (pagination.Page - 1) * pagination.Size}})
	pipelines = append(pipelines, bson.D{{"$limit", pagination.Size}})
	if err := svc.parent.colD.Aggregate(pipelines, nil).All(&depsResults); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// iterate dependency results
	for i, dr := range depsResults {
		// skip if the latest version does not exist
		if dr.LatestVersion == "" {
			continue
		}

		// latest version
		lv, err := semver.Make(dr.LatestVersion)
		if err != nil {
			continue
		}

		for _, v := range dr.Versions {
			// current version
			cv, err := semver.Make(v)
			if err != nil {
				continue
			}

			// compare with the latest version
			if lv.Compare(cv) > 0 {
				depsResults[i].Upgradable = true
				break
			}
		}
	}

	// total
	var depsResultTotal entity.DependencyResult
	pipelinesTotal := basePipelines
	pipelinesTotal = append(pipelinesTotal, bson.D{{
		"$group",
		bson.M{
			"_id":   nil,
			"count": bson.M{"$sum": 1},
		},
	}})
	if err := svc.parent.colD.Aggregate(pipelinesTotal, nil).One(&depsResultTotal); err != nil {
		if err.Error() != mongo2.ErrNoDocuments.Error() {
			controllers.HandleErrorInternalServerError(c, err)
			return
		}
	}
	total := depsResultTotal.Count

	controllers.HandleSuccessWithListData(c, depsResults, total)
}

func (svc *baseService) _update() (err error) {
	// setting
	if err := svc._getSetting(); err != nil {
		return err
	}

	// nodes
	nodes, err := svc.parent._getNodes(bson.M{"active": true})
	if err != nil {
		return err
	}

	// wait group
	wg := sync.WaitGroup{}
	wg.Add(len(nodes))

	// iterate nodes
	for _, n := range nodes {
		go func(n models2.Node) {
			// notify channel
			ch := svc._getDefaultCh()

			// params data
			data, _ := json.Marshal(&entity.UpdateParams{
				Cmd: svc._getCmd(),
			})

			// message data
			msgDataBytes, _ := json.Marshal(&entity.MessageData{
				Code: svc.codes.Update,
				Data: data,
			})

			// message
			msg := &grpc.StreamMessage{
				Code:    grpc.StreamMessageCode_SEND,
				NodeKey: svc.parent.currentNode.GetKey(),
				From:    "plugin:" + svc.parent.currentNode.GetKey(),
				To:      "plugin:" + n.GetKey(),
				Data:    msgDataBytes,
			}

			// send message
			if err := svc.parent.msgStream.Send(msg); err != nil {
				trace.PrintError(err)
				wg.Done()
				return
			}

			// wait
			<-ch
			wg.Done()
		}(n)
	}

	// wait for all nodes to finish
	wg.Wait()

	// update latest version
	go svc._updateDependenciesLatestVersion()

	return nil
}

// updateDependencyList get dependency list on local node and
// send them to master node
func (svc *baseService) updateDependencyList(msg *grpc.StreamMessage, msgData entity.MessageData) {
	// params
	var params entity.UpdateParams
	if err := json.Unmarshal(msgData.Data, &params); err != nil {
		trace.PrintError(err)
		return
	}

	// installed dependencies
	deps, err := svc.svc.GetDependencies(params)
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
		Code: svc.codes.Save,
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
		NodeKey: svc.parent.currentNode.GetKey(),
		From:    "plugin:" + svc.parent.currentNode.GetKey(),
		To:      "plugin:" + svc.parent.masterNode.GetKey(),
		Data:    msgDataBytes,
	}

	// send message
	if err := svc.parent.msgStream.Send(msg); err != nil {
		trace.PrintError(err)
		return
	}
}

func (svc *baseService) _saveDependencyList(msg *grpc.StreamMessage, msgData entity.MessageData) {
	// notify channel
	ch := svc._getDefaultCh()

	// dependencies
	var deps []models.Dependency
	if err := json.Unmarshal(msgData.Data, &deps); err != nil {
		trace.PrintError(err)
		ch <- true
		return
	}

	// installed dependency names
	var depNames []string
	for _, d := range deps {
		depNames = append(depNames, d.Name)
	}

	// node model service
	nodeModelSvc, err := svc.parent.GetModelService().NewBaseServiceDelegate(interfaces.ModelIdNode)
	if err != nil {
		trace.PrintError(err)
		ch <- true
		return
	}

	// node
	doc, err := nodeModelSvc.Get(bson.M{"key": msg.NodeKey}, nil)
	if err != nil {
		trace.PrintError(err)
		ch <- true
		return
	}
	n, ok := doc.(interfaces.Node)
	if !ok {
		trace.PrintError(errors.New("invalid type"))
		ch <- true
		return
	}

	// run transaction to update dependencies
	err = mongo.RunTransaction(func(ctx mongo2.SessionContext) (err error) {
		// remove non-existing dependencies
		if err := svc.parent.colD.Delete(bson.M{
			"type":    svc.key,
			"node_id": n.GetId(),
			"name":    bson.M{"$nin": depNames},
		}); err != nil {
			return err
		}

		// existing dependencies
		query := bson.M{
			"type":    svc.key,
			"node_id": n.GetId(),
		}
		var depsDb []models.Dependency
		if err := svc.parent.colD.Find(query, nil).All(&depsDb); err != nil {
			return err
		}
		depsDbMap := map[string]models.Dependency{}
		for _, d := range depsDb {
			depsDbMap[d.Name] = d
		}

		// new dependencies
		var depsNew []interface{}
		for _, d := range deps {
			if _, ok := depsDbMap[d.Name]; !ok {
				d.Id = primitive.NewObjectID()
				d.Type = svc.key
				d.NodeId = n.GetId()
				depsNew = append(depsNew, d)
			}
		}

		// skip if no new dependencies
		if len(depsNew) == 0 {
			return
		}

		// add new dependencies
		if _, err := svc.parent.colD.InsertMany(depsNew); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		trace.PrintError(err)
		ch <- true
		return
	}

	// notify update success
	ch <- true
}

func (svc *baseService) installDependency(msg *grpc.StreamMessage, msgData entity.MessageData) {
	// dependencies
	var params entity.InstallParams
	if err := json.Unmarshal(msgData.Data, &params); err != nil {
		trace.PrintError(err)
		svc.parent._sendTaskStatus(params.TaskId, constants2.TaskStatusError, err)
		return
	}

	// install
	if err := svc.svc.InstallDependencies(params); err != nil {
		trace.PrintError(err)
		svc.parent._sendTaskStatus(params.TaskId, constants2.TaskStatusError, err)
		return
	}

	// success
	svc.parent._sendTaskStatus(params.TaskId, constants2.TaskStatusFinished, nil)

	// update dependencies
	svc.updateDependencyList(msg, msgData)
}

func (svc *baseService) uninstallDependency(msg *grpc.StreamMessage, msgData entity.MessageData) {
	// dependencies
	var params entity.UninstallParams
	if err := json.Unmarshal(msgData.Data, &params); err != nil {
		trace.PrintError(err)
		svc.parent._sendTaskStatus(params.TaskId, constants2.TaskStatusError, err)
		return
	}

	// uninstall
	if err := svc.svc.UninstallDependencies(params); err != nil {
		trace.PrintError(err)
		svc.parent._sendTaskStatus(params.TaskId, constants2.TaskStatusError, err)
		return
	}

	// success
	svc.parent._sendTaskStatus(params.TaskId, constants2.TaskStatusFinished, nil)

	// update dependencies
	svc.updateDependencyList(msg, msgData)
}

func (svc *baseService) _updateDependenciesLatestVersion() {
	// dependencies
	var deps []models.Dependency
	query := bson.M{
		"latest_version": bson.M{
			"$exists": false,
		},
	}
	if err := svc.parent.colD.Find(query, nil).All(&deps); err != nil {
		trace.PrintError(err)
		return
	}

	// iterate dependencies
	for _, d := range deps {
		svc._updateDependencyLatestVersion(d)
	}
}

func (svc *baseService) _updateDependencyLatestVersion(dep models.Dependency) {
	// version
	var v string

	// attempt to load from cache
	r, ok := svc.vCache.Load(dep.Name)
	if ok {
		// exists in cache
		v, _ = r.(string)
	} else {
		// version
		var err error
		v, err = svc.svc.GetLatestVersion(dep)
		if err != nil {
			trace.PrintError(err)
			return
		}

		// store in cache
		svc.vCache.Store(dep.Name, v)
	}

	// update
	update := bson.M{
		"$set": bson.M{
			"latest_version": v,
		},
	}
	if err := svc.parent.colD.UpdateId(dep.Id, update); err != nil {
		trace.PrintError(err)
		return
	}
}

func (svc *baseService) _getDefaultCh() (ch chan bool) {
	return svc._getCh(svc.parent.currentNode.GetKey())
}

func (svc *baseService) _getCh(key string) (ch chan bool) {
	res, ok := svc.chMap.Load(key)
	if ok {
		ch, ok := res.(chan bool)
		if ok {
			return ch
		}
	}
	ch = make(chan bool)
	svc.chMap.Store(key, ch)
	return ch
}

func (svc *baseService) _getSetting() (err error) {
	if err := svc.parent.colS.Find(bson.M{"key": svc.key}, nil).One(&svc.s); err != nil {
		return trace.TraceError(err)
	}
	return nil
}

func (svc *baseService) _getCmd() (cmd string) {
	if svc.s.Cmd == "" {
		return svc.defaultCmd
	}
	return svc.s.Cmd
}

func (svc *baseService) _getInstallWorkspacePath(params entity.InstallParams) (workspacePath string, err error) {
	// spider fs service
	fsSvc, err := fs.NewSpiderFsService(params.SpiderId)
	if err != nil {
		return workspacePath, err
	}

	// sync to workspace
	if err := fsSvc.GetFsService().SyncToWorkspace(); err != nil {
		return workspacePath, err
	}

	return fsSvc.GetWorkspacePath(), nil
}

func newBaseService(svc DependencyService, parent *Service, key string, codes entity.MessageCodes) (res *baseService) {
	return &baseService{
		svc:    svc,
		parent: parent,
		api:    parent.GetApi(),
		chMap:  sync.Map{},
		key:    key,
		codes:  codes,
	}
}
