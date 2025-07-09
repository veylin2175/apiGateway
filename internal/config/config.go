package config

import (
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
	"time"
)

// Config анмаршлит данные из конфига в структуры
type Config struct {
	Env        string     `yaml:"env" env-default:"local"`
	HTTPServer HTTPServer `yaml:"http_server"`
	Kafka      Kafka      `yaml:"kafka"`
	Blockchain Blockchain `yaml:"blockchain"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8062"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

type Kafka struct {
	Brokers         []string `yaml:"brokers" env-required:"true"`
	Topic           string   `yaml:"topic" env-required:"true"`
	GroupID         string   `yaml:"group_id" env-default:"voting-service"`
	AutoOffsetReset string   `yaml:"auto_offset_reset" env-default:"earliest"`
	MaxPollRecords  int      `yaml:"max_poll_records" env-default:"1"`
}

type Blockchain struct {
	RpcUrl          string `yaml:"rpc_url" env-required:"true"`
	WalletAddress   string `yaml:"wallet_address" env-required:"true"`
	ContractAddress string `yaml:"contract_address" env-required:"true"`
	PrivateKey      string `yaml:"private_key" env-required:"true"`
}

// MustLoad выгружает данные с конфига по пути до файла
func MustLoad() *Config {
	path := fetchConfigPath()

	if path == "" {
		panic("config path is empty")
	}

	return MustLoadByPath(path)
}

func MustLoadByPath(configPath string) *Config {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("cannot read config" + err.Error())
	}

	return &cfg
}

// fetchConfigPath извлекает путь конфигурации из флага командной строки или переменной среды.
// Приоритет: flag > env > default.
// Дефолтное значение — пустая строка.
func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "config file path")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
