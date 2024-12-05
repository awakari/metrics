package config

import (
    "github.com/stretchr/testify/assert"
    "os"
    "testing"
)

func TestConfig(t *testing.T) {
    os.Setenv("LOG_LEVEL", "4")
    os.Setenv("API_PORT", "56789")
    os.Setenv("LIMITS_DEFAULT_GROUPS", "group0,group1,group2")
    os.Setenv("API_HTTP_COOKIE_DOMAIN", "domain")
    os.Setenv("API_HTTP_COOKIE_SECRET", "secret")
    cfg, err := NewConfigFromEnv()
    assert.Nil(t, err)
    assert.Equal(t, uint16(56789), cfg.Api.Port)
    assert.Equal(t, 4, cfg.Log.Level)
    assert.Equal(t, []string{"group0", "group1", "group2"}, cfg.Limits.Default.Groups)
}
