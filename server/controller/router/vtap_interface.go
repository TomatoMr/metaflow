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

	"github.com/metaflowys/metaflow/server/controller/config"
	"github.com/metaflowys/metaflow/server/controller/service"
)

func VTapInterface(e *gin.Engine, cfg *config.ControllerConfig) {
	e.GET("/v1/vtap-interfaces/", getVTapInterfaces)
}

func getVTapInterfaces(c *gin.Context) {
	args := make(map[string]interface{})
	data, err := service.GetVTapInterfaces(args)
	JsonResponse(c, data, err)
}
