package src

import (
	"github.com/awakari/metrics/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type Handler interface {
	FeedCount(ctx *gin.Context)
	SocialCount(ctx *gin.Context)
	RealtimeCount(ctx *gin.Context)
}

type handler struct {
	svcMetrics service.Service
}

func NewHandler(svcMetrics service.Service) Handler {
	return handler{
		svcMetrics: svcMetrics,
	}
}

func (h handler) FeedCount(ctx *gin.Context) {
	countHistory, err := h.svcMetrics.GetNumberHistory(ctx, "awk_source_feeds_count_pull")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	ctx.Header("Cache-Control", "max-age=300, public")
	ctx.Header("Date", time.Now().Format(http.TimeFormat))
	ctx.JSON(http.StatusOK, countHistory)
	return
}

func (h handler) SocialCount(ctx *gin.Context) {
	countHistory, err := h.svcMetrics.GetNumberHistory(ctx, "awk_source_activitypub_count_total")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	ctx.Header("Cache-Control", "max-age=300, public")
	ctx.Header("Date", time.Now().Format(http.TimeFormat))
	ctx.JSON(http.StatusOK, countHistory)

}

func (h handler) RealtimeCount(ctx *gin.Context) {
	countHistory, err := h.svcMetrics.GetNumberHistory(ctx, "awk_source_feeds_count_push")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	ctx.Header("Cache-Control", "max-age=300, public")
	ctx.Header("Date", time.Now().Format(http.TimeFormat))
	ctx.JSON(http.StatusOK, countHistory)

}
