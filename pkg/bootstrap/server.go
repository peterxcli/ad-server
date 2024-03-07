package bootstrap

type Server struct {
	Port     uint   `env:"PORT" envDefault:"8000"`
	TimeZone string `env:"TIMEZONE" envDefault:"Asia/Taipei"`
}
