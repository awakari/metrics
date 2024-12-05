package sites

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

func (sl serviceLogging) Read(ctx context.Context, addr string) (site *Site, err error) {
    site, err = sl.svc.Read(ctx, addr)
    sl.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("sites.Read(%s): site=%+v, err=%s", addr, site, err))
    return
}
