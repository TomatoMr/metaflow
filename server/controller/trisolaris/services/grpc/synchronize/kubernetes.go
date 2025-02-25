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

package synchronize

import (
	api "github.com/metaflowys/metaflow/message/trident"
	context "golang.org/x/net/context"

	"github.com/metaflowys/metaflow/server/controller/common"
	"github.com/metaflowys/metaflow/server/controller/trisolaris"
)

type KubernetesClusterIDEvent struct {
}

func NewKubernetesClusterIDEvent() *KubernetesClusterIDEvent {
	return &KubernetesClusterIDEvent{}
}

func (k *KubernetesClusterIDEvent) GetKubernetesClusterID(ctx context.Context, in *api.KubernetesClusterIDRequest) (*api.KubernetesClusterIDResponse, error) {
	clusterID, err := common.GenerateKuberneteClusterIDByMD5(in.GetCaMd5())
	if err != nil {
		errorMsg := err.Error()
		return &api.KubernetesClusterIDResponse{ErrorMsg: &errorMsg}, nil
	}

	// cache clusterID & create kubernetes domain
	kubernetesInfo := trisolaris.GetGKubernetesInfo()
	kubernetesInfo.CacheClusterID(clusterID)

	return &api.KubernetesClusterIDResponse{ClusterId: &clusterID}, nil
}
