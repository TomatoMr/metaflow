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

package common

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"inet.af/netaddr"

	"github.com/bitly/go-simplejson"
	mapset "github.com/deckarep/golang-set"
	"github.com/mikioh/ipaddr"
	logging "github.com/op/go-logging"
	uuid "github.com/satori/go.uuid"

	"github.com/metaflowys/metaflow/server/controller/cloud/config"
	"github.com/metaflowys/metaflow/server/controller/cloud/model"
	"github.com/metaflowys/metaflow/server/controller/common"
	"github.com/metaflowys/metaflow/server/controller/db/mysql"
)

var log = logging.MustGetLogger("cloud.common")

func StringStringMapKeys(m map[string]string) (keys []string) {
	for k := range m {
		keys = append(keys, k)
	}
	return
}

func StringInterfaceMapKeys(m map[string]interface{}) (keys []string) {
	for k := range m {
		keys = append(keys, k)
	}
	return
}

func StringInterfaceMapKVs(m map[string]interface{}, sep string) (items []string) {
	keys := []string{}
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, k := range keys {
		newString := k + sep + m[k].(string)
		items = append(items, newString)
	}
	return
}

func StringSliceStringMapKeys(m map[string][]string) (keys []string) {
	for k := range m {
		keys = append(keys, k)
	}
	return
}

func StringStringMapValues(m map[string]string) (values []string) {
	for k := range m {
		values = append(values, m[k])
	}
	return
}

func UnionMapStringInt(m, n map[string]int) map[string]int {
	for k, v := range n {
		m[k] = v
	}
	return m
}

func UnionMapStringString(m, n map[string]string) map[string]string {
	for k, v := range n {
		m[k] = v
	}
	return m
}

func UnionMapStringSet(m, n map[string]mapset.Set) map[string]mapset.Set {
	for k, v := range n {
		if _, ok := m[k]; !ok {
			m[k] = v
		} else {
			m[k] = m[k].Union(v)
		}
	}
	return m
}

func ReadJSONFile(path string) (*simplejson.Json, error) {
	jsonFile, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("read json file error:" + err.Error())
	}
	js, err := simplejson.NewJson([]byte(jsonFile))

	if err != nil {
		return nil, errors.New("initialization simplejson error:" + err.Error())
	}
	return js, nil
}

func ReadLineJSONFile(path string) (js []*simplejson.Json, err error) {
	jsFile, oErr := os.Open(path)
	defer jsFile.Close()
	if oErr != nil {
		err = oErr
		return
	}
	buf := bufio.NewReader(jsFile)
	for {
		lineFile, _, eof := buf.ReadLine()
		if eof == io.EOF {
			break
		}
		lineJs, sErr := simplejson.NewJson(lineFile)
		if sErr != nil {
			err = sErr
			return
		}
		js = append(js, lineJs)
	}
	return
}

func GenerateIPMask(ip string) int {
	netO, err := netaddr.ParseIPPrefix(ip)
	if err == nil {
		maskLen, _ := netO.IPNet().Mask.Size()
		return maskLen
	}
	if strings.Contains(ip, ":") {
		return common.IPV6_MAX_MASK
	}
	return common.IPV4_MAX_MASK
}

func IPAndMaskToCIDR(ip string, mask int) (string, error) {
	ipO, err := netaddr.ParseIP(ip)
	if err != nil {
		return "", errors.New("ip and mask to cidr ip format error:" + err.Error())
	}
	IPString := ipO.String() + "/" + strconv.Itoa(mask)
	netO, err := netaddr.ParseIPPrefix(IPString)
	if err != nil {
		return "", errors.New("ip and mask to cidr format error" + err.Error())
	}
	netRange, ok := netO.Range().Prefix()
	if !ok {
		return "", errors.New("ip and mask to cidr format not valid")
	}
	return netRange.String(), nil
}

func TidyIPString(ipsString []string) (v4Prefix, v6Prefix []netaddr.IPPrefix, err error) {
	for _, ipS := range ipsString {
		_, ignoreErr := netaddr.ParseIPPrefix(ipS)
		if ignoreErr != nil {
			switch {
			case strings.Contains(ipS, "."):
				ipS = ipS + "/32"
			case strings.Contains(ipS, ":"):
				ipS = ipS + "/128"
			}
		}
		ipPrefix, prefixErr := netaddr.ParseIPPrefix(ipS)
		if prefixErr != nil {
			err = prefixErr
			return
		}
		switch {
		case ipPrefix.IP().Is4():
			v4Prefix = append(v4Prefix, ipPrefix)
		case ipPrefix.IP().Is6():
			v6Prefix = append(v6Prefix, ipPrefix)
		}
	}
	return
}

