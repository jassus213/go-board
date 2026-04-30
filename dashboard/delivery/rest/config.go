package rest

const defaultAPIPrefix = "/api/v1"

// Config describes transport-level HTTP settings for REST/WS delivery.
type Config struct {
	EnableWebSocket      bool
	CORSAllowedOrigins   []string
	CORSAllowCredentials bool
	APIPrefix            string
}

func (c Config) normalized() Config {
	if c.APIPrefix == "" {
		c.APIPrefix = defaultAPIPrefix
	}
	return c
}
