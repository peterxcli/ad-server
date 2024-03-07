package bootstrap

type Server struct {
	Port     uint   `env:"PORT" envDefault:"8080"`
	TimeZone string `env:"TIMEZONE" envDefault:"Asia/Taipei"`
}
