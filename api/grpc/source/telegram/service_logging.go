package telegram

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

func (sl serviceLogging) Read(ctx context.Context, link string) (ch *Channel, err error) {
    ch, err = sl.svc.Read(ctx, link)
    sl.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("grpc.source.telegram.Read(%s): %+v, %s", link, ch, err))
    return
}
