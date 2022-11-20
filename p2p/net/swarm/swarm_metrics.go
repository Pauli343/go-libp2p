package swarm

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/network"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

const metricNamespace = "swarm/"

var (
	connsOpened          = stats.Int64(metricNamespace+"connections_opened", "Connections Opened", stats.UnitDimensionless)
	connsClosed          = stats.Int64(metricNamespace+"connections_closed", "Connections Closed", stats.UnitDimensionless)
	connDuration         = stats.Int64(metricNamespace+"connection_duration", "Duration of a Connection", stats.UnitMilliseconds)
	connHandshakeLatency = stats.Int64(metricNamespace+"handshake_latency", "Duration of the libp2p handshake", stats.UnitMilliseconds)
)

var (
	directionTag, _ = tag.NewKey("dir")
	transportTag, _ = tag.NewKey("transport")
	securityTag, _  = tag.NewKey("security")
	muxerTag, _     = tag.NewKey("muxer")
)

func exponentialDistribution(min, max float64) []float64 {
	var v []float64
	for d := min; d < 2*max; d *= 2 {
		v = append(v, d)
	}
	return v
}

var (
	connDurationMilliseconds     = exponentialDistribution(250, float64((7 * 24 * time.Hour).Milliseconds()))
	handshakeLatencyMilliseconds = exponentialDistribution(1, 60*1000)
)

var (
	connOpenView = &view.View{
		Measure:     connsOpened,
		Aggregation: view.Sum(),
		TagKeys:     []tag.Key{directionTag, transportTag, securityTag, muxerTag},
	}
	connClosedView = &view.View{
		Measure:     connsClosed,
		Aggregation: view.Sum(),
		TagKeys:     []tag.Key{directionTag, transportTag, securityTag, muxerTag},
	}
	connDurationView = &view.View{
		Measure:     connDuration,
		Aggregation: view.Distribution(connDurationMilliseconds...),
		TagKeys:     []tag.Key{directionTag, transportTag, securityTag, muxerTag},
	}
	connHandshakeLatencyView = &view.View{
		Measure:     connHandshakeLatency,
		Aggregation: view.Distribution(handshakeLatencyMilliseconds...),
		TagKeys:     []tag.Key{transportTag, securityTag, muxerTag},
	}
)

var DefaultViews = []*view.View{connOpenView, connClosedView, connDurationView, connHandshakeLatencyView}

func getDirection(dir network.Direction) string {
	switch dir {
	case network.DirOutbound:
		return "outbound"
	case network.DirInbound:
		return "inbound"
	default:
		return "unknown"
	}
}

func appendConnectionState(tags []tag.Mutator, cs network.ConnectionState) []tag.Mutator {
	if cs.Transport == "" {
		// This shouldn't happen, unless the transport doesn't properly set the Transport field in the ConnectionState.
		tags = append(tags, tag.Upsert(transportTag, "unknown"))
	} else {
		tags = append(tags, tag.Upsert(transportTag, cs.Transport))
	}
	// Only set the security and muxer tag for transports that actually use that field.
	// For example, QUIC doesn't set security nor muxer.
	if cs.Security != "" {
		tags = append(tags, tag.Upsert(securityTag, cs.Security))
	}
	if cs.StreamMultiplexer != "" {
		tags = append(tags, tag.Upsert(muxerTag, cs.StreamMultiplexer))
	}
	return tags
}

func recordConnectionOpened(dir network.Direction, cs network.ConnectionState) {
	tags := make([]tag.Mutator, 0, 4)
	tags = append(tags, tag.Upsert(directionTag, getDirection(dir)))
	tags = appendConnectionState(tags, cs)
	stats.RecordWithTags(context.Background(), tags, connsOpened.M(1))
}

func recordConnectionClosed(dir network.Direction, cs network.ConnectionState) {
	tags := make([]tag.Mutator, 0, 4)
	tags = append(tags, tag.Upsert(directionTag, getDirection(dir)))
	tags = appendConnectionState(tags, cs)
	stats.RecordWithTags(context.Background(), tags, connsClosed.M(1))
}

func recordConnectionDuration(dir network.Direction, t time.Duration, cs network.ConnectionState) {
	tags := make([]tag.Mutator, 0, 4)
	tags = append(tags, tag.Upsert(directionTag, getDirection(dir)))
	tags = appendConnectionState(tags, cs)
	stats.RecordWithTags(context.Background(), tags, connDuration.M(t.Milliseconds()))
}

func recordHandshakeLatency(t time.Duration, cs network.ConnectionState) {
	tags := make([]tag.Mutator, 0, 3)
	tags = appendConnectionState(tags, cs)
	stats.RecordWithTags(context.Background(), tags, connHandshakeLatency.M(t.Milliseconds()))
}
