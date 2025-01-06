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
	pubMin         int64
	pubMax         int64
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
	pubMin int64,
	pubMax int64,
	svcFeeds feeds.Service,
	svcSites sites.Service,
	svcTg telegram.Service,
	svcAp activitypub.Service,
	groupIdDefault string,
) Controller {
	return controller{
		svcLimits:      svcLimits,
		svc:            svcMetrics,
		pubMin:         pubMin,
		pubMax:         pubMax,
		svcFeeds:       svcFeeds,
		svcSites:       svcSites,
		svcTg:          svcTg,
		svcAp:          svcAp,
		groupIdDefault: groupIdDefault,
	}
}

func (c controller) SetMostReadLimits(ctx context.Context, req *SetMostReadLimitsRequest) (resp *SetMostReadLimitsResponse, err error) {
	resp = &SetMostReadLimitsResponse{}
	var rateBySrc map[string]service.RateAverage
	if c.svc != nil {
		resp.LimitBySource = make(map[string]int64)
		var rateSum service.RateAverage
		err = c.svc.GetRateAverage(ctx, "awk_reader_read_count", "service", &rateSum)
		if err == nil {
			rateBySrc, err = c.svc.GetRelativeRateByLabel(ctx, rateSum, "awk_reader_sources_read_count", "source")
		}
	}
	if err == nil && len(rateBySrc) > 0 {
		for srcUrl, rateRel := range rateBySrc {
			if rateRel.Day > 0 {
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
				l, err = c.svcLimits.GetRaw(ctx, groupId, srcUrl, model.SubjectPublishEvents)
				switch {
				case errors.Is(err, limits.ErrNotFound):
					fallthrough
				case !l.Expires.IsZero() && l.Expires.Before(time.Now().UTC().Add(limitAutoExpirationThreshold)):
					l.Count = c.pubMin + int64(float64(c.pubMax)*rateRel.Day)
					l.Expires = time.Now().UTC().Add(limitAutoExpirationDefault)
					err = c.svcLimits.Set(ctx, groupId, srcUrl, model.SubjectPublishEvents, l.Count, l.Expires)
					if err == nil {
						resp.LimitBySource[srcUrl] = l.Count
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
