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

package service

import (
	"encoding/json"
	"fmt"
	"strings"

	uuid "github.com/satori/go.uuid"
	"gorm.io/gorm/clause"

	cloudcommon "github.com/metaflowys/metaflow/server/controller/cloud/common"
	k8s "github.com/metaflowys/metaflow/server/controller/cloud/kubernetes_gather"
	"github.com/metaflowys/metaflow/server/controller/common"
	"github.com/metaflowys/metaflow/server/controller/db/mysql"
	"github.com/metaflowys/metaflow/server/controller/model"
)

func GetDomains(filter map[string]interface{}) (resp []model.Domain, err error) {
	var response []model.Domain
	var domains []mysql.Domain
	var azs []mysql.AZ
	var subDomains []mysql.SubDomain
	var controllers []mysql.Controller
	var domainLcuuids []string
	var domainToAZLcuuids map[string][]string
	var domainToRegionLcuuidsToAZLcuuids map[string](map[string][]string)
	var controllerIPToName map[string]string

	Db := mysql.Db
	if _, ok := filter["lcuuid"]; ok {
		Db = Db.Where("lcuuid = ?", filter["lcuuid"])
	}
	if _, ok := filter["name"]; ok {
		Db = Db.Where("name = ?", filter["name"])
	}
	Db.Order("created_at DESC").Find(&domains)

	for _, domain := range domains {
		domainLcuuids = append(domainLcuuids, domain.Lcuuid)
	}
	mysql.Db.Where("domain IN (?)", domainLcuuids).Find(&azs)

	domainToAZLcuuids = make(map[string][]string)
	domainToRegionLcuuidsToAZLcuuids = make(map[string]map[string][]string)
	for _, az := range azs {
		domainToAZLcuuids[az.Domain] = append(domainToAZLcuuids[az.Domain], az.Lcuuid)
		if _, ok := domainToRegionLcuuidsToAZLcuuids[az.Domain]; ok {
			regionToAZLcuuids := domainToRegionLcuuidsToAZLcuuids[az.Domain]
			regionToAZLcuuids[az.Region] = append(regionToAZLcuuids[az.Region], az.Lcuuid)
		} else {
			regionToAZLcuuids := map[string][]string{az.Region: {az.Lcuuid}}
			domainToRegionLcuuidsToAZLcuuids[az.Domain] = regionToAZLcuuids
		}
	}

	mysql.Db.Find(&controllers)
	controllerIPToName = make(map[string]string)
	for _, controller := range controllers {
		controllerIPToName[controller.IP] = controller.Name
	}

	mysql.Db.Find(&subDomains)
	domainToSubDomainNames := make(map[string][]string)
	for _, subDomain := range subDomains {
		domainToSubDomainNames[subDomain.Domain] = append(
			domainToSubDomainNames[subDomain.Domain], subDomain.Name,
		)
	}

	for _, domain := range domains {
		syncedAt := ""
		if domain.SyncedAt != nil {
			syncedAt = domain.SyncedAt.Format(common.GO_BIRTHDAY)
		}
		domainResp := model.Domain{
			ID:           domain.ClusterID,
			Name:         domain.Name,
			DisplayName:  domain.DisplayName,
			ClusterID:    domain.ClusterID,
			Type:         domain.Type,
			Enabled:      domain.Enabled,
			State:        domain.State,
			ErrorMsg:     domain.ErrorMsg,
			ControllerIP: domain.ControllerIP,
			IconID:       domain.IconID, // 后续与前端沟通icon作为默认配置
			CreatedAt:    domain.CreatedAt.Format(common.GO_BIRTHDAY),
			SyncedAt:     syncedAt,
			Lcuuid:       domain.Lcuuid,
		}

		if _, ok := domainToRegionLcuuidsToAZLcuuids[domain.Lcuuid]; ok {
			domainResp.RegionCount = len(domainToRegionLcuuidsToAZLcuuids[domain.Lcuuid])
		}
		if _, ok := domainToAZLcuuids[domain.Lcuuid]; ok {
			domainResp.AZCount = len(domainToAZLcuuids[domain.Lcuuid])
		}
		if _, ok := controllerIPToName[domain.ControllerIP]; ok {
			domainResp.ControllerName = controllerIPToName[domain.ControllerIP]
		}
		if domain.Type != common.KUBERNETES {
			domainResp.K8sEnabled = 1
			if subDomains, ok := domainToSubDomainNames[domain.Lcuuid]; ok {
				domainResp.PodClusters = subDomains
			}
		} else {
			var k8sCluster mysql.KubernetesCluster
			if err = mysql.Db.Where("cluster_id = ?", domain.ClusterID).First(&k8sCluster).Error; err == nil {
				v := strings.Split(k8sCluster.Value, "-")
				if len(v) == 2 {
					var vtap mysql.VTap
					if err = mysql.Db.Where("ctrl_ip = ? AND ctrl_mac = ?", v[0], v[1]).First(&vtap).Error; err == nil {
						domainResp.VTapName = vtap.Name
						domainResp.VTapCtrlIP = vtap.CtrlIP
						domainResp.VTapCtrlMAC = vtap.CtrlMac
					}
				}
			}
		}

		domainResp.Config = make(map[string]interface{})
		json.Unmarshal([]byte(domain.Config), &domainResp.Config)
		for _, key := range []string{
			"admin_password", "secret_key", "password", "boss_secret_key",
		} {
			if _, ok := domainResp.Config[key]; ok {
				domainResp.Config[key] = common.DEFAULT_ENCRYPTION_PASSWORD
			}
		}

		response = append(response, domainResp)
	}
	return response, nil
}

