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

package config

type GenesisConfig struct {
	AgingTime               float64  `default:"86400" yaml:"aging_time"`
	VinterfaceAgingTime     float64  `default:"300" yaml:"vinterface_aging_time"`
	LocalIPRanges           []string `yaml:"local_ip_ranges"`
	ExcludeIPRanges         []string `yaml:"exclude_ip_ranges"`
	QueueLengths            int      `default:"60" yaml:"queue_length"`
	DataPersistenceInterval int      `default:"60" yaml:"data_persistence_interval"`
	IPv4CIDRMaxMask         int      `default:"16" yaml:"ipv4_cidr_max_mask"`
	IPv6CIDRMaxMask         int      `default:"64" yaml:"ipv6_cidr_max_mask"`
	GRPCServerPort          string   `default:"30035" yaml:"grpc_server_port"`
}