func AggregateCIDR(ips []netaddr.IPPrefix, maxMask int) (cirdsString []string) {
	CIDRs := []*ipaddr.Prefix{}
	for _, Prefix := range ips {
		aggFlag := false
		ipNet := ipaddr.NewPrefix(Prefix.IPNet())
		for i, CIDR := range CIDRs {
			pSlice := []ipaddr.Prefix{*ipNet, *CIDR}
			aggCIDR := ipaddr.Supernet(pSlice)
			if aggCIDR == nil {
				continue
			}
			aggCIDRMask, _ := aggCIDR.IPNet.Mask.Size()
			if aggCIDRMask >= maxMask {
				CIDRs[i] = aggCIDR
				aggFlag = true
				break
			} else {
				continue
			}
		}
		if !aggFlag {
			CIDRs = append(CIDRs, ipNet)
		}
	}
	for _, i := range CIDRs {
		cirdsString = append(cirdsString, i.String())
	}
	return
}

func IsIPInCIDR(ip, cidr string) bool {
	if strings.Contains(cidr, "/") {
		_, nCIDR, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Errorf("parse cidr failed: %v", err)
			return false
		}
		return nCIDR.Contains(net.ParseIP(ip))
	} else {
		if ip == cidr {
			return true
		}
		return false
	}
}

// 针对各私有云平台，每个区域生成一个基础VPC和子网
// 宿主机及管理节点的接口和IP属于基础VPC和子网
func GetBasicVPCLcuuid(uuidGenerate, regionLcuuid string) string {
	return common.GenerateUUID(uuidGenerate + regionLcuuid)
}

func GetBasicNetworkLcuuid(vpcLcuuid string) string {
	return common.GenerateUUID(vpcLcuuid)
}

func GetBasicVPCAndNetworks(regions []model.Region, domainName, uuidGenerate string) ([]model.VPC, []model.Network) {
	var retVPCs []model.VPC
	var retNetworks []model.Network

	for _, region := range regions {
		vpcLcuuid := GetBasicVPCLcuuid(uuidGenerate, region.Lcuuid)
		vpcName := domainName + fmt.Sprintf("%s_基础VPC_%s", domainName, region.Name)
		retVPCs = append(retVPCs, model.VPC{
			Lcuuid:       vpcLcuuid,
			Name:         vpcName,
			RegionLcuuid: region.Lcuuid,
		})
		retNetworks = append(retNetworks, model.Network{
			Lcuuid:         GetBasicNetworkLcuuid(vpcLcuuid),
			Name:           vpcName + "子网",
			SegmentationID: 1,
			NetType:        common.NETWORK_TYPE_LAN,
			VPCLcuuid:      vpcLcuuid,
			RegionLcuuid:   region.Lcuuid,
		})
	}

	return retVPCs, retNetworks
}

