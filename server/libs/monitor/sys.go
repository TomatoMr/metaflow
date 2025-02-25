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

package monitor

import (
	"os"

	"github.com/op/go-logging"
	"github.com/shirou/gopsutil/process"

	"github.com/metaflowys/metaflow/server/libs/stats"
)

var log = logging.MustGetLogger("monitor")

type SysCounter struct {
	CpuPercent float64 `statsd:"cpu_percent,gauge"`
	Memory     uint64  `statsd:"memory,gauge"` // physical in bytes
}

type Monitor process.Process

func (m *Monitor) GetCounter() interface{} {
	percent, err := (*process.Process)(m).Percent(0)
	if err != nil {
		return SysCounter{}
	}
	mem, err := (*process.Process)(m).MemoryInfo()
	if err != nil {
		return SysCounter{}
	}
	return SysCounter{percent, mem.RSS}
}

func (m *Monitor) Closed() bool {
	return false
}

func init() {
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		log.Errorf("%v", err)
		return
	}
	m := (*Monitor)(proc)
	stats.RegisterCountable("monitor", m)
}
