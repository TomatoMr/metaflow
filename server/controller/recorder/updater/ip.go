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

package updater

import (
	cloudmodel "github.com/metaflowys/metaflow/server/controller/cloud/model"
	"github.com/metaflowys/metaflow/server/controller/common"
	"github.com/metaflowys/metaflow/server/controller/recorder/cache"
	rcommon "github.com/metaflowys/metaflow/server/controller/recorder/common"
)

type IP struct {
	cache        *cache.Cache
	cloudData    []cloudmodel.IP
	wanIPUpdater *WANIP
	lanIPUpdater *LANIP
}

func NewIP(cache *cache.Cache, cloudData []cloudmodel.IP) *IP {
	return &IP{
		cache:     cache,
		cloudData: cloudData,
	}
}

func (i *IP) HandleAddAndUpdate() {
	wanCloudData, lanCloudData := i.splitToWANAndLAN(i.cloudData)
	i.wanIPUpdater = NewWANIP(i.cache, wanCloudData)
	i.lanIPUpdater = NewLANIP(i.cache, lanCloudData)
	i.wanIPUpdater.HandleAddAndUpdate()
	i.lanIPUpdater.HandleAddAndUpdate()
}

func (i *IP) HandleDelete() {
	i.wanIPUpdater.HandleDelete()
	i.lanIPUpdater.HandleDelete()
}

func (i *IP) splitToWANAndLAN(cloudData []cloudmodel.IP) ([]cloudmodel.IP, []cloudmodel.IP) {
	wanCloudData := []cloudmodel.IP{}
	lanCloudData := []cloudmodel.IP{}
	for _, cloudItem := range cloudData {
		vinterface, exists := i.cache.VInterfaces[cloudItem.VInterfaceLcuuid]
		if !exists {
			log.Error(resourceAForResourceBNotFound(
				rcommon.RESOURCE_TYPE_VINTERFACE_EN, cloudItem.VInterfaceLcuuid,
				rcommon.RESOURCE_TYPE_LAN_IP_EN, cloudItem.Lcuuid,
			))
			continue
		}
		if vinterface.Type == common.VIF_TYPE_WAN {
			wanCloudData = append(wanCloudData, cloudItem)
		} else {
			lanCloudData = append(lanCloudData, cloudItem)
		}
	}
	return wanCloudData, lanCloudData
}