func CreateDomain(domainCreate model.DomainCreate) (*model.Domain, error) {
	var count int64

	mysql.Db.Model(&mysql.Domain{}).Where("name = ?", domainCreate.Name).Count(&count)
	if count > 0 {
		return nil, NewError(common.RESOURCE_ALREADY_EXIST, fmt.Sprintf("domain (%s) already exist", domainCreate.Name))
	}

	mysql.Db.Model(&mysql.SubDomain{}).Where("name = ?", domainCreate.Name).Count(&count)
	if count > 0 {
		return nil, NewError(common.RESOURCE_ALREADY_EXIST, fmt.Sprintf("sub_domain (%s) already exist", domainCreate.Name))
	}

	log.Infof("create domain (%v)", domainCreate)

	domain := mysql.Domain{}
	displayName := common.GetUUID(domainCreate.KubernetesClusterID, uuid.Nil)
	lcuuid := common.GetUUID(displayName, uuid.Nil)
	domain.Lcuuid = lcuuid
	domain.Name = domainCreate.Name
	domain.DisplayName = displayName
	domain.Type = domainCreate.Type
	domain.IconID = domainCreate.IconID

	// set region and controller ip if not specified
	if domainCreate.Config == nil {
		domainCreate.Config = map[string]interface{}{
			"region_uuid":   "",
			"controller_ip": "",
		}
	}
	var regionLcuuid string
	confRegion, ok := domainCreate.Config["region_uuid"]
	if !ok || confRegion.(string) == "" {
		var region mysql.Region
		res := mysql.Db.Find(&region)
		if res.RowsAffected != int64(1) {
			return nil, NewError(common.INVALID_PARAMETERS, fmt.Sprintf("can not find region, please specify or create one"))
		}
		domainCreate.Config["region_uuid"] = region.Lcuuid
		regionLcuuid = region.Lcuuid
	} else {
		regionLcuuid = confRegion.(string)
	}
	// TODO: controller_ip拿到config外面，直接作为domain的一级参数
	var controllerIP string
	confControllerIP, ok := domainCreate.Config["controller_ip"]
	if !ok || confControllerIP.(string) == "" {
		var azConn mysql.AZControllerConnection
		res := mysql.Db.Where("region = ?", regionLcuuid).First(&azConn)
		if res.RowsAffected != int64(1) {
			return nil, NewError(common.INVALID_PARAMETERS, fmt.Sprintf("can not find controller ip, please specify or create one"))
		}
		domainCreate.Config["controller_ip"] = azConn.ControllerIP
		controllerIP = azConn.ControllerIP
	} else {
		controllerIP = confControllerIP.(string)
	}
	domain.ControllerIP = controllerIP
	configStr, _ := json.Marshal(domainCreate.Config)
	domain.Config = string(configStr)

	if domainCreate.Type == common.KUBERNETES {
		// support specify cluster_id
		if domainCreate.KubernetesClusterID != "" {
			domain.ClusterID = domainCreate.KubernetesClusterID
		} else {
			domain.ClusterID = "d-" + common.GenerateShortUUID()
		}
		createKubernetesRelatedResources(domain, regionLcuuid)
	}
	mysql.Db.Clauses(clause.Insert{Modifier: "IGNORE"}).Create(&domain)

	response, _ := GetDomains(map[string]interface{}{"lcuuid": lcuuid})
	return &response[0], nil
}

