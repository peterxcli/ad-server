package bootstrap

type JWTEnv struct {
	AccessTokenSecret string `env:"ACCESS_SECRET" envDefault:"access-secret"`
	AccessTokenExpiry int64  `env:"ACCESS_EXPIRY" envDefault:"3600"` // in seconds

	RefreshTokenSecret string `env:"REFRESH_SECRET" envDefault:"refresh-secret"`
	RefreshTokenExpiry int64  `env:"REFRESH_EXPIRY" envDefault:"2592000"` // in seconds
}
