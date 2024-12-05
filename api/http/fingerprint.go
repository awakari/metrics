package http

import (
    "github.com/gin-gonic/gin"
    "strings"
)

func fingerprint(ctx *gin.Context) (fp string) {
    fp = strings.Join(
        []string{
            ctx.ClientIP(),
            ctx.GetHeader("accept"),
            ctx.GetHeader("accept-encoding"),
            ctx.GetHeader("accept-language"),
            ctx.GetHeader("connection"),
            ctx.GetHeader("host"),
            ctx.GetHeader("user-agent"),
        },
        "\n",
    )
    return
}