// 根据采集器上报的接口信息，生成宿主机的接口和IP信息
func GetHostNics(hosts []model.Host, domainName, uuidGenerate, portNameRegex string, excludeIPs []string) (
	[]model.Subnet, []model.VInterface, []model.IP, map[string][]model.Subnet, error,
) {
	var retSubnets []model.Subnet
	var retVInterfaces []model.VInterface
	var retIPs []model.IP

	// 查询数据库获取采集器，生成采集器ctrl_ip到launch_server的对应关系
	vtaps := []mysql.VTap{}
	mysql.Db.Find(&vtaps)

	vtapLaunchServerToCtrlIP := make(map[string]string)
	for _, vtap := range vtaps {
		vtapLaunchServerToCtrlIP[vtap.LaunchServer] = vtap.CtrlIP
	}

	// 调用genesis API获取vinterfaces
	// TODO: genesis重构后，修改为内部函数调用
	response, err := common.CURLPerform(
		"GET", "http://genesis:20015/v1/vinterfaces", map[string]interface{}{},
	)
	if err != nil {
		log.Errorf("call genesis vinterfaces api failed: (%s)", err.Error())
		return nil, nil, nil, nil, err
	}
	// 获取hostIP与vinterfaces的对应关系
	hostIPToVInterfaces := make(map[string][]*simplejson.Json)
	for i := range response.Get("DATA").MustArray() {
		r := response.Get("DATA").GetIndex(i)
		if r.Get("DEVICE_TYPE").MustString() != "kvm-host" {
			continue
		}
		hostIP := r.Get("HOST_IP").MustString()
		hostIPToVInterfaces[hostIP] = append(hostIPToVInterfaces[hostIP], r)
	}

	var reg *regexp.Regexp
	if portNameRegex != "" {
		reg, _ = regexp.Compile(portNameRegex)
	}
	// 遍历宿主机生成网段、接口和IP信息
	vpcLcuuidToSubnets := make(map[string][]model.Subnet)
	for _, host := range hosts {
		vinterfaces, ok := hostIPToVInterfaces[host.IP]
		if !ok {
			continue
		}
		vpcLcuuid := GetBasicVPCLcuuid(uuidGenerate, host.RegionLcuuid)
		networkLcuuid := GetBasicNetworkLcuuid(vpcLcuuid)
		subnets, ok := vpcLcuuidToSubnets[vpcLcuuid]
		if !ok {
			subnets = []model.Subnet{}
		}

		// 遍历采集器上报的宿主机接口列表
		// 额外对接路由接口为空 或者 不匹配额外对接路由接口时，跳过该接口
		includeHostIP := false
		for _, vinterface := range vinterfaces {
			vinterfaceName := vinterface.Get("NAME").MustString()
			if reg == nil || reg.MatchString(vinterfaceName) {
				continue
			}
			mac := vinterface.Get("MAC").MustString()
			vinterfaceLcuuid := common.GenerateUUID(host.Lcuuid + mac)

			for i := range vinterface.Get("IPS").MustArray() {
				ip := vinterface.Get("IPS").GetIndex(i).MustString()
				if ip == host.IP {
					includeHostIP = true
				}

				subnetLcuuid := ""
				ipMasks := strings.Split(ip, "/")
				ipAddr := netaddr.IP{}
				ipMask := strconv.Itoa(common.IPV4_MAX_MASK)
				if strings.Contains(ip, ":") {
					ipMask = strconv.Itoa(common.IPV6_MAX_MASK)
				}
				if len(ipMasks) > 1 {
					ipAddr, _ = netaddr.ParseIP(ipMasks[0])
					ipMask = ipMasks[1]
				}
				// 判断是否在excludeIPs；如果是，则跳过
				IsExcludeIP := false
				for _, excludeIP := range excludeIPs {
					if IsIPInCIDR(ipMasks[0], excludeIP) {
						IsExcludeIP = true
						break
					}
				}
				if IsExcludeIP {
					continue
				}

				// 判断IP + 掩码信息是否已经在当前网段中；如果不在，则生成新的网段信息
				for _, subnet := range subnets {
					subnetCidr, _ := netaddr.ParseIPPrefix(subnet.CIDR)
					if subnetCidr.Contains(ipAddr) {
						subnetLcuuid = subnet.Lcuuid
						break
					}
				}
				if subnetLcuuid == "" {
					cidrParse, _ := ipaddr.Parse(ip)
					subnetCidr := cidrParse.First().IP.String() + "/" + ipMask
					subnetLcuuid = common.GenerateUUID(networkLcuuid + subnetCidr)
					retSubnet := model.Subnet{
						Lcuuid:        subnetLcuuid,
						Name:          subnetCidr,
						CIDR:          subnetCidr,
						NetworkLcuuid: networkLcuuid,
						VPCLcuuid:     vpcLcuuid,
					}
					retSubnets = append(retSubnets, retSubnet)
					vpcLcuuidToSubnets[vpcLcuuid] = append(
						vpcLcuuidToSubnets[vpcLcuuid], retSubnet,
					)
				}

				// 增加IP信息
				retIPs = append(retIPs, model.IP{
					Lcuuid:           common.GenerateUUID(vinterfaceLcuuid + ipMasks[0]),
					VInterfaceLcuuid: vinterfaceLcuuid,
					IP:               ipMasks[0],
					SubnetLcuuid:     subnetLcuuid,
					RegionLcuuid:     host.RegionLcuuid,
				})
			}
			// 增加接口信息
			retVInterfaces = append(retVInterfaces, model.VInterface{
				Lcuuid:        vinterfaceLcuuid,
				Type:          common.VIF_TYPE_LAN,
				Mac:           mac,
				DeviceType:    common.VIF_DEVICE_TYPE_HOST,
				DeviceLcuuid:  host.Lcuuid,
				NetworkLcuuid: networkLcuuid,
				VPCLcuuid:     vpcLcuuid,
				RegionLcuuid:  host.RegionLcuuid,
			})
		}

		// 如果vinterface中没有返回hostIP，则使用全0的MAC生成接口和IP信息
		if includeHostIP {
			continue
		}
		// 判断IP是否已经在当前网段中；如果不在，则生成新的网段信息
		ipAddr, _ := netaddr.ParseIP(host.IP)
		subnetLcuuid := ""
		for _, subnet := range subnets {
			subnetCidr, _ := netaddr.ParseIPPrefix(subnet.CIDR)
			if subnetCidr.Contains(ipAddr) {
				subnetLcuuid = subnet.Lcuuid
				break
			}
		}
		if subnetLcuuid == "" {
			ipMask := strconv.Itoa(common.IPV4_DEFAULT_NETMASK)
			if strings.Contains(host.IP, ":") {
				ipMask = strconv.Itoa(common.IPV6_DEFAULT_NETMASK)
			}
			cidrParse, _ := ipaddr.Parse(host.IP + "/" + ipMask)
			subnetCidr := cidrParse.First().IP.String() + "/" + ipMask
			subnetLcuuid = common.GenerateUUID(networkLcuuid + subnetCidr)
			retSubnet := model.Subnet{
				Lcuuid:        subnetLcuuid,
				Name:          subnetCidr,
				CIDR:          subnetCidr,
				NetworkLcuuid: networkLcuuid,
				VPCLcuuid:     vpcLcuuid,
			}
			retSubnets = append(retSubnets, retSubnet)
			vpcLcuuidToSubnets[vpcLcuuid] = append(
				vpcLcuuidToSubnets[vpcLcuuid], retSubnet,
			)
		}

		// 增加接口和IP信息
		mac := common.VIF_DEFAULT_MAC
		vinterfaceLcuuid := common.GenerateUUID(host.Lcuuid + mac)
		retVInterfaces = append(retVInterfaces, model.VInterface{
			Lcuuid:        vinterfaceLcuuid,
			Type:          common.VIF_TYPE_LAN,
			Mac:           mac,
			DeviceType:    common.VIF_DEVICE_TYPE_HOST,
			DeviceLcuuid:  host.Lcuuid,
			NetworkLcuuid: networkLcuuid,
			VPCLcuuid:     vpcLcuuid,
			RegionLcuuid:  host.RegionLcuuid,
		})
		retIPs = append(retIPs, model.IP{
			Lcuuid:           common.GenerateUUID(vinterfaceLcuuid + host.IP),
			VInterfaceLcuuid: vinterfaceLcuuid,
			IP:               host.IP,
			SubnetLcuuid:     subnetLcuuid,
			RegionLcuuid:     host.RegionLcuuid,
		})
	}

	return retSubnets, retVInterfaces, retIPs, vpcLcuuidToSubnets, nil
}

