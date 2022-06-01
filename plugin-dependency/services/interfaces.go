package services

import (
	"github.com/crawlab-team/plugin-dependency/entity"
	"github.com/crawlab-team/plugin-dependency/models"
	"github.com/gin-gonic/gin"
)

type DependencyService interface {
	Init()
	GetRepoList(c *gin.Context)
	GetDependencies(params entity.UpdateParams) (deps []models.Dependency, err error)
	InstallDependencies(params entity.InstallParams) (err error)
	UninstallDependencies(params entity.UninstallParams) (err error)
	GetLatestVersion(dep models.Dependency) (v string, err error)
}