func createKubernetesRelatedResources(domain mysql.Domain, regionLcuuid string) {
	if regionLcuuid == "" {
		regionLcuuid = common.DEFAULT_REGION
	}
	az := mysql.AZ{}
	az.Lcuuid = cloudcommon.GetAZLcuuidFromUUIDGenerate(domain.DisplayName)
	az.Name = domain.Name
	az.Domain = domain.Lcuuid
	az.Region = regionLcuuid
	az.CreateMethod = common.CREATE_METHOD_LEARN
	err := mysql.Db.Create(&az).Error
	if err != nil {
		log.Errorf("create az failed: %s", err)
	}
	vpc := mysql.VPC{}
	vpc.Lcuuid = k8s.GetVPCLcuuidFromUUIDGenerate(domain.DisplayName)
	vpc.Name = domain.Name
	vpc.Domain = domain.Lcuuid
	vpc.Region = regionLcuuid
	vpc.CreateMethod = common.CREATE_METHOD_LEARN
	err = mysql.Db.Create(&vpc).Error
	if err != nil {
		log.Errorf("create vpc failed: %s", err)
	}
	return
}

func UpdateDomain(lcuuid string, domainUpdate map[string]interface{}) (*model.Domain, error) {
	var domain mysql.Domain
	var dbUpdateMap = make(map[string]interface{})

	if ret := mysql.Db.Where("lcuuid = ?", lcuuid).First(&domain); ret.Error != nil {
		return nil, NewError(
			common.RESOURCE_NOT_FOUND, fmt.Sprintf("domain (%s) not found", lcuuid),
		)
	}

	log.Infof("update domain (%s) config (%v)", domain.Name, domainUpdate)

	// 修改名称
	if _, ok := domainUpdate["NAME"]; ok {
		dbUpdateMap["name"] = domainUpdate["NAME"]
	}

	// 禁用/启用
	if _, ok := domainUpdate["ENABLED"]; ok {
		dbUpdateMap["enabled"] = domainUpdate["ENABLED"]
	}

	// 图标
	if _, ok := domainUpdate["ICON_ID"]; ok {
		dbUpdateMap["icon_id"] = domainUpdate["ICON_ID"]
	}

	// 控制器IP
	if _, ok := domainUpdate["CONTROLLER_IP"]; ok {
		dbUpdateMap["controller_ip"] = domainUpdate["CONTROLLER_IP"]
	}

	// config
	// 注意：密码相关字段因为返回是****，所以不能直接把页面更新入库
	if _, ok := domainUpdate["CONFIG"]; ok && domainUpdate["CONFIG"] != nil {
		config := make(map[string]interface{})
		json.Unmarshal([]byte(domain.Config), &config)

		configUpdate := domainUpdate["CONFIG"].(map[string]interface{})
		for _, key := range []string{
			"admin_password", "secret_key", "password", "boss_secret_key",
		} {
			if _, ok := configUpdate[key]; ok {
				if configUpdate[key] == common.DEFAULT_ENCRYPTION_PASSWORD {
					configUpdate[key] = config[key]
				}
			}
		}
		// 如果存在资源同步控制器IP的修改，则需要更新controller_ip字段
		if controllerIP, ok := configUpdate["controller_ip"]; ok {
			if controllerIP != domain.ControllerIP {
				dbUpdateMap["controller_ip"] = controllerIP
			}
		}
		// 如果修改region，则清理掉云平台下所有软删除的数据
		if region, ok := configUpdate["region_uuid"]; ok {
			if region != config["region_uuid"] {
				log.Infof("delete domain (%s) soft deleted resource", domain.Name)
				deleteSoftDeletedResource(lcuuid)
			}
		}
		configStr, _ := json.Marshal(domainUpdate["CONFIG"])
		dbUpdateMap["config"] = string(configStr)
	}

	// 更新domain DB
	mysql.Db.Model(&domain).Updates(dbUpdateMap)

	response, _ := GetDomains(map[string]interface{}{"lcuuid": domain.Lcuuid})
	return &response[0], nil
}

