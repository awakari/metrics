package limits

import (
    "context"
    "fmt"
    "github.com/awakari/metrics/model"
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
    defer func() {
        sl.log.Debug(fmt.Sprintf("limits.GetRaw(%s, %s, %s): %+v, err=%s", groupId, userId, subj, l, err))
    }()
    return sl.svc.GetRaw(ctx, groupId, userId, subj)
}

func (sl serviceLogging) Set(ctx context.Context, groupId, userId string, subj model.Subject, count int64, expires time.Time) (err error) {
    defer func() {
        sl.log.Debug(fmt.Sprintf("limits.Set(%s, %s, %s, %d, %s): err=%s", groupId, userId, subj, count, expires, err))
    }()
    return sl.svc.Set(ctx, groupId, userId, subj, count, expires)
}
