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

package platform

import (
	"errors"
	"fmt"

	logging "github.com/op/go-logging"

	"github.com/metaflowys/metaflow/server/controller/cloud/aliyun"
	"github.com/metaflowys/metaflow/server/controller/cloud/baidubce"
	"github.com/metaflowys/metaflow/server/controller/cloud/config"
	"github.com/metaflowys/metaflow/server/controller/cloud/genesis"
	"github.com/metaflowys/metaflow/server/controller/cloud/kubernetes"
	"github.com/metaflowys/metaflow/server/controller/cloud/model"
	"github.com/metaflowys/metaflow/server/controller/cloud/qingcloud"
	"github.com/metaflowys/metaflow/server/controller/common"
	"github.com/metaflowys/metaflow/server/controller/db/mysql"
)

var log = logging.MustGetLogger("cloud.platform")

type Platform interface {
	CheckAuth() error
	GetCloudData() (model.Resource, error)
}

func NewPlatform(domain mysql.Domain, cfg config.CloudConfig) (Platform, error) {
	var platform Platform
	var err error

	switch domain.Type {
	case common.ALIYUN:
		platform, err = aliyun.NewAliyun(domain)
	case common.AGENT_SYNC:
		platform, err = genesis.NewGenesis(domain, cfg)
	case common.QINGCLOUD:
		platform, err = qingcloud.NewQingCloud(domain)
	case common.BAIDU_BCE:
		platform, err = baidubce.NewBaiduBce(domain)
	case common.KUBERNETES:
		platform, err = kubernetes.NewKubernetes(domain)
	// TODO: other platform
	default:
		return nil, errors.New(fmt.Sprintf("domain type (%d) not supported", domain.Type))
	}
	return platform, err
}
