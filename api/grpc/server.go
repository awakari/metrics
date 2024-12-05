package grpc

import (
    "fmt"
    grpcLimits "github.com/awakari/metrics/api/grpc/limits"
    "github.com/awakari/metrics/api/grpc/source/activitypub"
    "github.com/awakari/metrics/api/grpc/source/feeds"
    "github.com/awakari/metrics/api/grpc/source/sites"
    "github.com/awakari/metrics/api/grpc/source/telegram"
    "github.com/awakari/metrics/config"
    "github.com/awakari/metrics/service"
    "google.golang.org/grpc"
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
    "google.golang.org/grpc/reflection"
    "net"
)

func Serve(
    cfg config.Config,
    svcLimits grpcLimits.Service,
    svcMetrics service.Service,
    svcSrcFeeds feeds.Service,
    svcSrcSites sites.Service,
    svcSrcTg telegram.Service,
    svcSrcAp activitypub.Service,
) (err error) {
    adminSrv := grpc.NewServer()
    reflection.Register(adminSrv)
    grpc_health_v1.RegisterHealthServer(adminSrv, health.NewServer())
    controllerAdmin := NewController(
        svcLimits,
        svcMetrics,
        cfg.Limits.Default.User.PublishMessages,
        cfg.Limits.Max.User.PublishMessages,
        svcSrcFeeds,
        svcSrcSites,
        svcSrcTg,
        svcSrcAp,
        cfg.Limits.Default.Groups[0],
    )
    RegisterServiceServer(adminSrv, controllerAdmin)
    reflection.Register(adminSrv)
    grpc_health_v1.RegisterHealthServer(adminSrv, health.NewServer())
    adminConn, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Api.Port))
    if err == nil {
        err = adminSrv.Serve(adminConn)
    }
    return
}
