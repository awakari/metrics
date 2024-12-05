package limits

import (
	"context"
	"fmt"
	"github.com/awakari/metrics/model"
	"github.com/awakari/metrics/util"
	"log/slog"
	"time"
)

type serviceLogging struct {
	svc Service
	log *slog.Logger
}

func NewServiceLogging(svc Service, log *slog.Logger) Service {
	return serviceLogging{
		svc: svc,
		log: log,
	}
}

func (sl serviceLogging) GetRaw(ctx context.Context, groupId, userId string, subj model.Subject) (l model.Limit, err error) {
	l, err = sl.svc.GetRaw(ctx, groupId, userId, subj)
	sl.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("limits.GetRaw(%s, %s, %s): %+v, err=%s", groupId, userId, subj, l, err))
	return
}

func (sl serviceLogging) Set(ctx context.Context, groupId, userId string, subj model.Subject, count int64, expires time.Time) (err error) {
	err = sl.svc.Set(ctx, groupId, userId, subj, count, expires)
	sl.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("limits.Set(%s, %s, %s, %d, %s): err=%s", groupId, userId, subj, count, expires, err))
	return
}
