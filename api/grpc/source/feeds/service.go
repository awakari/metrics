package feeds

import (
    "context"
)

type Service interface {
    Read(ctx context.Context, url string) (feed *Feed, err error)
}

type service struct {
    client ServiceClient
}

func NewService(client ServiceClient) Service {
    return service{
        client: client,
    }
}

func (svc service) Read(ctx context.Context, url string) (feed *Feed, err error) {
    var resp *ReadResponse
    resp, err = svc.client.Read(ctx, &ReadRequest{
        Url: url,
    })
    if resp != nil {
        feed = resp.Feed
    }
    return
}
