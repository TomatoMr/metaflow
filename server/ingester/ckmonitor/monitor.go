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

package ckmonitor

import (
	"fmt"
	"strings"
	"time"

	logging "github.com/op/go-logging"

	"database/sql"

	"github.com/metaflowys/metaflow/server/ingester/common"
	"github.com/metaflowys/metaflow/server/ingester/config"
)

var log = logging.MustGetLogger("monitor")

const (
	DFDiskPrefix   = "path_" // clickhouse的config.xml配置文件中，deepflow写入数据的disk名称以‘path_’开头
	DFS3DiskPrefix = "s3"    // clickhouse的config.xml配置文件中，deepflow写入数据的对象存储disk名称以‘s3’开头
)

type Monitor struct {
	checkInterval              int
	freeSpaceThreshold         int
	usedPercentThreshold       int
	primaryConn, secondaryConn *sql.DB
	primaryAddr, secondaryAddr string
	username, password         string
	exit                       bool
}

type DiskInfo struct {
	name, path                           string
	freeSpace, totalSpace, keepFreeSpace uint64
}

type Partition struct {
	partition, database, table string
	minTime, maxTime           time.Time
	rows, bytesOnDisk          uint64
}

func NewCKMonitor(cfg *config.CKDiskMonitor, primaryAddr, secondaryAddr, username, password string) (*Monitor, error) {
	m := &Monitor{
		checkInterval:        cfg.CheckInterval,
		usedPercentThreshold: cfg.UsedPercent,
		freeSpaceThreshold:   cfg.FreeSpace << 30, // GB
		primaryAddr:          primaryAddr,
		secondaryAddr:        secondaryAddr,
		username:             username,
		password:             password,
	}
	var err error
	m.primaryConn, err = common.NewCKConnection(primaryAddr, username, password)
	if err != nil {
		return nil, err
	}

	if secondaryAddr != "" {
		m.secondaryConn, err = common.NewCKConnection(secondaryAddr, username, password)
		if err != nil {
			return nil, err
		}
	}

	return m, nil
}

func (m *Monitor) updateConnection(connect *sql.DB, addr string) *sql.DB {
	if addr == "" {
		return nil
	}

	if connect == nil || connect.Ping() != nil {
		if connect != nil {
			connect.Close()
		}
		connectNew, err := common.NewCKConnection(addr, m.username, m.password)
		if err != nil {
			log.Warning(err)
		}
		return connectNew
	}
	return connect
}

// 如果clickhouse重启等，需要自动更新连接
func (m *Monitor) updateConnections() {
	m.primaryConn = m.updateConnection(m.primaryConn, m.primaryAddr)
	m.secondaryConn = m.updateConnection(m.secondaryConn, m.secondaryAddr)
}

func getDFDiskInfos(connect *sql.DB) ([]*DiskInfo, bool, error) {
	hasS3Disk := false
	rows, err := connect.Query("SELECT name,path,free_space,total_space,keep_free_space FROM system.disks")
	if err != nil {
		return nil, false, err
	}

	diskInfos := []*DiskInfo{}
	for rows.Next() {
		var (
			name, path                           string
			freeSpace, totalSpace, keepFreeSpace uint64
		)
		err := rows.Scan(&name, &path, &freeSpace, &totalSpace, &keepFreeSpace)
		if err != nil {
			return nil, false, nil
		}
		log.Debugf("name: %s, path: %s, freeSpace: %d, totalSpace: %d, keepFreeSpace: %d", name, path, freeSpace, totalSpace, keepFreeSpace)
		// deepflow的数据, 写入`path_` 开头的disk下
		if strings.HasPrefix(name, DFDiskPrefix) {
			diskInfos = append(diskInfos, &DiskInfo{name, path, freeSpace, totalSpace, keepFreeSpace})
		} else if strings.HasPrefix(name, DFS3DiskPrefix) {
			hasS3Disk = true
		}
	}
	if len(diskInfos) == 0 {
		return nil, hasS3Disk, fmt.Errorf("can not find any deepflow data disk like '%s'", DFDiskPrefix)
	}
	return diskInfos, hasS3Disk, nil
}

