package telegram

import (
    "context"
)

type Service interface {
    Read(ctx context.Context, link string) (ch *Channel, err error)
}

type service struct {
    client ServiceClient
}

func NewService(client ServiceClient) Service {
    return service{
        client: client,
    }
}

func (svc service) Read(ctx context.Context, link string) (ch *Channel, err error) {
    var resp *ReadResponse
    resp, err = svc.client.Read(ctx, &ReadRequest{
        Link: link,
    })
    if resp != nil {
        ch = resp.Channel
    }
    return
}
