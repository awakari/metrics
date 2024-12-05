package limits

import (
    "context"
    "errors"
    "fmt"
    "github.com/awakari/metrics/api/grpc/auth"
    "github.com/awakari/metrics/api/grpc/subject"
    "github.com/awakari/metrics/model"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/timestamppb"
    "io"
    "time"
)

type Service interface {
    GetRaw(ctx context.Context, groupId, userId string, subj model.Subject) (l model.Limit, err error)
    Set(ctx context.Context, groupId, userId string, subj model.Subject, count int64, expires time.Time) (err error)
}

type service struct {
    client ServiceClient
}

var ErrInternal = errors.New("internal failure")
var ErrInvalid = errors.New("invalid")
var ErrNotFound = errors.New("not found")
var ErrForbidden = errors.New("forbidden")

func NewService(
    client ServiceClient,
) Service {
    return service{
        client: client,
    }
}

func (svc service) GetRaw(ctx context.Context, groupId, userId string, subj model.Subject) (l model.Limit, err error) {
    req := GetRequest{
        Raw: true,
    }
    var resp *GetResponse
    req.Subj, err = subject.Encode(subj)
    if err == nil {
        ctxAuth := auth.SetOutgoingAuthInfo(ctx, groupId, userId)
        resp, err = svc.client.Get(ctxAuth, &req)
    }
    if err == nil {
        l.Count = resp.Count
        l.UserId = resp.UserId
        if resp.Expires != nil {
            l.Expires = resp.Expires.AsTime()
        }
    }
    err = decodeError(err)
    return
}

func (svc service) Set(ctx context.Context, groupId, userId string, subj model.Subject, count int64, expires time.Time) (err error) {
    req := SetRequest{
        Count:   count,
        UserId:  userId,
        GroupId: groupId,
    }
    req.Subj, err = subject.Encode(subj)
    if !expires.IsZero() {
        req.Expires = timestamppb.New(expires.UTC())
    }
    _, err = svc.client.Set(ctx, &req)
    err = decodeError(err)
    return
}

func decodeError(src error) (dst error) {
    switch {
    case src == io.EOF:
        dst = src // return as it is
    case status.Code(src) == codes.OK:
        dst = nil
    case status.Code(src) == codes.InvalidArgument:
        dst = fmt.Errorf("%w: %s", ErrInvalid, src)
    case status.Code(src) == codes.NotFound:
        dst = fmt.Errorf("%w: %s", ErrNotFound, src)
    case status.Code(src) == codes.Unauthenticated:
        dst = fmt.Errorf("%w: %s", ErrForbidden, src)
    default:
        dst = fmt.Errorf("%w: %s", ErrInternal, src)
    }
    return
}
