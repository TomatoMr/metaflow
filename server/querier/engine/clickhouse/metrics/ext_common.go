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

package metrics

import (
	"fmt"

	"github.com/metaflowys/metaflow/server/querier/config"
	"github.com/metaflowys/metaflow/server/querier/engine/clickhouse/client"
)

var EXT_METRICS = map[string]*Metrics{}

func GetExtMetrics(db, table, where string) (map[string]*Metrics, error) {
	loadMetrics := make(map[string]*Metrics)
	var err error
	if db == "ext_metrics" {
		externalChClient := client.Client{
			Host:     config.Cfg.Clickhouse.Host,
			Port:     config.Cfg.Clickhouse.Port,
			UserName: config.Cfg.Clickhouse.User,
			Password: config.Cfg.Clickhouse.Password,
			DB:       db,
		}
		externalMetricIntSql := fmt.Sprintf("SELECT arrayJoin(metrics_int_names) AS metrics_int_name FROM (SELECT metrics_int_names FROM %s) GROUP BY metrics_int_name", table)
		externalMetricFloatSql := fmt.Sprintf("SELECT arrayJoin(metrics_float_names) AS metrics_float_name FROM (SELECT metrics_float_names FROM %s) GROUP BY metrics_float_name", table)
		externalMetricIntRst, err := externalChClient.DoQuery(externalMetricIntSql, nil, "")
		if err != nil {
			log.Error(err)
			return nil, err
		}
		externalMetricFloatRst, err := externalChClient.DoQuery(externalMetricFloatSql, nil, "")
		if err != nil {
			log.Error(err)
			return nil, err
		}
		for i, _tagName := range externalMetricIntRst["values"] {
			tagName := _tagName.([]interface{})[0]
			externalTag := tagName.(string)
			dbField := fmt.Sprintf("metrics_int_values[indexOf(metrics_int_names, '%s')]", externalTag)
			lm := NewMetrics(
				i, dbField, externalTag, "", METRICS_TYPE_COUNTER,
				"原始Tag", []bool{true, true, true}, externalTag,
				table,
			)
			metricName := fmt.Sprintf("%s.%s", "int", externalTag)
			loadMetrics[metricName] = lm
		}
		for i, _tagName := range externalMetricFloatRst["values"] {
			tagName := _tagName.([]interface{})[0]
			externalTag := tagName.(string)
			dbField := fmt.Sprintf("metrics_float_values[indexOf(metrics_float_names, '%s')]", externalTag)
			lm := NewMetrics(
				i+len(externalMetricIntRst["values"]), dbField, externalTag, "", METRICS_TYPE_COUNTER,
				"原始Tag", []bool{true, true, true}, externalTag, table,
			)
			metricName := fmt.Sprintf("%s.%s", "float", externalTag)
			loadMetrics[metricName] = lm
		}
	}
	return loadMetrics, err
}