func EliminateEmptyRegions(regions []model.Region, regionLcuuidToResourceNum map[string]int) []model.Region {
	var retRegions []model.Region

	for _, region := range regions {
		resourceNum := 0
		resourceNum, ok := regionLcuuidToResourceNum[region.Lcuuid]
		if !ok || resourceNum == 0 {
			continue
		}
		retRegions = append(retRegions, region)
	}
	return retRegions
}

func EliminateEmptyAZs(azs []model.AZ, azLcuuidToResourceNum map[string]int) []model.AZ {
	var retAZs []model.AZ

	for _, az := range azs {
		resourceNum := 0
		resourceNum, ok := azLcuuidToResourceNum[az.Lcuuid]
		if !ok || resourceNum == 0 {
			continue
		}
		retAZs = append(retAZs, az)
	}
	return retAZs
}

// 根据主机名获取主机IP
// 不同方式优先级：DNS > file > Hash
func GetHostIPByName(name string) (string, error) {
	if config.CONF.DNSEnable {
		ips, err := net.LookupIP(name) // TODO 是否需要自定义err
		if err == nil {
			return ips[0].String(), err
		} else {
			log.Errorf("lookup for hostname: %s failed: %v", name, err)
		}
	}

	// TODO 将此文件内容持久化，避免每次都重新读取
	f, err := os.Open(config.CONF.HostnameToIPFile)
	if err == nil {
		defer f.Close()

		csvReader := csv.NewReader(f)
		lines, err := csvReader.ReadAll()
		if err == nil {
			for _, line := range lines {
				if len(line) != 2 {
					continue
				}
				if line[0] == name {
					return line[1], nil
				}
			}
		} else {
			log.Errorf("read file: %s failed: %v", config.CONF.HostnameToIPFile, err)
		}
	} else {
		log.Errorf("open file: %s failed: %v", config.CONF.HostnameToIPFile, err)
	}

	// TODO hash实现
	return "", nil
}

func GetAZLcuuidFromUUIDGenerate(uuidGenerate string) string {
	lcuuid := common.GetUUID(uuidGenerate, uuid.Nil)
	return lcuuid[:len(lcuuid)-2] + "ff"
}
