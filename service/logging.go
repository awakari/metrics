package service

import (
	"context"
	"fmt"
	"github.com/awakari/metrics/util"
	"log/slog"
	"time"
)

type logging struct {
	svc Service
	log *slog.Logger
}

func NewLogging(svc Service, log *slog.Logger) Service {
	return logging{
		svc: svc,
		log: log,
	}
}
func (l logging) GetRateAverage(ctx context.Context, metricName string, sumBy string, rate *RateAverage) (errs error) {
	errs = l.svc.GetRateAverage(ctx, metricName, sumBy, rate)
	l.log.Log(ctx, util.LogLevel(errs), fmt.Sprintf("service.GetRateAverage(%s, %s): %v, %s", metricName, sumBy, rate, errs))
	return
}

func (l logging) GetNumberHistory(ctx context.Context, metricName string) (nh NumberHistory, errs error) {
	nh, errs = l.svc.GetNumberHistory(ctx, metricName)
	l.log.Log(ctx, util.LogLevel(errs), fmt.Sprintf("service.GetNumberHistory(%s): %v, %s", metricName, nh, errs))
	return
}

func (l logging) GetRelativeRateByLabel(ctx context.Context, rateSum RateAverage, metricName string, key string) (rateByKey map[string]RateAverage, errs error) {
	rateByKey, errs = l.svc.GetRelativeRateByLabel(ctx, rateSum, metricName, key)
	l.log.Log(ctx, util.LogLevel(errs), fmt.Sprintf("service.GetRelativeRateByLabel(%v, %s, %s): %d, %s", rateSum, metricName, key, len(rateByKey), errs))
	return
}

func (l logging) GetEventAttributeTypes(ctx context.Context, metric, sumBy, period string) (attrs Attributes, err error) {
	attrs, err = l.svc.GetEventAttributeTypes(ctx, metric, sumBy, period)
	l.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("service.GetEventAttributeTypes(%s, %s, %s): %v, %s", metric, sumBy, period, attrs, err))
	return
}

func (l logging) GetEventAttributeValuesByName(ctx context.Context, name string) (vals []string, err error) {
	vals, err = l.svc.GetEventAttributeValuesByName(ctx, name)
	l.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("service.GetEventAttributeValuesByName(%s): %d, %s", name, len(vals), err))
	return
}

func (l logging) GetDuration(ctx context.Context, metricName string, quantile float64, t time.Duration) (dSeconds float64, errs error) {
	dSeconds, errs = l.svc.GetDuration(ctx, metricName, quantile, t)
	l.log.Log(ctx, util.LogLevel(errs), fmt.Sprintf("service.GetDuration(%s, %f, %s): %f, %s", metricName, quantile, t, dSeconds, errs))
	return
}
