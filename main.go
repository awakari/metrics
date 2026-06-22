package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync/atomic"

	apiGrpc "github.com/awakari/metrics/api/grpc"
	apiGrpcInterests "github.com/awakari/metrics/api/grpc/interests"
	apiGrpcLimits "github.com/awakari/metrics/api/grpc/limits"
	apiGrpcSrcAp "github.com/awakari/metrics/api/grpc/source/activitypub"
	apiGrpcSrcFeeds "github.com/awakari/metrics/api/grpc/source/feeds"
	apiGrpcSrcSites "github.com/awakari/metrics/api/grpc/source/sites"
	apiGrpcSrcTg "github.com/awakari/metrics/api/grpc/source/telegram"
	apiHttp "github.com/awakari/metrics/api/http"
	apiHttpSrc "github.com/awakari/metrics/api/http/src"
	"github.com/awakari/metrics/config"
	"github.com/awakari/metrics/model"
	"github.com/awakari/metrics/service"
	"github.com/gin-gonic/gin"
	"github.com/jellydator/ttlcache/v3"
	grpcpool "github.com/processout/grpc-go-pool"
	apiProm "github.com/prometheus/client_golang/api"
	apiPromV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {

	slog.Info("starting...")
	cfg, err := config.NewConfigFromEnv()
	if err != nil {
		panic(err)
	}
	opts := slog.HandlerOptions{
		Level: slog.Level(cfg.Log.Level),
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &opts))

	clientProm, err := apiProm.NewClient(apiProm.Config{
		Address: cfg.Api.Prometheus.Uri,
	})
	var ap apiPromV1.API
	switch err {
	case nil:
		ap = apiPromV1.NewAPI(clientProm)
	default:
		panic(err)
	}

	rCounter := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "awk_reader_read_count",
			Help: "Awakari Reader: total count of messages been read",
		},
	)
	prometheus.MustRegister(rCounter)
	rCounterBySrc := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "awk_reader_sources_read_count",
			Help: "Awakari Reader: most read source read count",
		},
		[]string{
			model.KeySrc,
		},
	)
	prometheus.MustRegister(rCounterBySrc)
	srcStatsCache := ttlcache.New[string, *atomic.Uint32](
		ttlcache.WithTTL[string, *atomic.Uint32](cfg.Api.Source.MostRead.Ttl),
		ttlcache.WithCapacity[string, *atomic.Uint32](cfg.Api.Source.MostRead.Capacity),
	)
	metricDuration := promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "awk_duration",
		Help: "Awakari core events processing duration",
		Buckets: []float64{
			30,
			100,
			300,
			1000,
			3000,
			10000,
		},
	})

	svc := service.NewService(
		ap,
		cfg.Api.Source.MostRead.Limit,
		rCounter,
		rCounterBySrc,
		srcStatsCache,
		metricDuration,
	)
	svc = service.NewLogging(svc, log)

	connPoolInterests, err := grpcpool.New(
		func() (*grpc.ClientConn, error) {
			return grpc.NewClient(cfg.Api.Interests.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
		},
		int(cfg.Api.Interests.Connection.Count.Init),
		int(cfg.Api.Interests.Connection.Count.Max),
		cfg.Api.Interests.Connection.IdleTimeout,
	)
	if err != nil {
		panic(err)
	}
	defer connPoolInterests.Close()
	clientInterests := apiGrpcInterests.NewClientPool(connPoolInterests)
	clientInterests = apiGrpcInterests.NewClientLogging(clientInterests, log)

	// init the source-feeds client
	connSrcFeeds, err := grpc.NewClient(cfg.Api.Source.Feeds.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		log.Info("connected the source-feeds service")
		defer connSrcFeeds.Close()
	} else {
		log.Error(fmt.Sprintf("failed to connect the source-feeds service: %s", err))
	}
	clientSrcFeeds := apiGrpcSrcFeeds.NewServiceClient(connSrcFeeds)
	svcSrcFeeds := apiGrpcSrcFeeds.NewService(clientSrcFeeds)
	svcSrcFeeds = apiGrpcSrcFeeds.NewServiceLogging(svcSrcFeeds, log)

	// init the source-telegram client
	connSrcTg, err := grpc.NewClient(cfg.Api.Source.Telegram.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		log.Info("connected the source-telegram service")
		defer connSrcTg.Close()
	} else {
		log.Error(fmt.Sprintf("failed to connect the source-telegram service: %s", err))
	}
	clientSrcTg := apiGrpcSrcTg.NewServiceClient(connSrcTg)
	svcSrcTg := apiGrpcSrcTg.NewService(clientSrcTg)
	svcSrcTg = apiGrpcSrcTg.NewServiceLogging(svcSrcTg, log)

	// init the source-sites client
	connSrcSites, err := grpc.NewClient(cfg.Api.Source.Sites.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		log.Info("connected the source-sites service")
		defer connSrcSites.Close()
	} else {
		log.Error(fmt.Sprintf("failed to connect the source-sites service: %s", err))
	}
	clientSrcSites := apiGrpcSrcSites.NewServiceClient(connSrcSites)
	svcSrcSites := apiGrpcSrcSites.NewService(clientSrcSites)
	svcSrcSites = apiGrpcSrcSites.NewServiceLogging(svcSrcSites, log)

	// init the int-activitypub client
	connSrcAp, err := grpc.NewClient(cfg.Api.Source.ActivityPub.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		log.Info("connected the int-activitypub service")
		defer connSrcAp.Close()
	} else {
		log.Error(fmt.Sprintf("failed to connect the int-activitypub service: %s", err))
	}
	clientSrcAp := apiGrpcSrcAp.NewServiceClient(connSrcAp)
	svcSrcAp := apiGrpcSrcAp.NewService(clientSrcAp)
	svcSrcAp = apiGrpcSrcAp.NewLogging(svcSrcAp, log)

	connPoolLimits, err := grpcpool.New(
		func() (*grpc.ClientConn, error) {
			return grpc.NewClient(cfg.Api.Usage.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
		},
		int(cfg.Api.Usage.Connection.Count.Init),
		int(cfg.Api.Usage.Connection.Count.Max),
		cfg.Api.Usage.Connection.IdleTimeout,
	)
	if err != nil {
		panic(err)
	}
	defer connPoolLimits.Close()
	clientLimits := apiGrpcLimits.NewClientPool(connPoolLimits)
	svcLimits := apiGrpcLimits.NewService(clientLimits)
	svcLimits = apiGrpcLimits.NewServiceLogging(svcLimits, log)

	handlerCookies := apiHttp.NewCookieHandler(cfg.Api.Http.Cookie)

	r := gin.Default()
	handlerStatus := apiHttp.NewHandler(svc, clientInterests, cfg.Limits.Default.Groups)
	r.
		Group("/v1/public", handlerCookies.Handle).
		GET("/pub-rate/:period", handlerStatus.GetPublishRate).
		GET("/read/:period", handlerStatus.GetReadStatus).
		GET("/subscriptions", handlerStatus.GetSubscriptionsCount).
		GET("/top-interests", handlerStatus.GetTopInterests).
		GET("/new-interests", handlerStatus.GetNewInterests).
		GET("/duration", handlerStatus.GetCoreDuration)
	r.
		Group("/v1/attr", handlerCookies.Handle).
		GET("/types", handlerStatus.GetEventAttributeTypes).
		GET("/values/:name", handlerStatus.GetEventAttributeValuesByName)

	handlerSrc := apiHttpSrc.NewHandler(svc)
	r.
		Group("/v1/src", handlerCookies.Handle).
		GET("/feeds", handlerSrc.FeedCount).
		GET("/socials", handlerSrc.SocialCount).
		GET("/realtime", handlerSrc.RealtimeCount)
	go func() {
		err = r.Run(fmt.Sprintf(":%d", cfg.Api.Http.Port))
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(fmt.Sprintf(":%d", cfg.Api.Metrics.Port), nil)
	}()

	log.Info(fmt.Sprintf("starting to listen the API @ port #%d...", cfg.Api.Port))
	err = apiGrpc.Serve(
		cfg,
		svcLimits,
		svc,
		svcSrcFeeds,
		svcSrcSites,
		svcSrcTg,
		svcSrcAp,
	)
	if err != nil {
		panic(err)
	}
}
