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
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/protobuf/proto"

	models "github.com/metaflowys/metaflow/server/controller/db/mysql"
	"github.com/metaflowys/metaflow/server/controller/trisolaris/utils/atomicbool"
)

type TSDBCache struct {
	syncedAt          *time.Time
	cpuNum            int
	memorySize        int64
	ip                string
	arch              *string
	os                *string
	kernelVersion     *string
	natIP             *string
	pcapDataMountPath *string
	syncFlag          atomicbool.Bool
}

func newTSDBCache(tsdb *models.Analyzer) *TSDBCache {
	syncedAt := tsdb.SyncedAt
	return &TSDBCache{
		syncedAt:          &syncedAt,
		cpuNum:            tsdb.CPUNum,
		memorySize:        tsdb.MemorySize,
		ip:                tsdb.IP,
		arch:              proto.String(tsdb.Arch),
		os:                proto.String(tsdb.Os),
		kernelVersion:     proto.String(tsdb.KernelVersion),
		natIP:             proto.String(tsdb.NATIP),
		pcapDataMountPath: proto.String(tsdb.PcapDataMountPath),
		syncFlag:          atomicbool.NewBool(false),
	}
}

func (c *TSDBCache) GetKey() string {
	return c.ip
}

func (c *TSDBCache) GetArch() string {
	if c.arch != nil {
		return *c.arch
	}

	return ""
}

func (c *TSDBCache) GetOS() string {
	if c.os != nil {
		return *c.os
	}

	return ""
}

func (c *TSDBCache) GetKernelVersion() string {
	if c.kernelVersion != nil {
		return *c.kernelVersion
	}

	return ""
}

func (c *TSDBCache) GetPcapDataMountPath() string {
	if c.pcapDataMountPath != nil {
		return *c.pcapDataMountPath
	}

	return ""
}

func (c *TSDBCache) setSyncFlag() {
	c.syncFlag.Set()
}

func (c *TSDBCache) unsetSyncFlag() {
	c.syncFlag.Unset()
}

func (c *TSDBCache) updateNatIP(natIP string) {
	c.natIP = &natIP
}

func (c *TSDBCache) GetSyncedAt() *time.Time {
	return c.syncedAt
}

func (c *TSDBCache) UpdateSyncedAt(syncedAt time.Time) {
	c.syncedAt = &syncedAt
	c.setSyncFlag()
}

func (c *TSDBCache) UpdateSystemInfo(cpuNum int, memorySize int64, arch string, os string,
	kernelVersion string, pcapDataMountPath string) {

	c.cpuNum = cpuNum
	c.memorySize = memorySize
	c.arch = &arch
	c.os = &os
	c.kernelVersion = &kernelVersion
	c.pcapDataMountPath = &pcapDataMountPath
}

type TSDBCacheMap struct {
	sync.RWMutex
	keyToTSDBCache map[string]*TSDBCache
}

func newTSDBCacheMap() *TSDBCacheMap {
	return &TSDBCacheMap{
		keyToTSDBCache: make(map[string]*TSDBCache),
	}
}

func (m *TSDBCacheMap) Add(tsdbCache *TSDBCache) {
	m.Lock()
	defer m.Unlock()
	m.keyToTSDBCache[tsdbCache.GetKey()] = tsdbCache
}

func (m *TSDBCacheMap) Delete(key string) {
	m.Lock()
	defer m.Unlock()
	delete(m.keyToTSDBCache, key)
}

func (m *TSDBCacheMap) Get(key string) *TSDBCache {
	m.RLock()
	defer m.RUnlock()
	if vTapCache, ok := m.keyToTSDBCache[key]; ok {
		return vTapCache
	}

	return nil
}

func (m *TSDBCacheMap) List() []string {
	m.RLock()
	defer m.RUnlock()
	keys := make([]string, 0, len(m.keyToTSDBCache))
	for key, _ := range m.keyToTSDBCache {
		keys = append(keys, key)
	}

	return keys
}

func (m *TSDBCacheMap) GetKeySet() mapset.Set {
	m.RLock()
	defer m.RUnlock()
	keys := mapset.NewSet()
	for key, _ := range m.keyToTSDBCache {
		keys.Add(key)
	}

	return keys
}
