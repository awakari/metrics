package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/metrics/model"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/jellydator/ttlcache/v3"
	apiPromV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus"
	modelProm "github.com/prometheus/common/model"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Service interface {
	GetRateAverage(ctx context.Context, metricName string, sumBy string, period string) (rate float64, err error)
	GetNumberHistory(ctx context.Context, metricName string) (nh model.NumberHistory, errs error)
	GetRelativeRateByLabel(ctx context.Context, rateSum float64, metricName string, key string, period string) (rateByKey map[string]float64, errs error)
	GetEventAttributeTypes(ctx context.Context, metric, sumBy, period string) (attrs model.Attributes, err error)
	GetEventAttributeValuesByName(ctx context.Context, name string) (vals []string, err error)
	GetDuration(ctx context.Context, metricName string, quantile float64, t time.Duration) (dSeconds float64, errs error)
	AccountRead(ctx context.Context, msgs []*pb.CloudEvent, push bool) (err error)
}

type service struct {
	apiProm              apiPromV1.API
	readStatsLimit       int
	readCounter          prometheus.Counter
	readCounterBySrc     *prometheus.CounterVec
	readCounterCache     *ttlcache.Cache[string, *atomic.Uint32]
	readCounterCacheLock sync.Locker
	dur                  prometheus.Histogram
}

const fmtQuerySumRate = "sum by (%s) (rate(%s[%s]))"
const fmtQueryHistogramQuantile = "histogram_quantile(%f, sum(increase(%s[%s])) by (le))"

func NewService(
	apiProm apiPromV1.API,
	readStatsLimit uint32,
	readCounter prometheus.Counter,
	readCounterBySrc *prometheus.CounterVec,
	readCounterCache *ttlcache.Cache[string, *atomic.Uint32],
	dur prometheus.Histogram,
) Service {
	return service{
		apiProm:              apiProm,
		readStatsLimit:       int(readStatsLimit),
		readCounter:          readCounter,
		readCounterBySrc:     readCounterBySrc,
		readCounterCache:     readCounterCache,
		readCounterCacheLock: &sync.Mutex{},
		dur:                  dur,
	}
}

func (svc service) GetRateAverage(ctx context.Context, metricName string, sumBy string, period string) (rate float64, errs error) {

	now := time.Now().UTC()
	q := fmt.Sprintf(fmtQuerySumRate, sumBy, metricName, period)
	v, _, err := svc.apiProm.Query(ctx, q, now)
	if err == nil {
		if v.Type() == modelProm.ValVector {
			if vv := v.(modelProm.Vector); len(vv) > 0 {
				rate = float64(vv[0].Value)
			}
		}
	}
	return
}

func (svc service) GetNumberHistory(ctx context.Context, metricName string) (nh model.NumberHistory, errs error) {

	now := time.Now().UTC()

	v, _, err := svc.apiProm.Query(ctx, metricName, now)
	if err == nil {
		if v.Type() == modelProm.ValVector {
			if vv := v.(modelProm.Vector); len(vv) > 0 {
				nh.Current = float64(vv[0].Value)
			}
		}
	} else {
		errs = errors.Join(errs, err)
	}

	v, _, err = svc.apiProm.Query(ctx, metricName, now.Add(-time.Hour))
	if err == nil {
		if v.Type() == modelProm.ValVector {
			if vv := v.(modelProm.Vector); len(vv) > 0 {
				nh.Past.Hour = float64(vv[0].Value)
			}
		}
	} else {
		errs = errors.Join(errs, err)
	}

	v, _, err = svc.apiProm.Query(ctx, metricName, now.Add(-24*time.Hour))
	if err == nil {
		if v.Type() == modelProm.ValVector {
			if vv := v.(modelProm.Vector); len(vv) > 0 {
				nh.Past.Day = float64(vv[0].Value)
			}
		}
	} else {
		errs = errors.Join(errs, err)
	}

	v, _, err = svc.apiProm.Query(ctx, metricName, now.Add(-30*24*time.Hour))
	if err == nil {
		if v.Type() == modelProm.ValVector {
			if vv := v.(modelProm.Vector); len(vv) > 0 {
				nh.Past.Month = float64(vv[0].Value)
			}
		}
	} else {
		errs = errors.Join(errs, err)
	}

	return
}

func (svc service) GetRelativeRateByLabel(ctx context.Context, rateSum float64, metricName string, key string, period string) (rateByKey map[string]float64, err error) {
	rateByKey = make(map[string]float64)
	now := time.Now().UTC()
	if rateSum > 0 {
		q := fmt.Sprintf(fmtQuerySumRate, key, metricName, period)
		var v modelProm.Value
		v, _, err = svc.apiProm.Query(ctx, q, now)
		if err == nil {
			if v.Type() == modelProm.ValVector {
				vec := v.(modelProm.Vector)
				for _, rec := range vec {
					for _, lblVal := range rec.Metric {
						rateRatio := float64(rec.Value) / rateSum
						if rateRatio > 0 {
							rateByKey[string(lblVal)] = rateRatio
						}
					}
				}
			}
		}
	}
	return
}

