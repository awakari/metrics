package interests

import (
	"context"
	"fmt"
	"github.com/awakari/metrics/util"
	grpc "google.golang.org/grpc"
	"log/slog"
)

type clientLogging struct {
	client ServiceClient
	log    *slog.Logger
}

func NewClientLogging(client ServiceClient, log *slog.Logger) ServiceClient {
	return clientLogging{
		client: client,
		log:    log,
	}
}

func (cl clientLogging) Read(ctx context.Context, req *ReadRequest, opts ...grpc.CallOption) (resp *ReadResponse, err error) {
	resp, err = cl.client.Read(ctx, req, opts...)
	cl.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("interests.Read(%s): %+v, err=%s", req.Id, resp, err))
	return
}

func (cl clientLogging) Search(ctx context.Context, req *SearchRequest, opts ...grpc.CallOption) (resp *SearchResponse, err error) {
	resp, err = cl.client.Search(ctx, req, opts...)
	ll := util.LogLevel(err)
	switch resp {
	case nil:
		cl.log.Log(ctx, ll, fmt.Sprintf("interests.Search(%+v): <nil>, err=%s", req, err))
	default:
		cl.log.Log(ctx, ll, fmt.Sprintf("interests.Search(%+v): %d, err=%s", req, len(resp.Ids), err))
	}
	return
}
