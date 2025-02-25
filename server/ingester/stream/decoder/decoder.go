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

package decoder

import (
	"strconv"

	"github.com/golang/protobuf/proto"

	logging "github.com/op/go-logging"
	v1 "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/metaflowys/metaflow/server/ingester/stream/config"
	"github.com/metaflowys/metaflow/server/ingester/stream/jsonify"
	"github.com/metaflowys/metaflow/server/ingester/stream/throttler"
	"github.com/metaflowys/metaflow/server/libs/codec"
	"github.com/metaflowys/metaflow/server/libs/datatype"
	"github.com/metaflowys/metaflow/server/libs/datatype/pb"
	"github.com/metaflowys/metaflow/server/libs/grpc"
	"github.com/metaflowys/metaflow/server/libs/queue"
	"github.com/metaflowys/metaflow/server/libs/receiver"
	"github.com/metaflowys/metaflow/server/libs/stats"
	"github.com/metaflowys/metaflow/server/libs/utils"
)

var log = logging.MustGetLogger("stream.decoder")

const (
	BUFFER_SIZE  = 1024
	L7_PROTO_MAX = datatype.L7_PROTOCOL_DNS + 1
)

type Counter struct {
	RawCount         int64 `statsd:"raw-count"`
	L4Count          int64 `statsd:"l4-count"`
	L4DropCount      int64 `statsd:"l4-drop-count"`
	L7Count          int64 `statsd:"l7-count"`
	L7DropCount      int64 `statsd:"l7-drop-count"`
	L7HTTPCount      int64 `statsd:"l7-http-count"`
	L7HTTPDropCount  int64 `statsd:"l7-http-drop-count"`
	L7DNSCount       int64 `statsd:"l7-dns-count"`
	L7DNSDropCount   int64 `statsd:"l7-dns-drop-count"`
	L7SQLCount       int64 `statsd:"l7-sql-count"`
	L7SQLDropCount   int64 `statsd:"l7-sql-drop-count"`
	L7NoSQLCount     int64 `statsd:"l7-nosql-count"`
	L7NoSQLDropCount int64 `statsd:"l7-nosql-drop-count"`
	L7RPCCount       int64 `statsd:"l7-rpc-count"`
	L7RPCDropCount   int64 `statsd:"l7-rpc-drop-count"`
	L7MQCount        int64 `statsd:"l7-mq-count"`
	L7MQDropCount    int64 `statsd:"l7-mq-drop-count"`
	OTelCount        int64 `statsd:"otel-count"`
	OTelDropCount    int64 `statsd:"otel-drop-count"`
	ErrorCount       int64 `statsd:"err-count"`
}

type Decoder struct {
	index        int
	msgType      datatype.MessageType
	shardID      int
	platformData *grpc.PlatformInfoTable
	inQueue      queue.QueueReader
	throttler    *throttler.ThrottlingQueue
	debugEnabled bool

	l7Disableds [L7_PROTO_MAX]bool
	l7Disabled  bool
	l4Disabled  bool

	counter *Counter
	utils.Closable
}

func NewDecoder(
	index, shardID int, msgType datatype.MessageType,
	platformData *grpc.PlatformInfoTable,
	inQueue queue.QueueReader,
	throttler *throttler.ThrottlingQueue,
	flowLogDisabled *config.FlowLogDisabled,
) *Decoder {
	return &Decoder{
		index:        index,
		msgType:      msgType,
		shardID:      shardID,
		platformData: platformData,
		inQueue:      inQueue,
		throttler:    throttler,
		debugEnabled: log.IsEnabledFor(logging.DEBUG),
		counter:      &Counter{},
		l7Disableds:  getL7Disables(flowLogDisabled),
		l7Disabled:   flowLogDisabled.L7,
		l4Disabled:   flowLogDisabled.L4,
	}
}

func getL7Disables(flowLogConfig *config.FlowLogDisabled) [L7_PROTO_MAX]bool {
	l7Disableds := [L7_PROTO_MAX]bool{}
	l7Disableds[datatype.L7_PROTOCOL_HTTP_1] = flowLogConfig.Http
	l7Disableds[datatype.L7_PROTOCOL_HTTP_2] = flowLogConfig.Http
	l7Disableds[datatype.L7_PROTOCOL_DNS] = flowLogConfig.Dns
	l7Disableds[datatype.L7_PROTOCOL_MYSQL] = flowLogConfig.Mysql
	l7Disableds[datatype.L7_PROTOCOL_REDIS] = flowLogConfig.Redis
	l7Disableds[datatype.L7_PROTOCOL_DUBBO] = flowLogConfig.Dubbo
	l7Disableds[datatype.L7_PROTOCOL_KAFKA] = flowLogConfig.Kafka
	l7Disableds[datatype.L7_PROTOCOL_MQTT] = flowLogConfig.Mqtt
	return l7Disableds
}

func (d *Decoder) GetCounter() interface{} {
	var counter *Counter
	counter, d.counter = d.counter, &Counter{}
	return counter
}

