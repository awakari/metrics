package config

import (
	"github.com/kelseyhightower/envconfig"
	"time"
)

type Config struct {
	Admin struct {
	}
	Api struct {
		Port   uint16 `envconfig:"API_PORT" default:"50051" required:"true"`
		Source struct {
			ActivityPub ActivityPubConfig
			Feeds       FeedsConfig
			Sites       SitesConfig
			Telegram    TelegramConfig
		}
		Interests InterestsConfig
		Http      struct {
			Port   uint16 `envconfig:"API_HTTP_PORT" default:"8080"`
			Cookie CookieConfig
		}
		Metrics struct {
			Port uint16 `envconfig:"API_METRICS_PORT" default:"9090" required:"true"`
		}
		Prometheus PrometheusConfig
		Usage      UsageConfig
	}
	Limits LimitsConfig
	Log    struct {
		Level int `envconfig:"LOG_LEVEL" default:"-4" required:"true"`
	}
}

type LimitsConfig struct {
	Default struct {
		Groups []string `envconfig:"LIMITS_DEFAULT_GROUPS" default:"" required:"true"`
		User   struct {
			Publish struct {
				Hourly int64 `envconfig:"LIMITS_DEFAULT_USER_PUBLISH_HOURLY" default:"10" required:"true"`
				Daily  int64 `envconfig:"LIMITS_DEFAULT_USER_PUBLISH_DAILY" default:"100" required:"true"`
			}
		}
	}
	Max struct {
		User struct {
			Publish struct {
				Hourly int64 `envconfig:"LIMITS_MAX_USER_PUBLISH_HOURLY" default:"3600" required:"true"`
				Daily  int64 `envconfig:"LIMITS_MAX_USER_PUBLISH_DAILY" default:"86400" required:"true"`
			}
		}
	}
}

type FeedsConfig struct {
	Uri string `envconfig:"API_SOURCE_FEEDS_URI" default:"source-feeds:50051" required:"true"`
}

type TelegramConfig struct {
	Uri string `envconfig:"API_SOURCE_TELEGRAM_URI" default:"source-telegram:50051" required:"true"`
}

type SitesConfig struct {
	Uri string `envconfig:"API_SOURCE_SITES_URI" default:"source-sites:50051" required:"true"`
}

type ActivityPubConfig struct {
	Uri string `envconfig:"API_SOURCE_ACTIVITYPUB_URI" default:"int-activitypub:50051" required:"true"`
}

type InterestsConfig struct {
	Uri        string `envconfig:"API_INTERESTS_URI" default:"interests-api:50051" required:"true"`
	Connection struct {
		Count struct {
			Init uint32 `envconfig:"API_INTERESTS_CONN_COUNT_INIT" default:"1" required:"true"`
			Max  uint32 `envconfig:"API_INTERESTS_CONN_COUNT_MAX" default:"2" required:"true"`
		}
		IdleTimeout time.Duration `envconfig:"API_INTERESTS_CONN_IDLE_TIMEOUT" default:"15m" required:"true"`
	}
}

type PrometheusConfig struct {
	Uri string `envconfig:"API_PROMETHEUS_URI" default:"http://prometheus-server:80" required:"true"`
}

type CookieConfig struct {
	MaxAge   time.Duration `envconfig:"API_HTTP_COOKIE_MAX_AGE" default:"24h" required:"true"`
	Path     string        `envconfig:"API_HTTP_COOKIE_PATH" default:"/" required:"true"`
	Domain   string        `envconfig:"API_HTTP_COOKIE_DOMAIN" required:"true"`
	Secure   bool          `envconfig:"API_HTTP_COOKIE_SECURE" default:"true" required:"true"`
	HttpOnly bool          `envconfig:"API_HTTP_COOKIE_HTTP_ONLY" default:"true" required:"true"`
	Secret   string        `envconfig:"API_HTTP_COOKIE_SECRET" required:"true"`
}

type UsageConfig struct {
	Uri        string `envconfig:"API_USAGE_URI" default:"usage:50051" required:"true"`
	Connection struct {
		Count struct {
			Init uint32 `envconfig:"API_USAGE_CONN_COUNT_INIT" default:"1" required:"true"`
			Max  uint32 `envconfig:"API_USAGE_CONN_COUNT_MAX" default:"10" required:"true"`
		}
		IdleTimeout time.Duration `envconfig:"API_USAGE_CONN_IDLE_TIMEOUT" default:"15m" required:"true"`
	}
}

func NewConfigFromEnv() (cfg Config, err error) {
	err = envconfig.Process("", &cfg)
	return
}
