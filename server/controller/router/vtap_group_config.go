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

	"github.com/metaflowys/metaflow/server/controller/model"
	"github.com/metaflowys/metaflow/server/controller/service"
)

func VTapGroupConfigRouter(e *gin.Engine) {
	e.POST("/v1/vtap-group-configuration/", createVTapGroupConfig)
	e.DELETE("/v1/vtap-group-configuration/:lcuuid/", deleteVTapGroupConfig)
	e.PATCH("/v1/vtap-group-configuration/:lcuuid/", updateVTapGroupConfig)
	e.GET("/v1/vtap-group-configuration/", getVTapGroupConfigs)
	e.GET("/v1/vtap-group-configuration/detailed/:lcuuid/", getVTapGroupDetailedConfig)
	e.POST("/v1/vtap-group-configuration/advanced/", createVTapGroupAdvancedConfig)
	e.GET("/v1/vtap-group-configuration/advanced/:lcuuid/", getVTapGroupAdvancedConfig)
	e.GET("/v1/vtap-group-configuration/advanced/", getVTapGroupAdvancedConfigs)
	e.PATCH("/v1/vtap-group-configuration/advanced/:lcuuid/", updateVTapGroupAdvancedConfig)
	e.GET("/v1/vtap-group-configuration/example/", getVTapGroupExampleConfig)

	e.GET("/v1/vtap-group-configuration/filter/", getVTapGroupConfigByFilter)
	e.DELETE("/v1/vtap-group-configuration/filter/", deleteVTapGroupConfigByFilter)
}

func createVTapGroupConfig(c *gin.Context) {
	vTapGroupConfig := &model.VTapGroupConfiguration{}
	c.ShouldBindBodyWith(&vTapGroupConfig, binding.JSON)
	data, err := service.CreateVTapGroupConfig(vTapGroupConfig)
	JsonResponse(c, data, err)
}

func deleteVTapGroupConfig(c *gin.Context) {
	lcuuid := c.Param("lcuuid")
	data, err := service.DeleteVTapGroupConfig(lcuuid)
	JsonResponse(c, data, err)
}

func updateVTapGroupConfig(c *gin.Context) {
	lcuuid := c.Param("lcuuid")
	vTapGroupConfig := &model.VTapGroupConfiguration{}
	c.ShouldBindBodyWith(&vTapGroupConfig, binding.JSON)
	data, err := service.UpdateVTapGroupConfig(lcuuid, vTapGroupConfig)
	JsonResponse(c, data, err)
}

func getVTapGroupConfigs(c *gin.Context) {
	args := make(map[string]interface{})
	if value, ok := c.GetQuery("vtap_group_id"); ok {
		args["vtap_group_id"] = value
	}
	data, err := service.GetVTapGroupConfigs(args)
	JsonResponse(c, data, err)
}

func getVTapGroupDetailedConfig(c *gin.Context) {
	lcuuid := c.Param("lcuuid")
	data, err := service.GetVTapGroupDetailedConfig(lcuuid)
	JsonResponse(c, data, err)
}

func getVTapGroupAdvancedConfig(c *gin.Context) {
	lcuuid := c.Param("lcuuid")
	data, err := service.GetVTapGroupAdvancedConfig(lcuuid)
	JsonResponse(c, data, err)
}

func updateVTapGroupAdvancedConfig(c *gin.Context) {
	lcuuid := c.Param("lcuuid")
	vTapGroupConfig := &model.VTapGroupConfiguration{}
	c.ShouldBindBodyWith(&vTapGroupConfig, binding.YAML)
	data, err := service.UpdateVTapGroupAdvancedConfig(lcuuid, vTapGroupConfig)
	JsonResponse(c, data, err)
}

func createVTapGroupAdvancedConfig(c *gin.Context) {
	vTapGroupConfig := &model.VTapGroupConfiguration{}
	c.ShouldBindBodyWith(&vTapGroupConfig, binding.YAML)
	data, err := service.CreateVTapGroupAdvancedConfig(vTapGroupConfig)
	JsonResponse(c, data, err)
}

func getVTapGroupConfigByFilter(c *gin.Context) {
	args := make(map[string]string)
	if value, ok := c.GetQuery("vtap_group_id"); ok {
		args["vtap_group_id"] = value
	}
	data, err := service.GetVTapGroupConfigByFilter(args)
	JsonResponse(c, data, err)
}

func deleteVTapGroupConfigByFilter(c *gin.Context) {
	args := make(map[string]string)
	if value, ok := c.GetQuery("vtap_group_id"); ok {
		args["vtap_group_id"] = value
	}
	data, err := service.DeleteVTapGroupConfigByFilter(args)
	JsonResponse(c, data, err)
}

func getVTapGroupExampleConfig(c *gin.Context) {
	data, err := service.GetVTapGroupExampleConfig()
	JsonResponse(c, data, err)
}

func getVTapGroupAdvancedConfigs(c *gin.Context) {
	data, err := service.GetVTapGroupAdvancedConfigs()
	JsonResponse(c, data, err)
}
