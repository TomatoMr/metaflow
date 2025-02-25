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

package aliyun

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	simplejson "github.com/bitly/go-simplejson"
	logging "github.com/op/go-logging"

	"github.com/metaflowys/metaflow/server/controller/cloud/common"
	"github.com/metaflowys/metaflow/server/controller/cloud/model"
	"github.com/metaflowys/metaflow/server/controller/db/mysql"
)

var log = logging.MustGetLogger("cloud.aliyun")

type Aliyun struct {
	uuid           string
	uuidGenerate   string
	regionUuid     string
	secretID       string
	secretKey      string
	regionName     string
	includeRegions []string
	excludeRegions []string

	// 以下两个字段的作用：消除公有云的无资源的区域和可用区
	regionLcuuidToResourceNum map[string]int
	azLcuuidToResourceNum     map[string]int
}

func NewAliyun(domain mysql.Domain) (*Aliyun, error) {
	config, err := simplejson.NewJson([]byte(domain.Config))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	secretID, err := config.Get("secret_id").String()
	if err != nil {
		log.Error("secret_id must be specified")
		return nil, err
	}

	secretKey, err := config.Get("secret_key").String()
	if err != nil {
		log.Error("secret_key must be specified")
		return nil, err
	}

	excludeRegionsStr := config.Get("exclude_regions").MustString()
	excludeRegions := []string{}
	if excludeRegionsStr != "" {
		excludeRegions = strings.Split(excludeRegionsStr, ",")
		sort.Strings(excludeRegions)
	}
	includeRegionsStr := config.Get("include_regions").MustString()
	includeRegions := []string{}
	if includeRegionsStr != "" {
		includeRegions = strings.Split(includeRegionsStr, ",")
		sort.Strings(includeRegions)
	}

	return &Aliyun{
		uuid: domain.Lcuuid,
		// TODO: display_name后期需要修改为uuid_generate
		uuidGenerate: domain.DisplayName,
		regionUuid:   config.Get("region_uuid").MustString(),
		secretID:     secretID,
		secretKey:    secretKey,
		// TODO: 后期需要修改为从配置文件读取
		regionName:     "cn-beijing",
		excludeRegions: excludeRegions,
		includeRegions: includeRegions,

		regionLcuuidToResourceNum: make(map[string]int),
		azLcuuidToResourceNum:     make(map[string]int),
	}, nil
}

func (a *Aliyun) CheckAuth() error {
	_, err := sdk.NewClientWithAccessKey(a.regionName, a.secretID, a.secretKey)
	return err
}

func (a *Aliyun) getResponse(
	client interface{}, request interface{}, funcName string, resultKey string, once bool,
) ([]*simplejson.Json, error) {
	var resp []*simplejson.Json

	pageNum := 1
	pageSize := 50
	totalCount := 0
	for {
		rRequest := reflect.ValueOf(request)
		if !once {
			reflect.ValueOf(request).Elem().FieldByName("PageSize").SetString(strconv.Itoa(pageSize))
			reflect.ValueOf(request).Elem().FieldByName("PageNumber").SetString(strconv.Itoa(pageNum))
		}
		ret := reflect.ValueOf(client).MethodByName(funcName).Call([]reflect.Value{rRequest})

		if ret[1].Interface() != nil {
			return make([]*simplejson.Json, 0), ret[1].Interface().(error)
		}

		rStatus := ret[0].MethodByName("GetHttpStatus").Call([]reflect.Value{})[0]
		if rStatus.Interface().(int) != 200 {
			rContent := ret[0].MethodByName("GetHttpContentString").Call([]reflect.Value{})[0]
			err := errors.New(rContent.Interface().(string))
			return make([]*simplejson.Json, 0), err
		}

		rResult := ret[0].MethodByName("GetHttpContentBytes").Call([]reflect.Value{})

		result, err := simplejson.NewJson(rResult[0].Interface().([]byte))
		if err != nil {
			return make([]*simplejson.Json, 0), err
		}

		if curResp, ok := result.CheckGet(resultKey); ok {
			resp = append(resp, curResp)
		} else {
			break
		}

		if !once {
			pageNum += 1
			totalCount += pageSize
			if totalCount >= result.Get("TotalCount").MustInt() {
				break
			}
		} else {
			break
		}
	}
	return resp, nil
}

