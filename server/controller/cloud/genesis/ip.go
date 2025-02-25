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

package genesis

import (
	"github.com/metaflowys/metaflow/server/controller/cloud/model"
	"github.com/metaflowys/metaflow/server/controller/common"
	"github.com/metaflowys/metaflow/server/controller/genesis"

	uuid "github.com/satori/go.uuid"
)

func (g *Genesis) getIPs() ([]model.IP, error) {
	log.Debug("get ips starting")
	ips := []model.IP{}
	ipsData := genesis.GenesisService.GetIPsData()

	g.cloudStatsd.APICost["ips"] = []int{0}
	g.cloudStatsd.APICount["ips"] = []int{len(ipsData)}

	for _, i := range ipsData {
		if i.VInterfaceLcuuid == "" || i.SubnetLcuuid == "" {
			log.Debug("vinterface lcuuid or subnet lcuuid not found")
			continue
		}
		lcuuid := i.Lcuuid
		if lcuuid == "" {
			lcuuid = common.GetUUID(i.VInterfaceLcuuid+i.IP, uuid.Nil)
		}
		ip := model.IP{
			Lcuuid:           lcuuid,
			VInterfaceLcuuid: i.VInterfaceLcuuid,
			IP:               i.IP,
			SubnetLcuuid:     i.SubnetLcuuid,
			RegionLcuuid:     g.regionUuid,
		}
		ips = append(ips, ip)
	}
	log.Debug("get ips complete")
	return ips, nil
}
