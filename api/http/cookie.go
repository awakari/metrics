package http

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "github.com/awakari/metrics/config"
    "github.com/awakari/metrics/model"
    "github.com/gin-gonic/gin"
    "net/http"
    "time"
)

type CookieHandler interface {
    Handle(ctx *gin.Context)
}

type cookieHandler struct {
    cfg config.CookieConfig
}

const HeaderRetryAfter = "Retry-After"
const ValueRetryAfter = "1"

func NewCookieHandler(cfg config.CookieConfig) CookieHandler {
    return cookieHandler{
        cfg: cfg,
    }
}

func (ch cookieHandler) Handle(ctx *gin.Context) {
    auth := ctx.GetHeader("Authorization")
    if auth == "" { // no token auth in the request
        fp := fingerprint(ctx)
        h := hmac.New(sha256.New224, []byte(ch.cfg.Secret))
        expected := base64.URLEncoding.EncodeToString(h.Sum([]byte(fp)))
        actual, _ := ctx.Cookie(model.PrefixUserIdTmp)
        switch {
        case actual == expected:
            ctx.Set(model.KeyGroupId, ctx.GetHeader(model.KeyGroupId))
            ctx.Set(model.KeyUserId, model.PrefixUserIdTmp+actual)
        default:
            ctx.SetCookie(model.PrefixUserIdTmp, expected, int(ch.cfg.MaxAge/time.Second), ch.cfg.Path, ch.cfg.Domain, ch.cfg.Secure, ch.cfg.HttpOnly)
            ctx.Writer.Header().Add(HeaderRetryAfter, ValueRetryAfter)
            ctx.Status(http.StatusServiceUnavailable)
        }
    }
}
