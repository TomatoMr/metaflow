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

package qingcloud

import (
	"encoding/json"
	"strings"

	"github.com/metaflowys/metaflow/server/controller/cloud/model"
	"github.com/metaflowys/metaflow/server/controller/common"
)

func (q *QingCloud) GetSubDomains() ([]model.SubDomain, error) {
	var retSubDomains []model.SubDomain

	log.Debug("get sub_domains starting")

	for regionId := range q.RegionIdToLcuuid {
		kwargs := []*Param{
			{"zone", regionId},
			{"service", "qke"},
			{"status.1", "active"},
		}
		response, err := q.GetResponse("DescribeClusters", "cluster_set", kwargs)
		if err != nil {
			log.Error(err)
			return nil, err
		}

		for _, r := range response {
			for i := range r.MustArray() {
				cluster := r.GetIndex(i)
				clusterId := cluster.Get("cluster_id").MustString()
				if clusterId == "" {
					continue
				}
				// 针对私有云的特殊处理，私有云API会返回其他类型的集群信息，仅对接KubeSphere
				appInfo := cluster.Get("app_info").MustString()
				if appInfo != "" && !strings.Contains(appInfo, "KubeSphere") {
					continue
				}

				vpcLcuuid, _ := q.regionIdToDefaultVPCLcuuid[regionId]
				vpcRouterId := cluster.Get("vxnet").Get("vpc_router_id").MustString()
				if vpcRouterId != "" {
					vpcLcuuid = common.GenerateUUID(vpcRouterId)
				}

				config := map[string]string{
					"vpc_uuid":        vpcLcuuid,
					"cluster_id":      clusterId,
					"port_name_regex": common.DEFAULT_PORT_NAME_REGEX,
					"vtap_id":         "",
					"controller_ip":   "",
				}
				configJson, _ := json.Marshal(config)
				retSubDomains = append(retSubDomains, model.SubDomain{
					Lcuuid:      common.GenerateUUID(clusterId),
					Name:        cluster.Get("name").MustString(),
					DisplayName: clusterId,
					ClusterID:   clusterId,
					VpcUUID:     vpcLcuuid,
					Config:      string(configJson),
				})
			}
		}
	}

	log.Debug("get sub_domains complete")
	return retSubDomains, nil

}
