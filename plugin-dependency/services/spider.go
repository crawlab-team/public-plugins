package services

import (
	"errors"
	"fmt"
	"github.com/blang/semver/v4"
	"github.com/crawlab-team/crawlab-core/controllers"
	"github.com/crawlab-team/crawlab-core/spider/fs"
	"github.com/crawlab-team/crawlab-core/utils"
	"github.com/crawlab-team/go-trace"
	"github.com/crawlab-team/plugin-dependency/constants"
	"github.com/crawlab-team/plugin-dependency/entity"
	"github.com/crawlab-team/plugin-dependency/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongo2 "go.mongodb.org/mongo-driver/mongo"
	"io/ioutil"
	"path"
	"regexp"
	"strings"
)

type SpiderService struct {
	parent *Service
	api    *gin.Engine
}

func (svc *SpiderService) Init() {
	svc.api.GET("/spiders/:id", svc.get)
	svc.api.POST("/spiders/:id/install", svc.install)
	svc.api.POST("/spiders/:id/uninstall", svc.uninstall)
}

func (svc *SpiderService) install(c *gin.Context) {
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

	// dependency type
	dependencyType := svc._getDependencyType(workspacePath)

	// install
	switch dependencyType {
	case constants.DependencyConfigRequirementsTxt:
		svc.parent.pythonSvc.install(c)
	case constants.DependencyConfigPackageJson:
		svc.parent.nodeSvc.install(c)
	default:
		controllers.HandleErrorInternalServerError(c, errors.New(fmt.Sprintf("invalid dependency type: %s", dependencyType)))
		return
	}
}

func (svc *SpiderService) uninstall(c *gin.Context) {
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

	// dependency type
	dependencyType := svc._getDependencyType(workspacePath)

	// uninstall
	switch dependencyType {
	case constants.DependencyConfigRequirementsTxt:
		svc.parent.pythonSvc.uninstall(c)
	case constants.DependencyConfigPackageJson:
		svc.parent.nodeSvc.uninstall(c)
	default:
		controllers.HandleErrorInternalServerError(c, errors.New(fmt.Sprintf("invalid dependency type: %s", dependencyType)))
		return
	}
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

	// dependency type
	dependencyType := svc._getDependencyType(workspacePath)

	// spider info
	info := bson.M{}
	info["dependency_type"] = dependencyType

	// dependencies
	var dependencies []models.Dependency
	switch dependencyType {
	case constants.DependencyConfigRequirementsTxt:
		dependencies, err = svc._getDependenciesRequirementsTxt(workspacePath)
	case constants.DependencyConfigPackageJson:
		// TODO: implement
	}
	if err != nil {
		controllers.HandleErrorInternalServerError(c, err)
		return
	}
	info["dependencies"] = dependencies

	controllers.HandleSuccessWithData(c, info)
}

func (svc *SpiderService) _getDependencyType(workspacePath string) (t string) {
	if utils.Exists(path.Join(workspacePath, constants.DependencyConfigRequirementsTxt)) {
		return constants.DependencyConfigRequirementsTxt
	} else if utils.Exists(path.Join(workspacePath, constants.DependencyConfigPackageJson)) {
		return constants.DependencyConfigPackageJson
	}
	return ""
}

func (svc *SpiderService) _getDependenciesRequirementsTxt(workspacePath string) (deps []models.Dependency, err error) {
	// file path
	filePath := path.Join(workspacePath, constants.DependencyConfigRequirementsTxt)

	// file content
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, trace.TraceError(err)
	}
	content := string(data)

	// regex pattern
	pattern, err := regexp.Compile("(\\w+)([=<>]=.*)?")
	if err != nil {
		return nil, trace.TraceError(err)
	}

	// dependency names
	var depNames []string

	// iterate content lines
	for _, line := range strings.Split(content, "\n") {
		// trim space
		line = strings.TrimSpace(line)

		// validate regex match result
		if !pattern.MatchString(line) {
			return nil, trace.TraceError(errors.New(fmt.Sprintf("invalid %s", constants.DependencyConfigRequirementsTxt)))
		}

		// sub matches
		matches := pattern.FindStringSubmatch(line)
		if len(matches) < 2 {
			return nil, trace.TraceError(errors.New(fmt.Sprintf("invalid %s", constants.DependencyConfigRequirementsTxt)))
		}

		// dependency result
		d := models.Dependency{
			Name: matches[1],
		}

		// required version
		if len(matches) > 2 {
			d.Version = matches[2]
		}

		// add to dependency names
		depNames = append(depNames, d.Name)

		// add to dependencies
		deps = append(deps, d)
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
		return nil, err
	}

	// dependencies map
	depsResultsMap := map[string]entity.DependencyResult{}
	for _, dr := range depsResults {
		depsResultsMap[dr.Name] = dr
	}

	// iterate dependencies
	for i, d := range deps {
		// operator
		var op string
		if strings.HasPrefix(d.Version, "==") {
			op = "=="
			deps[i].Version = strings.Replace(d.Version, "==", "", 1)
		} else if strings.HasPrefix(d.Version, ">=") {
			op = ">="
		} else if strings.HasPrefix(d.Version, "<=") {
			op = "<="
		}

		// dependency result
		dr, ok := depsResultsMap[d.Name]
		if !ok {
			continue
		}
		deps[i].Result = dr

		// normalized version
		nv := strings.Replace(d.Version, op, "", 1)

		// required version
		rv, err := semver.Make(nv)
		if err != nil {
			continue
		}

		// iterate installed versions
		for _, v := range dr.Versions {
			// installed version
			iv, err := semver.Make(v)
			if err != nil {
				continue
			}

			// compare with the required version
			switch op {
			case "==":
				if rv.Compare(iv) > 0 {
					deps[i].Result.Upgradable = true
				} else if rv.Compare(iv) < 0 {
					deps[i].Result.Downgradable = true
				}
			case ">=":
				if rv.Compare(iv) > 0 {
					deps[i].Result.Upgradable = true
				}
			case "<=":
				if rv.Compare(iv) < 0 {
					deps[i].Result.Downgradable = true
				}
			}
		}
	}

	return deps, nil
}

func NewSpiderService(parent *Service) (svc *SpiderService) {
	svc = &SpiderService{
		parent: parent,
		api:    parent.GetApi(),
	}
	return svc
}
