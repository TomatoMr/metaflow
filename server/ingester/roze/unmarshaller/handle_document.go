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

package unmarshaller

import (
	"fmt"
	"net"

	"github.com/metaflowys/metaflow/message/trident"
	"github.com/metaflowys/metaflow/server/ingester/common"
	"github.com/metaflowys/metaflow/server/libs/app"
	"github.com/metaflowys/metaflow/server/libs/datatype"
	"github.com/metaflowys/metaflow/server/libs/grpc"
	"github.com/metaflowys/metaflow/server/libs/utils"
	"github.com/metaflowys/metaflow/server/libs/zerodoc"
)

const (
	EdgeCode    = zerodoc.IPPath | zerodoc.L3EpcIDPath
	MainAddCode = zerodoc.RegionID | zerodoc.HostID | zerodoc.L3Device | zerodoc.SubnetID | zerodoc.PodNodeID | zerodoc.AZID | zerodoc.PodGroupID | zerodoc.PodNSID | zerodoc.PodID | zerodoc.PodClusterID | zerodoc.ServiceID | zerodoc.Resource
	EdgeAddCode = zerodoc.RegionIDPath | zerodoc.HostIDPath | zerodoc.L3DevicePath | zerodoc.SubnetIDPath | zerodoc.PodNodeIDPath | zerodoc.AZIDPath | zerodoc.PodGroupIDPath | zerodoc.PodNSIDPath | zerodoc.PodIDPath | zerodoc.PodClusterIDPath | zerodoc.ServiceIDPath | zerodoc.ResourcePath
	PortAddCode = zerodoc.IsKeyService
)