func (d *Decoder) Run() {
	stats.RegisterCountable("decoder", d, stats.OptionStatTags{
		"thread":   strconv.Itoa(d.index),
		"msg_type": d.msgType.String()})
	buffer := make([]interface{}, BUFFER_SIZE)
	decoder := &codec.SimpleDecoder{}
	pbTaggedFlow := pb.NewTaggedFlow()
	pbTracesData := &v1.TracesData{}
	for {
		n := d.inQueue.Gets(buffer)
		for i := 0; i < n; i++ {
			if buffer[i] == nil {
				d.flush()
				continue
			}
			d.counter.RawCount++
			recvBytes, ok := buffer[i].(*receiver.RecvBuffer)
			if !ok {
				log.Warning("get decode queue data type wrong")
				continue
			}
			decoder.Init(recvBytes.Buffer[recvBytes.Begin:recvBytes.End])
			if d.msgType == datatype.MESSAGE_TYPE_PROTOCOLLOG && !d.l7Disabled {
				d.handleProtoLog(decoder)
			} else if d.msgType == datatype.MESSAGE_TYPE_TAGGEDFLOW && !d.l4Disabled {
				d.handleTaggedFlow(decoder, pbTaggedFlow)
			} else if d.msgType == datatype.MESSAGE_TYPE_OPENTELEMETRY {
				d.handleOpenTelemetry(recvBytes.VtapID, decoder, pbTracesData)
			}
			receiver.ReleaseRecvBuffer(recvBytes)
		}
	}
}

func (d *Decoder) handleTaggedFlow(decoder *codec.SimpleDecoder, pbTaggedFlow *pb.TaggedFlow) {
	for !decoder.IsEnd() {
		pbTaggedFlow.ResetAll()
		decoder.ReadPB(pbTaggedFlow)
		if decoder.Failed() {
			d.counter.ErrorCount++
			log.Errorf("flow decode failed, offset=%d len=%d", decoder.Offset(), len(decoder.Bytes()))
			return
		}
		if !pbTaggedFlow.IsValid() {
			d.counter.ErrorCount++
			log.Warningf("invalid flow %s", pbTaggedFlow.Flow)
			continue
		}
		d.sendFlow(pbTaggedFlow)
	}
}

func (d *Decoder) handleProtoLog(decoder *codec.SimpleDecoder) {
	for !decoder.IsEnd() {
		protoLog := pb.AcquirePbAppProtoLogsData()

		decoder.ReadPB(protoLog)
		if decoder.Failed() || !protoLog.IsValid() {
			d.counter.ErrorCount++
			pb.ReleasePbAppProtoLogsData(protoLog)
			log.Errorf("proto log decode failed, offset=%d len=%d", decoder.Offset(), len(decoder.Bytes()))
			return
		}
		d.sendProto(protoLog)
	}
}

func (d *Decoder) handleOpenTelemetry(vtapID uint16, decoder *codec.SimpleDecoder, pbTracesData *v1.TracesData) {
	var err error
	for !decoder.IsEnd() {
		pbTracesData.Reset()
		bytes := decoder.ReadBytes()
		if len(bytes) > 0 {
			err = proto.Unmarshal(bytes, pbTracesData)
		}
		if decoder.Failed() || err != nil {
			if d.counter.ErrorCount == 0 {
				log.Errorf("OpenTelemetry log decode failed, offset=%d len=%d err: %s", decoder.Offset(), len(decoder.Bytes()), err)
			}
			d.counter.ErrorCount++
			return
		}
		d.sendOpenMetetry(vtapID, pbTracesData)
	}
}

func (d *Decoder) sendOpenMetetry(vtapID uint16, tracesData *v1.TracesData) {
	if d.debugEnabled {
		log.Debugf("decoder %d vtap %d recv otel: %s", d.index, vtapID, tracesData)
	}
	d.counter.OTelCount++
	ls := jsonify.OTelTracesDataToL7Loggers(vtapID, tracesData, d.shardID, d.platformData)
	for _, l := range ls {
		if !d.throttler.Send(l) {
			d.counter.OTelDropCount++
		}
	}
}

func (d *Decoder) sendFlow(flow *pb.TaggedFlow) {
	if d.debugEnabled {
		log.Debugf("decoder %d recv flow: %s", d.index, flow)
	}
	d.counter.L4Count++
	l := jsonify.TaggedFlowToLogger(flow, d.shardID, d.platformData)
	if !d.throttler.Send(l) {
		d.counter.L4DropCount++
	}
}

func (d *Decoder) sendProto(proto *pb.AppProtoLogsData) {
	if d.debugEnabled {
		log.Debugf("decoder %d recv proto: %s", d.index, proto)
	}

	d.counter.L7Count++
	drop := int64(0)
	if proto.Base.Head.Proto < uint32(L7_PROTO_MAX) &&
		d.l7Disableds[proto.Base.Head.Proto] {
		drop = 1
	} else {
		l := jsonify.ProtoLogToL7Logger(proto, d.shardID, d.platformData)
		if !d.throttler.Send(l) {
			d.counter.L7DropCount++
			drop = 1
		}
	}
	proto.Release()

	switch datatype.L7Protocol(proto.Base.Head.Proto) {
	case datatype.L7_PROTOCOL_HTTP_1, datatype.L7_PROTOCOL_HTTP_2:
		d.counter.L7HTTPCount++
		d.counter.L7HTTPDropCount += drop
	case datatype.L7_PROTOCOL_DNS:
		d.counter.L7DNSCount++
		d.counter.L7DNSDropCount += drop
	case datatype.L7_PROTOCOL_MYSQL:
		d.counter.L7SQLCount++
		d.counter.L7SQLDropCount += drop
	case datatype.L7_PROTOCOL_REDIS:
		d.counter.L7NoSQLCount++
		d.counter.L7NoSQLDropCount += drop
	case datatype.L7_PROTOCOL_DUBBO:
		d.counter.L7RPCCount++
		d.counter.L7RPCDropCount += drop
	case datatype.L7_PROTOCOL_KAFKA:
		fallthrough
	case datatype.L7_PROTOCOL_MQTT:
		d.counter.L7MQCount++
		d.counter.L7MQDropCount += drop
	}
}

func (d *Decoder) flush() {
	if d.throttler != nil {
		d.throttler.Send(nil)
	}
}
