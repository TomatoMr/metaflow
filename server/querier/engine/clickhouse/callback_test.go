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

package clickhouse

import (
	//"github.com/k0kubun/pp"
	//"github.com/metaflowys/metaflow/server/querier/common"
	//"github.com/metaflowys/metaflow/server/querier/parse"
	//"github.com/metaflowys/metaflow/server/querier/querier"
	"reflect"
	"testing"

	"github.com/metaflowys/metaflow/server/querier/engine/clickhouse/view"
)

func TestTimeFill(t *testing.T) {
	m := view.NewModel()
	m.Time.TimeStart = 1645089282
	m.Time.TimeEnd = 1645175682
	m.Time.Fill = "0"
	m.Time.Interval = 21600
	m.Time.Alias = "time"
	callback := TimeFill([]interface{}{m})
	columns := []interface{}{
		"time",
		"field_0",
		"field_1",
	}
	values := []interface{}{
		[]interface{}{
			1645092000,
			1,
			2,
		},
	}
	want := []interface{}{
		[]interface{}{
			1645070400,
			0,
			0,
		},
		[]interface{}{
			1645092000,
			1,
			2,
		},
		[]interface{}{
			1645113600,
			0,
			0,
		},
		[]interface{}{
			1645135200,
			0,
			0,
		},
		[]interface{}{
			1645156800,
			0,
			0,
		},
	}
	newValues := callback(columns, values)
	if !reflect.DeepEqual(newValues, want) {
		t.Errorf("Callback: TimeFill, columns: %v, values: %v, newValues: %v, want: %v", columns, values, newValues, want)
	}
}
