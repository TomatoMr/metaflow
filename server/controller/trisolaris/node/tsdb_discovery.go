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

package node

import (
	"sync"

	"github.com/google/uuid"
	"github.com/metaflowys/metaflow/message/trident"

	. "github.com/metaflowys/metaflow/server/controller/common"
	models "github.com/metaflowys/metaflow/server/controller/db/mysql"
	. "github.com/metaflowys/metaflow/server/controller/trisolaris/common"
)

type TSDBDiscovery struct {
	sync.Mutex
	registration map[string]*models.Analyzer
}

func newTSDBDiscovery() *TSDBDiscovery {
	return &TSDBDiscovery{
		registration: make(map[string]*models.Analyzer),
	}
}

func (a *TSDBDiscovery) register(request *trident.SyncRequest) {
	pcapDataMountPath := ""
	if request.GetTsdbReportInfo() != nil {
		pcapDataMountPath = request.GetTsdbReportInfo().GetPcapDataMountPath()
	}
	tsdb := &models.Analyzer{
		IP:                request.GetCtrlIp(),
		NATIPEnabled:      0,
		NATIP:             "",
		Name:              request.GetHost(),
		CPUNum:            int(request.GetCpuNum()),
		MemorySize:        int64(request.GetMemorySize()),
		Arch:              request.GetArch(),
		Os:                request.GetOs(),
		KernelVersion:     request.GetKernelVersion(),
		VTapMax:           TSDB_VTAP_MAX,
		State:             HOST_STATE_COMPLETE,
		Lcuuid:            uuid.NewString(),
		PcapDataMountPath: pcapDataMountPath,
	}
	a.Lock()
	defer a.Unlock()
	a.registration[request.GetCtrlIp()] = tsdb
}

func (a *TSDBDiscovery) getRegisterData() map[string]*models.Analyzer {
	a.Lock()
	defer a.Unlock()
	data := a.registration
	a.registration = make(map[string]*models.Analyzer)
	return data
}
