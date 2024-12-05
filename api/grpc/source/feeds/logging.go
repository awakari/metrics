package feeds

import (
    "context"
    "fmt"
    "github.com/awakari/metrics/util"
    "log/slog"
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

func (sl serviceLogging) Read(ctx context.Context, url string) (feed *Feed, err error) {
    feed, err = sl.svc.Read(ctx, url)
    sl.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("feeds.Read(%s): err=%s", url, err))
    return
}