func (a *Aliyun) getRegionLcuuid(lcuuid string) string {
	if a.regionUuid != "" {
		return a.regionUuid
	} else {
		return lcuuid
	}
}

func (a *Aliyun) checkRequiredAttributes(json *simplejson.Json, attributes []string) error {
	for _, attribute := range attributes {
		if _, ok := json.CheckGet(attribute); !ok {
			log.Infof("get attribute (%s) failed", attribute)
			return errors.New(fmt.Sprintf("get attribute (%s) failed", attribute))
		}
	}
	return nil
}

func (a *Aliyun) GetCloudData() (model.Resource, error) {
	var resource model.Resource
	var azs []model.AZ
	var vpcs []model.VPC
	var networks []model.Network
	var subnets []model.Subnet
	var vms []model.VM
	var vmSecurityGroups []model.VMSecurityGroup
	var vinterfaces []model.VInterface
	var ips []model.IP
	var floatingIPs []model.FloatingIP
	var securityGroups []model.SecurityGroup
	var securityGroupRules []model.SecurityGroupRule
	var vrouters []model.VRouter
	var routingTables []model.RoutingTable
	var natGateways []model.NATGateway
	var natRules []model.NATRule
	var lbs []model.LB
	var lbListeners []model.LBListener
	var lbTargetServers []model.LBTargetServer
	var redisInstances []model.RedisInstance
	var rdsInstances []model.RDSInstance
	var cens []model.CEN

	regions, err := a.getRegions()
	if err != nil {
		log.Error("get region data failed")
		return resource, err
	}
	for _, region := range regions {
		log.Infof("get region (%s) data starting", region.Name)

		// 可用区
		tmpAZs, err := a.getAZs(region)
		if err != nil {
			log.Errorf("get region (%s) az data failed", region.Name)
			return resource, err
		}
		azs = append(azs, tmpAZs...)

		// VPC
		tmpVPCs, err := a.getVPCs(region)
		if err != nil {
			log.Errorf("get region (%s) vpc data failed", region.Name)
			return resource, err
		}
		vpcs = append(vpcs, tmpVPCs...)

		// 子网及网段
		tmpNetworks, tmpSubnets, err := a.getNetworks(region)
		if err != nil {
			log.Errorf("get region (%s) vpc data failed", region.Name)
			return resource, err
		}
		networks = append(networks, tmpNetworks...)
		subnets = append(subnets, tmpSubnets...)

		// VM及相关资源
		tmpVMs, tmpVMSecurityGroups, tmpVInterfaces, tmpIPs, tmpFloatingIPs, vmLcuuidToVPCLcuuid, err := a.getVMs(region)
		if err != nil {
			log.Errorf("get region (%s) vm data failed", region.Name)
			return resource, err
		}
		vms = append(vms, tmpVMs...)
		vmSecurityGroups = append(vmSecurityGroups, tmpVMSecurityGroups...)
		vinterfaces = append(vinterfaces, tmpVInterfaces...)
		ips = append(ips, tmpIPs...)
		floatingIPs = append(floatingIPs, tmpFloatingIPs...)

		// VM接口信息
		tmpVInterfaces, tmpIPs, tmpFloatingIPs, tmpNATRules, err := a.getVMPorts(region)
		if err != nil {
			log.Errorf("get region (%s) port data failed", region.Name)
			return resource, err
		}
		vinterfaces = append(vinterfaces, tmpVInterfaces...)
		ips = append(ips, tmpIPs...)
		floatingIPs = append(floatingIPs, tmpFloatingIPs...)
		natRules = append(natRules, tmpNATRules...)

		// 安全组及规则
		tmpSecurityGroups, tmpSecurityGroupRules, err := a.getSecurityGroups(region)
		if err != nil {
			log.Errorf("get region (%s) security_group data failed", region.Name)
			return resource, err
		}
		securityGroups = append(securityGroups, tmpSecurityGroups...)
		securityGroupRules = append(securityGroupRules, tmpSecurityGroupRules...)

		// 路由表及规则
		tmpVRouters, tmpRoutingTables, err := a.getRouterAndTables(region)
		if err != nil {
			log.Errorf("get region (%s) router data failed", region.Name)
			return resource, err
		}
		vrouters = append(vrouters, tmpVRouters...)
		routingTables = append(routingTables, tmpRoutingTables...)

		// NAT网关及规则
		tmpNATGateways, tmpNATRules, tmpVInterfaces, tmpIPs, err := a.getNatGateways(region)
		if err != nil {
			log.Errorf("get region (%s) nat_gateway data failed", region.Name)
			return resource, err
		}
		natGateways = append(natGateways, tmpNATGateways...)
		natRules = append(natRules, tmpNATRules...)
		vinterfaces = append(vinterfaces, tmpVInterfaces...)
		ips = append(ips, tmpIPs...)

		// 负载均衡器及规则
		tmpLBs, tmpLBListeners, tmpLBTargetServers, tmpVInterfaces, tmpIPs, err := a.getLoadBalances(region, vmLcuuidToVPCLcuuid)
		if err != nil {
			log.Errorf("get region (%s) load_balance data failed", region.Label)
			return resource, err
		}
		lbs = append(lbs, tmpLBs...)
		lbListeners = append(lbListeners, tmpLBListeners...)
		lbTargetServers = append(lbTargetServers, tmpLBTargetServers...)
		vinterfaces = append(vinterfaces, tmpVInterfaces...)
		ips = append(ips, tmpIPs...)

		// 云企业网
		cens, err = a.getCens(region)
		if err != nil {
			log.Errorf("get region (%s) cen data failed", region.Label)
			return resource, err
		}

		// Redis
		tmpRedisInstances, tmpVInterfaces, tmpIPs, err := a.getRedisInstances(region)
		if err != nil {
			log.Errorf("get region (%s) redis data failed", region.Name)
			return resource, err
		}
		redisInstances = append(redisInstances, tmpRedisInstances...)
		vinterfaces = append(vinterfaces, tmpVInterfaces...)
		ips = append(ips, tmpIPs...)

		// RDS
		tmpRDSInstances, tmpVInterfaces, tmpIPs, err := a.getRDSInstances(region)
		if err != nil {
			log.Errorf("get region (%s) rds data failed", region.Name)
			return resource, err
		}
		rdsInstances = append(rdsInstances, tmpRDSInstances...)
		vinterfaces = append(vinterfaces, tmpVInterfaces...)
		ips = append(ips, tmpIPs...)

		log.Infof("get region (%s) data completed", region.Name)
	}

	resource.Regions = common.EliminateEmptyRegions(regions, a.regionLcuuidToResourceNum)
	resource.AZs = common.EliminateEmptyAZs(azs, a.azLcuuidToResourceNum)
	resource.VPCs = vpcs
	resource.Networks = networks
	resource.Subnets = subnets
	resource.VMs = vms
	resource.VMSecurityGroups = vmSecurityGroups
	resource.VInterfaces = vinterfaces
	resource.IPs = ips
	resource.FloatingIPs = floatingIPs
	resource.SecurityGroups = securityGroups
	resource.SecurityGroupRules = securityGroupRules
	resource.VRouters = vrouters
	resource.RoutingTables = routingTables
	resource.NATGateways = natGateways
	resource.NATRules = natRules
	resource.LBs = lbs
	resource.LBListeners = lbListeners
	resource.LBTargetServers = lbTargetServers
	resource.RedisInstances = redisInstances
	resource.RDSInstances = rdsInstances
	resource.CENs = cens
	return resource, err
}