func deleteSoftDeletedResource(lcuuid string) {
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.CEN{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.PeerConnection{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.RedisInstance{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.RDSInstance{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.LBTargetServer{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.LBListener{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.LB{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.NATGateway{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.SecurityGroup{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.DHCPPort{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.VRouter{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.Pod{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.PodReplicaSet{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.PodGroup{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.PodService{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.PodIngress{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.PodNamespace{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.PodNode{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.PodCluster{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.VM{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.Host{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.Network{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.VPC{})
	mysql.Db.Unscoped().Where("domain = ? & deleted_at != NULL", lcuuid).Delete(&mysql.AZ{})
}

func DeleteDomain(lcuuid string) (map[string]string, error) {
	var domain mysql.Domain

	if ret := mysql.Db.Where("lcuuid = ?", lcuuid).First(&domain); ret.Error != nil {
		return nil, NewError(
			common.RESOURCE_NOT_FOUND, fmt.Sprintf("domain (%s) not found", lcuuid),
		)
	}

	log.Infof("delete domain (%s)", domain.Name)

	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.WANIP{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.LANIP{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.FloatingIP{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.VInterface{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.CEN{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.PeerConnection{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.RedisInstance{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.RDSInstance{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.LBVMConnection{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.LBTargetServer{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.LBListener{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.LB{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.NATVMConnection{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.NATRule{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.NATGateway{})
	var sgs []mysql.SecurityGroup
	// mysql.Db.Unscoped().Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Where("domain = ?", lcuuid).Delete(&sgs)
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Find(&sgs)
	sgIDs := make([]int, len(sgs))
	for _, sg := range sgs {
		sgIDs = append(sgIDs, sg.ID)
	}
	mysql.Db.Unscoped().Where("sg_id IN ?", sgIDs).Delete(&mysql.VMSecurityGroup{})
	mysql.Db.Unscoped().Where("sg_id IN ?", sgIDs).Delete(&mysql.SecurityGroupRule{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.SecurityGroup{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.DHCPPort{})
	var vRouters []mysql.VRouter
	// mysql.Db.Unscoped().Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Where("domain = ?", lcuuid).Delete(&vRouters)
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Find(&vRouters)
	vRouterIDs := make([]int, len(vRouters))
	for _, vRouter := range vRouters {
		vRouterIDs = append(vRouterIDs, vRouter.ID)
	}
	mysql.Db.Unscoped().Where("vnet_id IN ?", vRouterIDs).Delete(&mysql.RoutingTable{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.VRouter{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.VMPodNodeConnection{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.Pod{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.PodReplicaSet{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.PodGroup{})
	var podServices []mysql.PodService
	// mysql.Db.Unscoped().Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Where("domain = ?", lcuuid).Delete(&podServices)
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Find(&podServices)
	podServiceIDs := make([]int, len(podServices))
	for _, podService := range podServices {
		podServiceIDs = append(podServiceIDs, podService.ID)
	}
	mysql.Db.Unscoped().Where("pod_service_id IN ?", podServiceIDs).Delete(&mysql.PodServicePort{})
	mysql.Db.Unscoped().Where("pod_service_id IN ?", podServiceIDs).Delete(&mysql.PodGroupPort{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.PodService{})
	var podIngresses []mysql.PodIngress
	// mysql.Db.Unscoped().Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Where("domain = ?", lcuuid).Delete(&podIngresses)
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Find(&podIngresses)
	podIngressIDs := make([]int, len(podIngresses))
	for _, podIngress := range podIngresses {
		podIngressIDs = append(podIngressIDs, podIngress.ID)
	}
	mysql.Db.Unscoped().Where("pod_ingress_id IN ?", podIngressIDs).Delete(&mysql.PodIngressRule{})
	mysql.Db.Unscoped().Where("pod_ingress_id IN ?", podIngressIDs).Delete(&mysql.PodIngressRuleBackend{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.PodIngress{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.PodNamespace{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.PodNode{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.PodCluster{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.VM{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.Host{})
	var networks []mysql.Network
	// mysql.Db.Unscoped().Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Where("domain = ?", lcuuid).Delete(&networks)
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Find(&networks)
	networkIDs := make([]int, len(networks))
	for _, network := range networks {
		networkIDs = append(networkIDs, network.ID)
	}
	mysql.Db.Unscoped().Where("vl2id IN ?", networkIDs).Delete(&mysql.Subnet{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.Network{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.VPC{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.SubDomain{})
	mysql.Db.Unscoped().Where("domain = ?", lcuuid).Delete(&mysql.AZ{})

	mysql.Db.Delete(&domain)
	return map[string]string{"LCUUID": lcuuid}, nil
}

func GetSubDomains(filter map[string]interface{}) ([]model.SubDomain, error) {
	var response []model.SubDomain
	var subDomains []mysql.SubDomain
	var vpcs []mysql.VPC

	Db := mysql.Db
	if _, ok := filter["lcuuid"]; ok {
		Db = Db.Where("lcuuid = ?", filter["lcuuid"])
	}
	if _, ok := filter["domain"]; ok {
		Db = Db.Where("domain = ?", filter["domain"])
	}
	Db.Order("created_at DESC").Find(&subDomains)

	mysql.Db.Select("name", "lcuuid").Find(&vpcs)
	lcuuidToVPCName := make(map[string]string)
	for _, vpc := range vpcs {
		lcuuidToVPCName[vpc.Lcuuid] = vpc.Name
	}

	for _, subDomain := range subDomains {
		syncedAt := ""
		if subDomain.SyncedAt != nil {
			syncedAt = subDomain.SyncedAt.Format(common.GO_BIRTHDAY)
		}
		subDomainResp := model.SubDomain{
			ID:           subDomain.ID,
			Name:         subDomain.Name,
			DisplayName:  subDomain.DisplayName,
			ClusterID:    subDomain.ClusterID,
			State:        subDomain.State,
			ErrorMsg:     subDomain.ErrorMsg,
			CreateMethod: subDomain.CreateMethod,
			CreatedAt:    subDomain.CreatedAt.Format(common.GO_BIRTHDAY),
			SyncedAt:     syncedAt,
			Domain:       subDomain.Domain,
			Lcuuid:       subDomain.Lcuuid,
		}

		subDomainResp.Config = make(map[string]interface{})
		json.Unmarshal([]byte(subDomain.Config), &subDomainResp.Config)

		if _, ok := subDomainResp.Config["vpc_uuid"]; ok {
			vpcLcuuid := subDomainResp.Config["vpc_uuid"].(string)
			if _, ok := lcuuidToVPCName[vpcLcuuid]; ok {
				subDomainResp.VPCName = lcuuidToVPCName[vpcLcuuid]
			}
		}
		response = append(response, subDomainResp)
	}
	return response, nil
}

func CreateSubDomain(subDomainCreate model.SubDomainCreate) (*model.SubDomain, error) {
	var count int64

	mysql.Db.Model(&mysql.SubDomain{}).Where("name = ?", subDomainCreate.Name).Count(&count)
	if count > 0 {
		return nil, NewError(common.RESOURCE_ALREADY_EXIST, fmt.Sprintf("sub_domain (%s) already exist", subDomainCreate.Name))
	}

	log.Infof("create sub_domain (%v)", subDomainCreate)

	subDomain := mysql.SubDomain{}
	displayName := common.GetUUID("", uuid.Nil)
	lcuuid := common.GetUUID(displayName, uuid.Nil)
	subDomain.Lcuuid = lcuuid
	subDomain.Name = subDomainCreate.Name
	subDomain.DisplayName = displayName
	subDomain.CreateMethod = common.CREATE_METHOD_USER_DEFINE
	subDomain.ClusterID = "d-" + common.GenerateShortUUID()
	subDomain.Domain = subDomainCreate.Domain
	configStr, _ := json.Marshal(subDomainCreate.Config)
	subDomain.Config = string(configStr)
	mysql.Db.Create(&subDomain)

	response, _ := GetSubDomains(map[string]interface{}{"lcuuid": lcuuid})
	return &response[0], nil
}

func UpdateSubDomain(lcuuid string, subDomainUpdate map[string]interface{}) (*model.SubDomain, error) {
	var subDomain mysql.SubDomain
	var dbUpdateMap = make(map[string]interface{})

	if ret := mysql.Db.Where("lcuuid = ?", lcuuid).First(&subDomain); ret.Error != nil {
		return nil, NewError(
			common.RESOURCE_NOT_FOUND, fmt.Sprintf("sub_domain (%s) not found", lcuuid),
		)
	}

	log.Infof("update sub_domain (%s) config (%v)", subDomain.Name, subDomainUpdate)

	// config
	if _, ok := subDomainUpdate["CONFIG"]; ok {
		configStr, _ := json.Marshal(subDomainUpdate["CONFIG"])
		dbUpdateMap["config"] = string(configStr)
	}

	// 更新domain DB
	mysql.Db.Model(&subDomain).Updates(dbUpdateMap)

	response, _ := GetSubDomains(map[string]interface{}{"lcuuid": lcuuid})
	return &response[0], nil
}

func DeleteSubDomain(lcuuid string) (map[string]string, error) {
	var subDomain mysql.SubDomain

	if ret := mysql.Db.Where("lcuuid = ?", lcuuid).First(&subDomain); ret.Error != nil {
		return nil, NewError(
			common.RESOURCE_NOT_FOUND, fmt.Sprintf("sub_domain (%s) not found", lcuuid),
		)
	}

	log.Infof("delete sub_domain (%s)", subDomain.Name)

	var podCluster mysql.PodCluster
	mysql.Db.Unscoped().Where("lcuuid = ?", lcuuid).Find(&podCluster)
	// TODO debug为什么此处赋值在mysql中没生效
	// mysql.Db.Unscoped().Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Where("lcuuid = ?", lcuuid).Delete(&podCluster)
	log.Info(podCluster)
	if podCluster.ID != 0 {
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.WANIP{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.LANIP{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.VInterface{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.Subnet{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.Network{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.VMPodNodeConnection{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.Pod{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.PodReplicaSet{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.PodGroupPort{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.PodGroup{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.PodServicePort{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.PodService{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.PodIngressRuleBackend{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.PodIngressRule{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.PodIngress{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.PodNamespace{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.PodNode{})
		mysql.Db.Unscoped().Where("sub_domain = ?", lcuuid).Delete(&mysql.PodCluster{})
	}

	mysql.Db.Delete(&subDomain)
	return map[string]string{"LCUUID": lcuuid}, nil
}
