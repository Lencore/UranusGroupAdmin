package dto

type (
	Config struct {
		LogLevel string `yaml:"log_level" default:"info"`
		Bot      struct {
			Token string `yaml:"token"`
			Debug bool   `yaml:"debug"`
		}
		DB struct {
			Name     string
			User     string `default:"admin"`
			Password string
			Host     string `default:"localhost"`
			Port     string `default:"5432"`
			Debug    bool
			SSLMode  string `yaml:"sslmode"`
		}
		Redis struct {
			Addr      string `yaml:"addr"`
			NameSpace string `yaml:"name_space"`
		}
	}
)
