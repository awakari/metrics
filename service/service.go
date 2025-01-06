package service

import (
	"context"
	"errors"
	"fmt"
	apiPromV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"time"
)

type Service interface {
	GetRateAverage(ctx context.Context, metricName string, sumBy string, period string) (rate float64, err error)
	GetNumberHistory(ctx context.Context, metricName string) (nh NumberHistory, errs error)
	GetRelativeRateByLabel(ctx context.Context, rateSum float64, metricName string, key string, period string) (rateByKey map[string]float64, errs error)
	GetEventAttributeTypes(ctx context.Context, metric, sumBy, period string) (attrs Attributes, err error)
	GetEventAttributeValuesByName(ctx context.Context, name string) (vals []string, err error)
	GetDuration(ctx context.Context, metricName string, quantile float64, t time.Duration) (dSeconds float64, errs error)
}

type service struct {
	apiProm apiPromV1.API
}

const fmtQuerySumRate = "sum by (%s) (rate(%s[%s]))"
const fmtQueryHistogramQuantile = "histogram_quantile(%f, sum(increase(%s[%s])) by (le))"

func NewService(apiProm apiPromV1.API) Service {
	return service{
		apiProm: apiProm,
	}
}

func (svc service) GetRateAverage(ctx context.Context, metricName string, sumBy string, period string) (rate float64, errs error) {

	now := time.Now().UTC()
	q := fmt.Sprintf(fmtQuerySumRate, sumBy, metricName, period)
	v, _, err := svc.apiProm.Query(ctx, q, now)
	if err == nil {
		if v.Type() == model.ValVector {
			if vv := v.(model.Vector); len(vv) > 0 {
				rate = float64(vv[0].Value)
			}
		}
	}
	return
}

func (svc service) GetNumberHistory(ctx context.Context, metricName string) (nh NumberHistory, errs error) {

	now := time.Now().UTC()

	v, _, err := svc.apiProm.Query(ctx, metricName, now)
	if err == nil {
		if v.Type() == model.ValVector {
			if vv := v.(model.Vector); len(vv) > 0 {
				nh.Current = float64(vv[0].Value)
			}
		}
	} else {
		errs = errors.Join(errs, err)
	}

	v, _, err = svc.apiProm.Query(ctx, metricName, now.Add(-time.Hour))
	if err == nil {
		if v.Type() == model.ValVector {
			if vv := v.(model.Vector); len(vv) > 0 {
				nh.Past.Hour = float64(vv[0].Value)
			}
		}
	} else {
		errs = errors.Join(errs, err)
	}

	v, _, err = svc.apiProm.Query(ctx, metricName, now.Add(-24*time.Hour))
	if err == nil {
		if v.Type() == model.ValVector {
			if vv := v.(model.Vector); len(vv) > 0 {
				nh.Past.Day = float64(vv[0].Value)
			}
		}
	} else {
		errs = errors.Join(errs, err)
	}

	v, _, err = svc.apiProm.Query(ctx, metricName, now.Add(-30*24*time.Hour))
	if err == nil {
		if v.Type() == model.ValVector {
			if vv := v.(model.Vector); len(vv) > 0 {
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
		var v model.Value
		v, _, err = svc.apiProm.Query(ctx, q, now)
		if err == nil {
			if v.Type() == model.ValVector {
				vec := v.(model.Vector)
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

func (svc service) GetEventAttributeTypes(ctx context.Context, metric, sumBy, period string) (attrs Attributes, err error) {
	attrs.TypesByKey = make(map[string][]string)
	q := fmt.Sprintf(fmtQuerySumRate, sumBy, metric, period)
	var v model.Value
	v, _, err = svc.apiProm.Query(ctx, q, time.Now().UTC())
	if err == nil {
		if v.Type() == model.ValVector {
			vec := v.(model.Vector)
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
	var v model.Value
	v, _, err = svc.apiProm.Query(ctx, q, time.Now().UTC())
	if err == nil {
		if v.Type() == model.ValVector {
			vec := v.(model.Vector)
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
		if v.Type() == model.ValVector {
			if vv := v.(model.Vector); len(vv) > 0 {
				dSeconds = float64(vv[0].Value)
			}
		}
	} else {
		errs = errors.Join(errs, err)
	}
	return
}
