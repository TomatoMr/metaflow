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

use std::sync::Arc;
use std::time::Duration;
use std::time::Instant;

use criterion::*;

use metaflow_agent::{
    _FlowPerfCounter as FlowPerfCounter, _TcpFlags as TcpFlags, _TcpPerf as TcpPerf,
    _TridentType as TridentType, _benchmark_report as benchmark_report,
    _benchmark_session_peer_seq_no_assert as benchmark_session_peer_seq_no_assert,
    _meta_flow_perf_update as meta_flow_perf_update,
    _new_flow_map_and_receiver as new_flow_map_and_receiver, _new_meta_packet as new_meta_packet,
    _reverse_meta_packet as reverse_meta_packet,
};

fn bench_flow_map(c: &mut Criterion) {
    c.bench_function("flow_map_syn_flood", |b| {
        b.iter_custom(|iters| {
            let (mut map, _) = new_flow_map_and_receiver(TridentType::TtProcess);
            let packets = (0..iters)
                .into_iter()
                .map(|i| {
                    let mut pkt = new_meta_packet();
                    pkt.lookup_key.src_port = i as u16;
                    pkt.lookup_key.dst_port = (i >> 16) as u16;
                    pkt
                })
                .collect::<Vec<_>>();
            let start = Instant::now();
            for pkt in packets {
                map.inject_meta_packet(pkt);
            }
            start.elapsed()
        })
    });

    c.bench_function("flow_map_with_ten_packets_flow_flood", |b| {
        b.iter_custom(|iters| {
            let (mut map, _) = new_flow_map_and_receiver(TridentType::TtProcess);
            let iters = (iters + 9) / 10 * 10;

            let mut packets = vec![];
            for i in (0..iters).step_by(10) {
                let src_port = i as u16;
                let dst_port = (i >> 16) as u16;

                let mut pkt = new_meta_packet();
                pkt.lookup_key.timestamp += Duration::from_nanos(100 * i);
                pkt.lookup_key.src_port = src_port;
                pkt.lookup_key.dst_port = dst_port;
                packets.push(pkt);

                let mut pkt = new_meta_packet();
                pkt.lookup_key.timestamp += Duration::from_nanos(100 * (i + 1));
                reverse_meta_packet(&mut pkt);
                pkt.lookup_key.src_port = dst_port;
                pkt.lookup_key.dst_port = src_port;
                pkt.tcp_data.flags = TcpFlags::SYN_ACK;
                packets.push(pkt);

                for k in 2..10 {
                    let mut pkt = new_meta_packet();
                    pkt.lookup_key.timestamp += Duration::from_nanos(100 * (i + k));
                    pkt.lookup_key.src_port = src_port;
                    pkt.lookup_key.dst_port = dst_port;
                    pkt.tcp_data.flags = TcpFlags::ACK;
                    packets.push(pkt);
                }
            }

            let start = Instant::now();
            for pkt in packets {
                map.inject_meta_packet(pkt);
            }
            start.elapsed()
        })
    });
}

fn bench_perf(c: &mut Criterion) {
    c.bench_function("perf_stats_report", |b| {
        b.iter_custom(|iters| {
            let mut perf = TcpPerf::new(Arc::new(FlowPerfCounter::default()));
            let start = Instant::now();
            for _ in 0..iters {
                benchmark_report(&mut perf);
            }
            start.elapsed()
        })
    });
    c.bench_function("perf_update", |b| {
        b.iter_custom(|iters| {
            let mut perf = TcpPerf::new(Arc::new(FlowPerfCounter::default()));
            let start = Instant::now();
            for _ in 0..iters {
                meta_flow_perf_update(&mut perf);
            }
            start.elapsed()
        })
    });
    c.bench_function("perf_session_peer_seq_no_assert_desc", |b| {
        b.iter(|| {
            benchmark_session_peer_seq_no_assert(true);
        })
    });
    c.bench_function("perf_session_peer_seq_no_assert", |b| {
        b.iter(|| {
            benchmark_session_peer_seq_no_assert(false);
        })
    });
}

criterion_group!(benches, bench_flow_map, bench_perf);
criterion_main!(benches);
