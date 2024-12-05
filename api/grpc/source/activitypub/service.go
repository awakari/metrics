package activitypub

import (
    "context"
)

type Service interface {
    Create(ctx context.Context, addr, groupId, userId string) (url string, err error)
    Read(ctx context.Context, url string) (src *Source, err error)
}

type service struct {
    client ServiceClient
}

func NewService(client ServiceClient) Service {
    return service{
        client: client,
    }
}

func (svc service) Create(ctx context.Context, addr, groupId, userId string) (url string, err error) {
    var resp *CreateResponse
    resp, err = svc.client.Create(ctx, &CreateRequest{
        Addr:    addr,
        GroupId: groupId,
        UserId:  userId,
    })
    if resp != nil {
        url = resp.Url
    }
    return
}

func (svc service) Read(ctx context.Context, url string) (src *Source, err error) {
    var resp *ReadResponse
    resp, err = svc.client.Read(ctx, &ReadRequest{
        Url: url,
    })
    if resp != nil {
        src = resp.Src
    }
    return
}