func (m *Monitor) isDiskNeedClean(diskInfo *DiskInfo) bool {
	if diskInfo.totalSpace == 0 {
		return false
	}

	usage := ((diskInfo.totalSpace-diskInfo.freeSpace)*100 + diskInfo.totalSpace - 1) / diskInfo.totalSpace
	if usage > uint64(m.usedPercentThreshold) && diskInfo.freeSpace < uint64(m.freeSpaceThreshold) {
		log.Infof("disk usage is over %d. disk name: %s, path: %s, total space: %d, free space: %d, usage: %d",
			m.usedPercentThreshold, diskInfo.name, diskInfo.path, diskInfo.totalSpace, diskInfo.freeSpace, usage)
		return true
	}
	return false
}

// 当所有磁盘都要满足清理条件时，才清理数据
func (m *Monitor) isDisksNeedClean(diskInfos []*DiskInfo) bool {
	if len(diskInfos) == 0 {
		return false
	}

	for _, diskInfo := range diskInfos {
		if !m.isDiskNeedClean(diskInfo) {
			return false
		}
	}
	log.Warningf("disk free space is not enough, will do drop or move partitions.")
	return true
}

func getMinPartitions(connect *sql.DB) ([]Partition, error) {
	sql := fmt.Sprintf("SELECT min(partition),count(distinct partition),database,table,min(min_time),max(max_time),sum(rows),sum(bytes_on_disk) FROM system.parts WHERE disk_name LIKE '%s' and active=1 GROUP BY database,table ORDER BY database,table ASC",
		DFDiskPrefix+"%")
	rows, err := connect.Query(sql)
	if err != nil {
		return nil, err
	}
	partitions := []Partition{}
	for rows.Next() {
		var (
			partition, database, table   string
			minTime, maxTime             time.Time
			rowCount, bytesOnDisk, count uint64
		)
		err := rows.Scan(&partition, &count, &database, &table, &minTime, &maxTime, &rowCount, &bytesOnDisk)
		if err != nil {
			return nil, err
		}
		log.Debugf("partition: %s, count: %d, database: %s, table: %s, minTime: %s, maxTime: %s, rows: %d, bytesOnDisk: %d", partition, count, database, table, minTime, maxTime, rowCount, bytesOnDisk)
		// 只删除partition数量2个以上的partition中最小的一个
		if count > 1 {
			partitions = append(partitions, Partition{partition, database, table, minTime, maxTime, rowCount, bytesOnDisk})
		}
	}
	return partitions, nil
}

func (m *Monitor) dropMinPartitions(connect *sql.DB) error {
	partitions, err := getMinPartitions(connect)
	if err != nil {
		return err
	}

	for _, p := range partitions {
		sql := fmt.Sprintf("ALTER TABLE %s.%s DROP PARTITION '%s'", p.database, p.table, p.partition)
		log.Warningf("drop partition: %s, database: %s, table: %s, minTime: %s, maxTime: %s, rows: %d, bytesOnDisk: %d", p.partition, p.database, p.table, p.minTime, p.maxTime, p.rows, p.bytesOnDisk)
		_, err := connect.Exec(sql)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Monitor) moveMinPartitions(connect *sql.DB) error {
	partitions, err := getMinPartitions(connect)
	if err != nil {
		return err
	}

	for _, p := range partitions {
		sql := fmt.Sprintf("ALTER TABLE %s.%s MOVE PARTITION '%s' TO VOLUME '%s'", p.database, p.table, p.partition, config.DefaultCKDBS3Volume)
		log.Warningf("move partition: %s, database: %s, table: %s, minTime: %s, maxTime: %s, rows: %d, bytesOnDisk: %d", p.partition, p.database, p.table, p.minTime, p.maxTime, p.rows, p.bytesOnDisk)
		_, err := connect.Exec(sql)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Monitor) Start() {
	go m.start()
}

func (m *Monitor) start() {
	counter := 0
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for !m.exit {
		<-ticker.C
		counter++
		if counter%m.checkInterval != 0 {
			continue
		}

		m.updateConnections()
		for _, connect := range []*sql.DB{m.primaryConn, m.secondaryConn} {
			if connect == nil {
				continue
			}
			diskInfos, hasS3Disk, err := getDFDiskInfos(connect)
			if err != nil {
				log.Warning(err)
				continue
			}
			if m.isDisksNeedClean(diskInfos) {
				if hasS3Disk {
					if err := m.moveMinPartitions(connect); err != nil {
						log.Warning("move partition failed.", err)
					}
				} else {
					if err := m.dropMinPartitions(connect); err != nil {
						log.Warning("drop partition failed.", err)
					}
				}
			}
		}
	}
}

func (m *Monitor) Close() error {
	m.exit = true
	return nil
}