func DocumentExpand(doc *app.Document, platformData *grpc.PlatformInfoTable) error {
	t := doc.Tagger.(*zerodoc.Tag)
	t.SetID("") // 由于需要修改Tag增删Field，清空ID避免字段脏

	// vtap_acl 分钟级数据不用填充
	if doc.Meter.ID() == zerodoc.ACL_ID &&
		t.DatabaseSuffixID() == 1 { // 只有acl后缀
		return nil
	}

	var info, info1 *grpc.Info
	myRegionID := uint16(platformData.QueryRegionID())
	if t.Code&zerodoc.ServerPort == zerodoc.ServerPort {
		t.Code |= PortAddCode
	}
	if t.Code&EdgeCode == EdgeCode {
		t.Code |= EdgeAddCode

		if t.L3EpcID == datatype.EPC_FROM_INTERNET && t.L3EpcID1 == datatype.EPC_FROM_INTERNET {
			return nil
		}
		// 当MAC/MAC1非0时，通过MAC来获取资源信息
		if t.MAC != 0 && t.MAC1 != 0 {
			info, info1 = platformData.QueryMacInfosPair(t.MAC|uint64(t.L3EpcID)<<48, t.MAC1|uint64(t.L3EpcID1)<<48)
			if info == nil {
				info = common.RegetInfoFromIP(t.IsIPv6 == 1, t.IP6, t.IP, t.L3EpcID, platformData)
			}
			if info1 == nil {
				info1 = common.RegetInfoFromIP(t.IsIPv6 == 1, t.IP61, t.IP1, t.L3EpcID1, platformData)
			}
		} else if t.MAC != 0 {
			info = platformData.QueryMacInfo(t.MAC | uint64(t.L3EpcID)<<48)
			if info == nil {
				info = common.RegetInfoFromIP(t.IsIPv6 == 1, t.IP6, t.IP, t.L3EpcID, platformData)
			}
			if t.IsIPv6 != 0 {
				info1 = platformData.QueryIPV6Infos(t.L3EpcID1, t.IP61)
			} else {
				info1 = platformData.QueryIPV4Infos(t.L3EpcID1, t.IP1)
			}
		} else if t.MAC1 != 0 {
			if t.IsIPv6 != 0 {
				info = platformData.QueryIPV6Infos(t.L3EpcID, t.IP6)
			} else {
				info = platformData.QueryIPV4Infos(t.L3EpcID, t.IP)
			}
			info1 = platformData.QueryMacInfo(t.MAC1 | uint64(t.L3EpcID1)<<48)
			if info1 == nil {
				info1 = common.RegetInfoFromIP(t.IsIPv6 == 1, t.IP61, t.IP1, t.L3EpcID1, platformData)
			}
		} else if t.IsIPv6 != 0 {
			info, info1 = platformData.QueryIPV6InfosPair(t.L3EpcID, t.IP6, t.L3EpcID1, t.IP61)
		} else {
			info, info1 = platformData.QueryIPV4InfosPair(t.L3EpcID, t.IP, t.L3EpcID1, t.IP1)
		}
		if info1 != nil {
			t.RegionID1 = uint16(info1.RegionID)
			t.HostID1 = uint16(info1.HostID)
			t.L3DeviceID1 = info1.DeviceID
			t.L3DeviceType1 = zerodoc.DeviceType(info1.DeviceType)
			t.SubnetID1 = uint16(info1.SubnetID)
			t.PodNodeID1 = info1.PodNodeID
			t.PodNSID1 = uint16(info1.PodNSID)
			t.AZID1 = uint16(info1.AZID)
			t.PodGroupID1 = info1.PodGroupID
			t.PodID1 = info1.PodID
			t.PodClusterID1 = uint16(info1.PodClusterID)
			serviceID := uint32(0)
			isKeyService := false
			if t.IsIPv6 != 0 {
				// 如果存在port，需要设置is_key_service, 并获取service_id
				if t.Code&PortAddCode != 0 {
					isKeyService, serviceID = platformData.QueryIPv6IsKeyServiceAndID(t.L3EpcID1, t.IP61, t.Protocol, t.ServerPort)
					if isKeyService {
						t.IsKeyService = 1
					}
				} else {
					// 没有port,如果是cluserIP 或 pod_IP，需要获取service_id
					if t.L3DeviceType1 == zerodoc.DeviceType(trident.DeviceType_DEVICE_TYPE_POD_SERVICE) ||
						t.PodID1 != 0 {
						_, serviceID = platformData.QueryIPv6IsKeyServiceAndID(t.L3EpcID1, t.IP61, t.Protocol, 0)
					}
				}
				// 如果是pod服务，需要设置service_id
				if t.L3DeviceType1 == zerodoc.DeviceType(trident.DeviceType_DEVICE_TYPE_POD_SERVICE) ||
					t.PodID1 != 0 ||
					t.PodNodeID1 != 0 {
					t.ServiceID1 = serviceID
				}
			} else {
				// 如果存在port，需要设置is_key_service, 并获取service_id
				if t.Code&PortAddCode != 0 {
					isKeyService, serviceID = platformData.QueryIsKeyServiceAndID(t.L3EpcID1, t.IP1, t.Protocol, t.ServerPort)
					if isKeyService {
						t.IsKeyService = 1
					}
				} else {
					// 没有port,如果是cluserIP 或 pod_IP，需要获取service_id
					if t.L3DeviceType1 == zerodoc.DeviceType(trident.DeviceType_DEVICE_TYPE_POD_SERVICE) ||
						t.PodID1 != 0 {
						_, serviceID = platformData.QueryIsKeyServiceAndID(t.L3EpcID1, t.IP1, t.Protocol, 0)
					}
				}
				// 如果是pod服务，需要设置service_id
				if t.L3DeviceType1 == zerodoc.DeviceType(trident.DeviceType_DEVICE_TYPE_POD_SERVICE) ||
					t.PodID1 != 0 ||
					t.PodNodeID1 != 0 {
					t.ServiceID1 = serviceID
				}
			}
			if info == nil {
				var ip0 net.IP
				if t.IsIPv6 != 0 {
					ip0 = t.IP6
				} else {
					ip0 = utils.IpFromUint32(t.IP)
				}
				// 当0侧是组播ip时，使用1侧的region_id,subnet_id,az_id来填充
				if ip0.IsMulticast() {
					t.RegionID = t.RegionID1
					t.SubnetID = t.SubnetID1
					t.AZID = t.AZID1
				}
			}
			if myRegionID != 0 && t.RegionID1 != 0 {
				if t.TAPSide == zerodoc.Server && t.RegionID1 != myRegionID { // 对于双端 的统计值，需要去掉 tap_side 对应的一侧与自身region_id 不匹配的内容。
					platformData.AddOtherRegion()
					return fmt.Errorf("My regionID is %d, but document regionID1 is %d", myRegionID, t.RegionID1)
				}
			}
		}
		t.ResourceGl0ID1, t.ResourceGl0Type1 = common.GetResourceGl0(t.PodID1, t.PodNodeID1, t.L3DeviceID1, uint8(t.L3DeviceType1), t.L3EpcID1)
		t.ResourceGl1ID1, t.ResourceGl1Type1 = common.GetResourceGl1(t.PodGroupID1, t.PodNodeID1, t.L3DeviceID1, uint8(t.L3DeviceType1), t.L3EpcID1)
		t.ResourceGl2ID1, t.ResourceGl2Type1 = common.GetResourceGl2(t.ServiceID1, t.PodGroupID1, t.PodNodeID1, t.L3DeviceID1, uint8(t.L3DeviceType1), t.L3EpcID1)
	} else {
		t.Code |= MainAddCode
		if t.L3EpcID == datatype.EPC_FROM_INTERNET {
			return nil
		}

		if t.MAC != 0 {
			info = platformData.QueryMacInfo(t.MAC | uint64(t.L3EpcID)<<48)
			if info == nil {
				info = common.RegetInfoFromIP(t.IsIPv6 == 1, t.IP6, t.IP, t.L3EpcID, platformData)
			}
		} else if t.IsIPv6 != 0 {
			info = platformData.QueryIPV6Infos(t.L3EpcID, t.IP6)
		} else {
			info = platformData.QueryIPV4Infos(t.L3EpcID, t.IP)
		}
	}

	if info != nil {
		t.RegionID = uint16(info.RegionID)
		t.HostID = uint16(info.HostID)
		t.L3DeviceID = info.DeviceID
		t.L3DeviceType = zerodoc.DeviceType(info.DeviceType)
		t.SubnetID = uint16(info.SubnetID)
		t.PodNodeID = info.PodNodeID
		t.PodNSID = uint16(info.PodNSID)
		t.AZID = uint16(info.AZID)
		t.PodGroupID = info.PodGroupID
		t.PodID = info.PodID
		t.PodClusterID = uint16(info.PodClusterID)
		isKeyService := false
		serviceID := uint32(0)
		if t.IsIPv6 != 0 {
			// 在0端, 有port无edge的数据需计算isKeyService，如:vtap_flow_port
			if t.Code&PortAddCode != 0 && t.Code&EdgeCode == 0 {
				isKeyService, serviceID = platformData.QueryIPv6IsKeyServiceAndID(t.L3EpcID, t.IP6, t.Protocol, t.ServerPort)
				if isKeyService {
					t.IsKeyService = 1
				}
			} else {
				// 有port有edge, 无port无edge，无port有edge
				// 如果是cluserIP 或 pod_IP，需要获取service_id
				if t.L3DeviceType == zerodoc.DeviceType(trident.DeviceType_DEVICE_TYPE_POD_SERVICE) ||
					t.PodID != 0 {
					_, serviceID = platformData.QueryIPv6IsKeyServiceAndID(t.L3EpcID, t.IP6, t.Protocol, 0)
				}
			}
			// 如果是pod服务，需要设置service_id
			if t.L3DeviceType == zerodoc.DeviceType(trident.DeviceType_DEVICE_TYPE_POD_SERVICE) ||
				t.PodID != 0 ||
				t.PodNodeID != 0 {
				t.ServiceID = serviceID
			}
		} else {
			// 在0端, 有port无edge的数据需计算isKeyService，如:vtap_flow_port
			if t.Code&PortAddCode != 0 && t.Code&EdgeCode == 0 {
				isKeyService, serviceID = platformData.QueryIsKeyServiceAndID(t.L3EpcID, t.IP, t.Protocol, t.ServerPort)
				if isKeyService {
					t.IsKeyService = 1
				}
			} else {
				// 有port有edge, 无port无edge，无port有edge
				// 如果是cluserIP 或 pod_IP，需要获取service_id
				if t.L3DeviceType == zerodoc.DeviceType(trident.DeviceType_DEVICE_TYPE_POD_SERVICE) ||
					t.PodID != 0 {
					_, serviceID = platformData.QueryIsKeyServiceAndID(t.L3EpcID, t.IP, t.Protocol, 0)
				}
			}
			// 如果是pod服务，需要设置service_id
			if t.L3DeviceType == zerodoc.DeviceType(trident.DeviceType_DEVICE_TYPE_POD_SERVICE) ||
				t.PodID != 0 ||
				t.PodNodeID != 0 {
				t.ServiceID = serviceID
			}
		}
		if info1 == nil && (t.Code&EdgeCode == EdgeCode) {
			var ip1 net.IP
			if t.IsIPv6 != 0 {
				ip1 = t.IP61
			} else {
				ip1 = utils.IpFromUint32(t.IP1)
			}
			// 当1侧是组播ip时，使用0侧的region_id,subnet_id,az_id来填充
			if ip1.IsMulticast() {
				t.RegionID1 = t.RegionID
				t.SubnetID1 = t.SubnetID
				t.AZID1 = t.AZID
			}
		}

		if myRegionID != 0 && t.RegionID != 0 {
			if t.Code&EdgeCode == EdgeCode { // 对于双端 的统计值，需要去掉 tap_side 对应的一侧与自身region_id 不匹配的内容。
				if t.TAPSide == zerodoc.Client && t.RegionID != myRegionID {
					platformData.AddOtherRegion()
					return fmt.Errorf("My regionID is %d, but document regionID is %d", myRegionID, t.RegionID)
				}
			} else { // 对于单端的统计值，需要去掉与自身region_id不匹配的内容
				if t.RegionID != myRegionID {
					platformData.AddOtherRegion()
					return fmt.Errorf("My regionID is %d, but document regionID is %d", myRegionID, t.RegionID)
				}
			}
		}
		t.ResourceGl0ID, t.ResourceGl0Type = common.GetResourceGl0(t.PodID, t.PodNodeID, t.L3DeviceID, uint8(t.L3DeviceType), t.L3EpcID)
		t.ResourceGl1ID, t.ResourceGl1Type = common.GetResourceGl1(t.PodGroupID, t.PodNodeID, t.L3DeviceID, uint8(t.L3DeviceType), t.L3EpcID)
		t.ResourceGl2ID, t.ResourceGl2Type = common.GetResourceGl2(t.ServiceID, t.PodGroupID, t.PodNodeID, t.L3DeviceID, uint8(t.L3DeviceType), t.L3EpcID)
	}

	return nil
}
