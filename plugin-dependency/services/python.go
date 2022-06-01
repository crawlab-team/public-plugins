package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/crawlab-team/crawlab-core/controllers"
	"github.com/crawlab-team/go-trace"
	"github.com/crawlab-team/plugin-dependency/constants"
	"github.com/crawlab-team/plugin-dependency/entity"
	"github.com/crawlab-team/plugin-dependency/models"
	"github.com/gin-gonic/gin"
	"github.com/imroc/req"
	"go.mongodb.org/mongo-driver/bson"
	mongo2 "go.mongodb.org/mongo-driver/mongo"
	"net/url"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

type PythonService struct {
	*baseService
}

func (svc *PythonService) Init() {
	svc.api.GET("/python", svc.getList)
	svc.api.POST("/python/update", svc.update)
	svc.api.POST("/python/install", svc.install)
	svc.api.POST("/python/uninstall", svc.uninstall)
}

func (svc *PythonService) GetRepoList(c *gin.Context) {
	// query
	query := c.Query("query")
	pagination := controllers.MustGetPagination(c)

	// validate
	if query == "" {
		controllers.HandleErrorBadRequest(c, errors.New("empty query"))
		return
	}

	// request session
	reqSession := req.New()

	// set timeout
	reqSession.SetTimeout(15 * time.Second)

	// user agent
	ua := req.Header{"user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.61 Safari/537.36"}

	// request url
	requestUrl := fmt.Sprintf("https://pypi.org/search?page=%d&q=%s", pagination.Page, url.QueryEscape(query))

	// perform request
	res, err := reqSession.Get(requestUrl, ua)
	if err != nil {
		if res != nil {
			_, _ = c.Writer.Write(res.Bytes())
			_ = c.AbortWithError(res.Response().StatusCode, err)
			return
		}
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// response bytes
	data, err := res.ToBytes()
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}
	buf := bytes.NewBuffer(data)

	// parse html
	doc, err := goquery.NewDocumentFromReader(buf)
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// dependencies
	var deps []models.Dependency
	var depNames []string
	doc.Find(".left-layout__main > form ul > li").Each(func(i int, s *goquery.Selection) {
		d := models.Dependency{
			Name:          s.Find(".package-snippet__name").Text(),
			LatestVersion: s.Find(".package-snippet__version").Text(),
		}
		deps = append(deps, d)
		depNames = append(depNames, d.Name)
	})

	// total
	totalStr := doc.Find(".left-layout__main .split-layout p > strong").Text()
	totalStr = strings.ReplaceAll(totalStr, ",", "")
	total, _ := strconv.Atoi(totalStr)

	// empty results
	if total == 0 {
		controllers.HandleSuccess(c)
		return
	}

	// dependencies in db
	var depsResults []entity.DependencyResult
	pipelines := mongo2.Pipeline{
		{{
			"$match",
			bson.M{
				"type": constants.DependencyTypePython,
				"name": bson.M{
					"$in": depNames,
				},
			},
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
			},
		}},
		{{
			"$project",
			bson.M{
				"name":     "$_id",
				"node_ids": "$node_ids",
				"versions": "$versions",
			},
		}},
	}
	if err := svc.parent.colD.Aggregate(pipelines, nil).All(&depsResults); err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}

	// dependencies map
	depsResultsMap := map[string]entity.DependencyResult{}
	for _, dr := range depsResults {
		depsResultsMap[dr.Name] = dr
	}

	// iterate dependencies
	for i, d := range deps {
		dr, ok := depsResultsMap[d.Name]
		if ok {
			deps[i].Result = dr
		}
	}

	controllers.HandleSuccessWithListData(c, deps, total)
}

func (svc *PythonService) GetDependencies(params entity.UpdateParams) (deps []models.Dependency, err error) {
	cmd := exec.Command(params.Cmd, "list", "--format", "json")
	data, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var _deps []models.Dependency
	if err := json.Unmarshal(data, &_deps); err != nil {
		return nil, err
	}
	for _, d := range _deps {
		if strings.HasPrefix(d.Name, "-") {
			continue
		}
		d.Type = constants.DependencyTypePython
		deps = append(deps, d)
	}
	return deps, nil
}

func (svc *PythonService) InstallDependencies(params entity.InstallParams) (err error) {
	// arguments
	var args []string

	// install
	args = append(args, "install")

	// proxy
	if params.Proxy != "" {
		args = append(args, "-i")
		args = append(args, params.Proxy)
	}

	if params.UseConfig {
		// workspace path
		workspacePath, err := svc._getInstallWorkspacePath(params)
		if err != nil {
			return err
		}

		// config path
		configPath := path.Join(workspacePath, constants.DependencyConfigRequirementsTxt)

		// use config
		args = append(args, "-r")
		args = append(args, configPath)
	} else {
		// upgrade
		if params.Upgrade {
			args = append(args, "-U")
		}

		// dependency names
		for _, depName := range params.Names {
			args = append(args, depName)
		}
	}

	// command
	cmd := exec.Command(params.Cmd, args...)

	// logging
	svc.parent._configureLogging(params.TaskId, cmd)

	// start
	if err := cmd.Start(); err != nil {
		return trace.TraceError(err)
	}

	// wait
	if err := cmd.Wait(); err != nil {
		return trace.TraceError(err)
	}

	return nil
}

func (svc *PythonService) UninstallDependencies(params entity.UninstallParams) (err error) {
	// arguments
	var args []string

	// uninstall
	args = append(args, "uninstall")
	args = append(args, "-y")

	// dependency names
	for _, depName := range params.Names {
		args = append(args, depName)
	}

	// command
	cmd := exec.Command(params.Cmd, args...)

	// logging
	svc.parent._configureLogging(params.TaskId, cmd)

	// start
	if err := cmd.Start(); err != nil {
		return trace.TraceError(err)
	}

	// wait
	if err := cmd.Wait(); err != nil {
		return trace.TraceError(err)
	}

	return nil
}

func (svc *PythonService) GetLatestVersion(dep models.Dependency) (v string, err error) {
	// not exists in cache, request from pypi
	reqSession := req.New()

	// set timeout
	reqSession.SetTimeout(60 * time.Second)

	// user agent
	ua := req.Header{"user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.61 Safari/537.36"}

	// request url
	requestUrl := fmt.Sprintf("https://pypi.org/project/%s/", dep.Name)

	// perform request
	res, err := reqSession.Get(requestUrl, ua)
	if err != nil {
		return "", trace.TraceError(err)
	}

	// response bytes
	data, err := res.ToBytes()
	if err != nil {
		return "", trace.TraceError(err)
	}
	buf := bytes.NewBuffer(data)

	// parse html
	doc, err := goquery.NewDocumentFromReader(buf)
	if err != nil {
		return "", trace.TraceError(err)
	}

	// latest version
	v = doc.Find(".release-timeline .release--current .release__version").Text()
	v = strings.TrimSpace(v)

	return v, nil
}

func NewPythonService(parent *Service) (svc *PythonService) {
	svc = &PythonService{}
	baseSvc := newBaseService(
		svc,
		parent,
		constants.DependencyTypePython,
		entity.MessageCodes{
			Update:    constants.MessageCodePythonUpdate,
			Save:      constants.MessageCodePythonSave,
			Install:   constants.MessageCodePythonInstall,
			Uninstall: constants.MessageCodePythonUninstall,
		},
	)
	svc.baseService = baseSvc
	return svc
}
