package sites

import (
    "context"
)

type Service interface {
    Read(ctx context.Context, addr string) (site *Site, err error)
}

type service struct {
    client ServiceClient
}

func NewService(client ServiceClient) Service {
    return service{
        client: client,
    }
}

func (svc service) Read(ctx context.Context, addr string) (feed *Site, err error) {
    var resp *ReadResponse
    resp, err = svc.client.Read(ctx, &ReadRequest{
        Addr: addr,
    })
    if resp != nil {
        feed = resp.Site
    }
    return
}
