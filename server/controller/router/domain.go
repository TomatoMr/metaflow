/*
 * Copyright (c) 2022 Yunshan Networks
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package router

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/metaflowys/metaflow/server/controller/common"
	"github.com/metaflowys/metaflow/server/controller/model"
	"github.com/metaflowys/metaflow/server/controller/service"
)

func DomainRouter(e *gin.Engine) {
	// TODO: 后续统一为v2
	e.GET("/v2/domains/:lcuuid/", getDomain)
	e.GET("/v2/domains/", getDomains)
	e.POST("/v1/domains/", createDomain)
	e.PATCH("/v1/domains/:lcuuid/", updateDomain)
	e.DELETE("/v1/domains/:lcuuid/", deleteDomain)

	e.GET("/v2/sub-domains/:lcuuid/", getSubDomain)
	e.GET("/v2/sub-domains/", getSubDomains)
	e.POST("/v2/sub-domains/", createSubDomain)
	e.PATCH("/v2/sub-domains/:lcuuid/", updateSubDomain)
	e.DELETE("/v2/sub-domains/:lcuuid/", deleteSubDomain)
}

func getDomain(c *gin.Context) {
	args := make(map[string]interface{})
	args["lcuuid"] = c.Param("lcuuid")
	data, err := service.GetDomains(args)
	JsonResponse(c, data, err)
}

func getDomains(c *gin.Context) {
	args := make(map[string]interface{})
	if value, ok := c.GetQuery("name"); ok {
		args["name"] = value
	}
	data, err := service.GetDomains(args)
	JsonResponse(c, data, err)
}

func createDomain(c *gin.Context) {
	var err error
	var domainCreate model.DomainCreate

	// 参数校验
	err = c.ShouldBindBodyWith(&domainCreate, binding.JSON)
	if err != nil {
		BadRequestResponse(c, common.INVALID_POST_DATA, err.Error())
		return
	}

	data, err := service.CreateDomain(domainCreate)
	JsonResponse(c, data, err)
}

func updateDomain(c *gin.Context) {
	var err error
	var domainUpdate model.DomainUpdate

	// 参数校验
	err = c.ShouldBindBodyWith(&domainUpdate, binding.JSON)
	if err != nil {
		BadRequestResponse(c, common.INVALID_PARAMETERS, err.Error())
		return
	}

	// 接收参数
	// 避免struct会有默认值，这里转为map作为函数入参
	patchMap := map[string]interface{}{}
	c.ShouldBindBodyWith(&patchMap, binding.JSON)

	lcuuid := c.Param("lcuuid")
	data, err := service.UpdateDomain(lcuuid, patchMap)
	JsonResponse(c, data, err)
}

func deleteDomain(c *gin.Context) {
	var err error

	lcuuid := c.Param("lcuuid")
	data, err := service.DeleteDomain(lcuuid)
	JsonResponse(c, data, err)
}

func getSubDomain(c *gin.Context) {
	args := make(map[string]interface{})
	args["lcuuid"] = c.Param("lcuuid")
	data, err := service.GetSubDomains(args)
	JsonResponse(c, data, err)
}

func getSubDomains(c *gin.Context) {
	args := make(map[string]interface{})
	if value, ok := c.GetQuery("domain"); ok {
		args["domain"] = value
	}
	data, err := service.GetSubDomains(args)
	JsonResponse(c, data, err)
}

func createSubDomain(c *gin.Context) {
	var err error
	var subDomainCreate model.SubDomainCreate

	// 参数校验
	err = c.ShouldBindBodyWith(&subDomainCreate, binding.JSON)
	if err != nil {
		BadRequestResponse(c, common.INVALID_POST_DATA, err.Error())
		return
	}

	data, err := service.CreateSubDomain(subDomainCreate)
	JsonResponse(c, data, err)
}

func deleteSubDomain(c *gin.Context) {
	var err error

	lcuuid := c.Param("lcuuid")
	data, err := service.DeleteSubDomain(lcuuid)
	JsonResponse(c, data, err)
}

func updateSubDomain(c *gin.Context) {
	var err error
	var subDomainUpdate model.SubDomainUpdate

	// 参数校验
	err = c.ShouldBindBodyWith(&subDomainUpdate, binding.JSON)
	if err != nil {
		BadRequestResponse(c, common.INVALID_PARAMETERS, err.Error())
		return
	}

	// 接收参数
	// 避免struct会有默认值，这里转为map作为函数入参
	patchMap := map[string]interface{}{}
	c.ShouldBindBodyWith(&patchMap, binding.JSON)

	lcuuid := c.Param("lcuuid")
	data, err := service.UpdateSubDomain(lcuuid, patchMap)
	JsonResponse(c, data, err)
}