func (svc service) GetEventAttributeTypes(ctx context.Context, metric, sumBy, period string) (attrs model.Attributes, err error) {
	attrs.TypesByKey = make(map[string][]string)
	q := fmt.Sprintf(fmtQuerySumRate, sumBy, metric, period)
	var v modelProm.Value
	v, _, err = svc.apiProm.Query(ctx, q, time.Now().UTC())
	if err == nil {
		if v.Type() == modelProm.ValVector {
			vec := v.(modelProm.Vector)
			for _, rec := range vec {
				var key, typ string
				for lblName, lblValue := range rec.Metric {
					switch lblName {
					case "key":
						key = string(lblValue)
					case "type":
						typ = string(lblValue)
					}
				}
				if key != "" && typ != "" {
					types := attrs.TypesByKey[key]
					types = append(types, typ)
					attrs.TypesByKey[key] = types
				}
			}
		}
	}
	return
}

func (svc service) GetEventAttributeValuesByName(ctx context.Context, name string) (vals []string, err error) {
	q := fmt.Sprintf(fmtQuerySumRate, name, "awk_published_events_count", "1w")
	var v modelProm.Value
	v, _, err = svc.apiProm.Query(ctx, q, time.Now().UTC())
	if err == nil {
		if v.Type() == modelProm.ValVector {
			vec := v.(modelProm.Vector)
			for _, rec := range vec {
				for _, val := range rec.Metric {
					vals = append(vals, string(val))
				}
			}
		}
	}
	return
}

func (svc service) GetDuration(ctx context.Context, metricName string, quantile float64, t time.Duration) (dSeconds float64, errs error) {
	q := fmt.Sprintf(fmtQueryHistogramQuantile, quantile, metricName, t)
	v, _, err := svc.apiProm.Query(ctx, q, time.Now().UTC())
	if err == nil {
		if v.Type() == modelProm.ValVector {
			if vv := v.(modelProm.Vector); len(vv) > 0 {
				dSeconds = float64(vv[0].Value)
			}
		}
	} else {
		errs = errors.Join(errs, err)
	}
	return
}

func (svc service) AccountRead(ctx context.Context, msgs []*pb.CloudEvent, push bool) (err error) {
	svc.readCounter.Add(float64(len(msgs)))
	var srcs []string
	var durationAccounted bool
	for _, msg := range msgs {
		if msg != nil {
			svc.accountReadTopSources(ctx, msg.Source)
			srcs = append(srcs, msg.Source)
			if push && !durationAccounted {
				attrTsPub, attrTsPubOk := msg.Attributes[model.CeKeyAwkPubTime]
				if attrTsPubOk {
					if err = svc.accountDuration(ctx, attrTsPub); err == nil {
						durationAccounted = true
					}
				}
			}
		}
	}
	topMostReadSrcs := svc.topMostReadSources()
	for _, src := range srcs {
		if topMostReadSrcs[src] {
			svc.readCounterBySrc.With(prometheus.Labels{model.KeySrc: src}).Inc()
		} else {
			svc.readCounterBySrc.Delete(prometheus.Labels{model.KeySrc: src})
		}
	}
	return
}

func (svc service) accountReadTopSources(_ context.Context, src string) {
	counterSrcRuntime := svc.readCounterCache.Get(src)
	if counterSrcRuntime == nil {
		counterSrcRuntime = svc.readCounterCache.Set(src, &atomic.Uint32{}, ttlcache.DefaultTTL)
	}
	counterSrcRuntime.Value().Add(1)
	return
}

func (svc service) accountDuration(_ context.Context, attrTsPub *pb.CloudEventAttributeValue) (err error) {
	var tPub time.Time
	tsPub := attrTsPub.GetCeTimestamp()
	switch tsPub {
	case nil:
		tPub, err = time.Parse(time.RFC3339Nano, attrTsPub.GetCeString())
	default:
		tPub = tsPub.AsTime().UTC()
	}
	var latency time.Duration
	if err == nil {
		latency = time.Now().UTC().Sub(tPub)
	}
	if latency > 0 {
		svc.dur.Observe(float64(latency) / float64(time.Second))
	}
	return
}

func (svc service) topMostReadSources() (topMostReadSrcs map[string]bool) {

	// get all cache entries
	items := svc.getCounterCacheSnapshot()

	// sort by counter value in the descending order
	sort.Slice(items, func(i, j int) bool {
		return items[i].Value().Load() > items[j].Value().Load()
	})

	// limit most read
	if len(items) > svc.readStatsLimit {
		items = items[:svc.readStatsLimit]
	}

	topMostReadSrcs = make(map[string]bool)
	for _, item := range items {
		topMostReadSrcs[item.Key()] = true
	}

	return
}

func (svc service) getCounterCacheSnapshot() (items []*ttlcache.Item[string, *atomic.Uint32]) {
	svc.readCounterCacheLock.Lock()
	defer svc.readCounterCacheLock.Unlock()
	for _, item := range svc.readCounterCache.Items() {
		items = append(items, item)
	}
	return
}
