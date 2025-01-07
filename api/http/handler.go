package http

import (
	"fmt"
	"github.com/awakari/metrics/api/grpc/auth"
	"github.com/awakari/metrics/api/grpc/interests"
	"github.com/awakari/metrics/service"
	"github.com/gin-gonic/gin"
	"math"
	"net/http"
	"sync"
	"time"
)

type Handler interface {
	GetEventAttributeTypes(ctx *gin.Context)
	GetEventAttributeValuesByName(ctx *gin.Context)
	GetPublishRate(ctx *gin.Context)
	GetReadStatus(ctx *gin.Context)
	GetFollowersCount(ctx *gin.Context)
	GetCoreDuration(ctx *gin.Context)
	GetTopInterests(ctx *gin.Context)
}

type handler struct {
	svcMetrics     service.Service
	svcInterests   interests.ServiceClient
	groupIdDefault string
}

var attrNamesBlackList = map[string]bool{
	"awakariuserid": true,
	"awkinternal":   true,
	"evtid":         true,
	"evtlink":       true,
	"reason":        true,
}
var attrNamesBuiltIn = map[string][]string{
	"": {
		"boolean",
		"bytes",
		"int32",
		"string",
		"uri",
		"uriref",
		"timestamp",
	},
	"data": {
		"bytes",
		"string",
	},
	"latitude": {
		"int32",
	},
	"longitude": {
		"int32",
	},
	"source": {
		"string",
	},
	"type": {
		"string",
	},
}

func NewHandler(svcMetrics service.Service, clientSubs interests.ServiceClient, groupIdsDefault []string) Handler {
	var groupIdDefault string
	if len(groupIdsDefault) > 0 {
		groupIdDefault = groupIdsDefault[0]
	}
	return handler{
		svcMetrics:     svcMetrics,
		svcInterests:   clientSubs,
		groupIdDefault: groupIdDefault,
	}
}

func (h handler) GetEventAttributeTypes(ctx *gin.Context) {
	attrs, err := h.svcMetrics.GetEventAttributeTypes(ctx, "awk_published_attrs_observed_count", "key, type", "1w")
	for k, _ := range attrNamesBlackList {
		delete(attrs.TypesByKey, k)
	}
	for k, typ := range attrNamesBuiltIn {
		attrs.TypesByKey[k] = typ
	}
	switch err {
	case nil:
		ctx.Header("Cache-Control", "max-age=300, public")
		ctx.Header("Date", time.Now().Format(http.TimeFormat))
		ctx.JSON(http.StatusOK, attrs)
	default:
		fmt.Printf("Get prometheus metrics failure(s): %s", err)
		ctx.JSON(http.StatusInternalServerError, attrs)
	}
	return
}

func (h handler) GetEventAttributeValuesByName(ctx *gin.Context) {
	name := ctx.Param("name")
	vals, err := h.svcMetrics.GetEventAttributeValuesByName(ctx, name)
	switch err {
	case nil:
		ctx.Header("Cache-Control", "max-age=300, public")
		ctx.Header("Date", time.Now().Format(http.TimeFormat))
		ctx.JSON(http.StatusOK, vals)
	default:
		fmt.Printf("Get prometheus metrics failure(s): %s", err)
		ctx.JSON(http.StatusInternalServerError, vals)
	}
	return
}

func (h handler) GetPublishRate(ctx *gin.Context) {
	period := ctx.Param("period")
	pubRate, err := h.svcMetrics.GetRateAverage(ctx, "awk_published_events_count", "service", period)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	d, err := time.ParseDuration(period)
	if err != nil {
		d = 24 * time.Hour // max
		err = nil
	}
	ctx.Header("Cache-Control", fmt.Sprintf("must-revalidate, public, max-age=%d", int(d.Seconds())))
	ctx.Header("Date", time.Now().Format(http.TimeFormat))
	ctx.JSON(http.StatusOK, map[string]float64{"value": pubRate})
	return
}

func (h handler) GetReadStatus(ctx *gin.Context) {
	period := ctx.Param("period")
	s := service.ReadStatus{
		SourcesMostRead: make(map[string]float64),
	}
	var err error
	s.ReadRate, err = h.svcMetrics.GetRateAverage(ctx, "awk_reader_read_count", "service", period)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	var srcs map[string]float64
	srcs, err = h.svcMetrics.GetRelativeRateByLabel(ctx, s.ReadRate, "awk_reader_sources_read_count", "source", period)
	for k, r := range srcs {
		s.SourcesMostRead[k] = r
	}
	d, err := time.ParseDuration(period)
	if err != nil {
		d = 24 * time.Hour // max
		err = nil
	}
	ctx.Header("Cache-Control", fmt.Sprintf("must-revalidate, public, max-age=%d", int(d.Seconds())))
	ctx.Header("Date", time.Now().Format(http.TimeFormat))
	ctx.JSON(http.StatusOK, s)
	return
}

func (h handler) GetFollowersCount(ctx *gin.Context) {
	uniqFollowers, err := h.svcMetrics.GetNumberHistory(ctx, "awk_followers_active_distinct_count")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	ctx.Header("Cache-Control", "max-age=300, public")
	ctx.Header("Date", time.Now().Format(http.TimeFormat))
	ctx.JSON(http.StatusOK, uniqFollowers)
	return
}

func (h handler) GetCoreDuration(ctx *gin.Context) {

	wg := sync.WaitGroup{}
	dur := &service.Duration{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		dur.Quantile05, _ = h.svcMetrics.GetDuration(ctx, "awk_duration_bucket", 0.5, 5*time.Minute)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		dur.Quantile075, _ = h.svcMetrics.GetDuration(ctx, "awk_duration_bucket", 0.75, 5*time.Minute)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		dur.Quantile095, _ = h.svcMetrics.GetDuration(ctx, "awk_duration_bucket", 0.95, 5*time.Minute)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		dur.Quantile099, _ = h.svcMetrics.GetDuration(ctx, "awk_duration_bucket", 0.99, 5*time.Minute)
	}()

	wg.Wait()
	ctx.Header("Cache-Control", "max-age=300, public")
	ctx.Header("Date", time.Now().Format(http.TimeFormat))
	ctx.JSON(http.StatusOK, dur)
	return
}

func (h handler) GetTopInterests(ctx *gin.Context) {

	topInterests := make(map[string]interests.ReadResponse)
	ctxSubs := auth.SetOutgoingAuthInfo(ctx, h.groupIdDefault, "metrics")
	resp, err := h.svcInterests.Search(ctxSubs, &interests.SearchRequest{
		Cursor: &interests.Cursor{
			Id:        "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz",
			Followers: math.MaxInt64,
		},
		Limit: 10,
		Order: interests.Order_DESC,
		Sort:  interests.Sort_FOLLOWERS,
	})

	switch err {
	case nil:
		var respRead *interests.ReadResponse
		wg := sync.WaitGroup{}
		for _, subId := range resp.Ids {
			wg.Add(1)
			go func() {
				defer wg.Done()
				respRead, err = h.svcInterests.Read(ctxSubs, &interests.ReadRequest{
					Id: subId,
				})
				if err == nil && respRead.Public {
					topInterests[subId] = interests.ReadResponse{
						Description: respRead.Description,
						Followers:   respRead.Followers,
					}
				}
			}()
		}
		wg.Wait()
	default:
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}

	ctx.Header("Cache-Control", "max-age=300, public")
	ctx.Header("Date", time.Now().Format(http.TimeFormat))
	ctx.JSON(http.StatusOK, topInterests)
	return
}
