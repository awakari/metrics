package service

import (
	"context"
	"errors"
	"fmt"
	apiPromV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"sync"
	"time"
)

type Service interface {
	GetRateAverage(ctx context.Context, metricName string, sumBy string, rate *RateAverage) (errs error)
	GetNumberHistory(ctx context.Context, metricName string) (nh NumberHistory, errs error)
	GetRelativeRateByLabel(ctx context.Context, rateSum RateAverage, metricName string, key string) (rateByKey map[string]RateAverage, errs error)
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

func (svc service) GetRateAverage(ctx context.Context, metricName string, sumBy string, rate *RateAverage) (errs error) {

	now := time.Now().UTC()
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		q := fmt.Sprintf(fmtQuerySumRate, sumBy, metricName, "5m")
		v, _, err := svc.apiProm.Query(ctx, q, now)
		if err == nil {
			if v.Type() == model.ValVector {
				if vv := v.(model.Vector); len(vv) > 0 {
					rate.Min5 = float64(vv[0].Value)
				}
			}
		} else {
			errs = errors.Join(errs, err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		q := fmt.Sprintf(fmtQuerySumRate, sumBy, metricName, "1h")
		v, _, err := svc.apiProm.Query(ctx, q, now)
		if err == nil {
			if v.Type() == model.ValVector {
				if vv := v.(model.Vector); len(vv) > 0 {
					rate.Hour = float64(vv[0].Value)
				}
			}
		} else {
			errs = errors.Join(errs, err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		q := fmt.Sprintf(fmtQuerySumRate, sumBy, metricName, "1d")
		v, _, err := svc.apiProm.Query(ctx, q, now)
		if err == nil {
			if v.Type() == model.ValVector {
				if vv := v.(model.Vector); len(vv) > 0 {
					rate.Day = float64(vv[0].Value)
				}
			}
		} else {
			errs = errors.Join(errs, err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		q := fmt.Sprintf(fmtQuerySumRate, sumBy, metricName, "30d")
		v, _, err := svc.apiProm.Query(ctx, q, now)
		if err == nil {
			if v.Type() == model.ValVector {
				if vv := v.(model.Vector); len(vv) > 0 {
					rate.Month = float64(vv[0].Value)
				}
			}
		} else {
			errs = errors.Join(errs, err)
		}
	}()

	wg.Wait()

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

func (svc service) GetRelativeRateByLabel(ctx context.Context, rateSum RateAverage, metricName string, key string) (rateByKey map[string]RateAverage, errs error) {
	rateByKey = make(map[string]RateAverage)
	now := time.Now().UTC()
	if rateSum.Day > 0 {
		q := fmt.Sprintf(fmtQuerySumRate, key, metricName, "1d")
		v, _, err := svc.apiProm.Query(ctx, q, now)
		if err == nil {
			if v.Type() == model.ValVector {
				vec := v.(model.Vector)
				for _, rec := range vec {
					for _, lblVal := range rec.Metric {
						rateRatio := float64(rec.Value) / rateSum.Day
						if rateRatio > 0 {
							rateByKey[string(lblVal)] = RateAverage{
								Day: rateRatio,
							}
						}
					}
				}
			}
		} else {
			errs = errors.Join(errs, err)
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
