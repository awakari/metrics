package grpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/metrics/api/grpc/limits"
	"github.com/awakari/metrics/api/grpc/source/activitypub"
	"github.com/awakari/metrics/api/grpc/source/feeds"
	"github.com/awakari/metrics/api/grpc/source/sites"
	"github.com/awakari/metrics/api/grpc/source/telegram"
	"github.com/awakari/metrics/model"
	"github.com/awakari/metrics/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"time"
)

type Controller interface {
	ServiceServer
}

type controller struct {
	svcLimits      limits.Service
	svc            service.Service
	pubMinHourly   int64
	pubMinDaily    int64
	pubMaxHourly   int64
	pubMaxDaily    int64
	svcFeeds       feeds.Service
	svcSites       sites.Service
	svcTg          telegram.Service
	svcAp          activitypub.Service
	groupIdDefault string
}

const limitAutoExpirationDefault = 1 * time.Hour
const limitAutoExpirationThreshold = 15 * time.Minute

func NewController(
	svcLimits limits.Service,
	svcMetrics service.Service,
	pubMinHourly int64,
	pubMinDaily int64,
	pubMaxHourly int64,
	pubMaxDaily int64,
	svcFeeds feeds.Service,
	svcSites sites.Service,
	svcTg telegram.Service,
	svcAp activitypub.Service,
	groupIdDefault string,
) Controller {
	return controller{
		svcLimits:      svcLimits,
		svc:            svcMetrics,
		pubMinHourly:   pubMinHourly,
		pubMinDaily:    pubMinDaily,
		pubMaxHourly:   pubMaxHourly,
		pubMaxDaily:    pubMaxDaily,
		svcFeeds:       svcFeeds,
		svcSites:       svcSites,
		svcTg:          svcTg,
		svcAp:          svcAp,
		groupIdDefault: groupIdDefault,
	}
}

func (c controller) SetMostReadLimits(ctx context.Context, req *SetMostReadLimitsRequest) (resp *SetMostReadLimitsResponse, err error) {
	resp = &SetMostReadLimitsResponse{}
	var rateBySrc map[string]float64
	if c.svc != nil {
		resp.HourlyLimitBySource = make(map[string]int64)
		resp.DailyLimitBySource = make(map[string]int64)
		var rateSum float64
		rateSum, err = c.svc.GetRateAverage(ctx, "awk_reader_read_count", "service", "1d")
		if err == nil {
			rateBySrc, err = c.svc.GetRelativeRateByLabel(ctx, rateSum, "awk_reader_sources_read_count", "source", "1d")
		}
	}
	if err == nil && len(rateBySrc) > 0 {
		for srcUrl, rateRel := range rateBySrc {
			if rateRel > 0 {
				var groupId string
				var userId string
				switch {
				case strings.HasPrefix(srcUrl, "site:"):
					var srcSite *sites.Site
					srcSite, _ = c.svcSites.Read(ctx, srcUrl[5:])
					if srcSite != nil {
						groupId = srcSite.GroupId
						userId = srcSite.UserId
					}
				default:
					if groupId == "" {
						var srcFeed *feeds.Feed
						srcFeed, _ = c.svcFeeds.Read(ctx, srcUrl)
						if srcFeed != nil {
							groupId = srcFeed.GroupId
							userId = srcFeed.UserId
						}
					}
					if groupId == "" {
						var srcAp *activitypub.Source
						srcAp, _ = c.svcAp.Read(ctx, srcUrl)
						if srcAp != nil {
							groupId = srcAp.GroupId
							userId = srcAp.UserId
						}
					}
					if groupId == "" {
						var srcTgCh *telegram.Channel
						srcTgCh, _ = c.svcTg.Read(ctx, srcUrl)
						if srcTgCh != nil {
							groupId = srcTgCh.GroupId
							userId = srcTgCh.UserId
						}
					}
					if groupId == "" {
						var srcSite *sites.Site
						srcSite, _ = c.svcSites.Read(ctx, srcUrl[5:])
						if srcSite != nil {
							groupId = srcSite.GroupId
							userId = srcSite.UserId
						}
					}
				}
				if groupId == "" {
					srcUrl, err = c.svcAp.Create(ctx, srcUrl, c.groupIdDefault, srcUrl)
					if err != nil {
						err = nil
						continue
					}
					groupId = c.groupIdDefault
					userId = srcUrl
				}
				if userId != "" && userId != srcUrl {
					fmt.Printf("SetMostReadLimits: source %s is sharing the limit of %s, skipping the automatic limit setting\n", srcUrl, userId)
					continue
				}

				var l model.Limit

				// hourly limit
				l, err = c.svcLimits.GetRaw(ctx, groupId, srcUrl, model.SubjectPublishHourly)
				switch {
				case errors.Is(err, limits.ErrNotFound):
					fallthrough
				case !l.Expires.IsZero() && l.Expires.Before(time.Now().UTC().Add(limitAutoExpirationThreshold)):
					l.Count = c.pubMinHourly + int64(float64(c.pubMaxHourly)*rateRel)
					l.Expires = time.Now().UTC().Add(limitAutoExpirationDefault)
					err = c.svcLimits.Set(ctx, groupId, srcUrl, model.SubjectPublishHourly, l.Count, l.Expires)
					if err == nil {
						resp.HourlyLimitBySource[srcUrl] = l.Count
					}
				default:
					fmt.Printf("SetMostReadLimits: source %s has a limit that isn't expiring (%s), skipping\n", srcUrl, l.Expires)
				}

				// daily limit
				l, err = c.svcLimits.GetRaw(ctx, groupId, srcUrl, model.SubjectPublishDaily)
				switch {
				case errors.Is(err, limits.ErrNotFound):
					fallthrough
				case !l.Expires.IsZero() && l.Expires.Before(time.Now().UTC().Add(limitAutoExpirationThreshold)):
					l.Count = c.pubMinDaily + int64(float64(c.pubMaxDaily)*rateRel)
					l.Expires = time.Now().UTC().Add(limitAutoExpirationDefault)
					err = c.svcLimits.Set(ctx, groupId, srcUrl, model.SubjectPublishDaily, l.Count, l.Expires)
					if err == nil {
						resp.DailyLimitBySource[srcUrl] = l.Count
					}
				default:
					fmt.Printf("SetMostReadLimits: source %s has a limit that isn't expiring (%s), skipping\n", srcUrl, l.Expires)
				}

				err = nil
			}
		}
	}
	err = encodeError(err)
	return
}

func encodeError(src error) (dst error) {
	switch {
	case src == nil:
	case errors.Is(src, limits.ErrInternal):
		dst = status.Error(codes.Internal, src.Error())
	default:
		dst = status.Error(codes.Unknown, src.Error())
	}
	return
}
