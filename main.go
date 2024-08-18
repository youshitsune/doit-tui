package main

import (
	"fmt"
	"os"
	"path"

	"github.com/go-zoox/fetch"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var conf = koanf.New(".")

type config struct {
	url      string
	username string
	password string
}

func load_config() config {
	config_path := path.Join(os.ExpandEnv("$XDG_CONFIG_HOME"), "/doit/config.yaml")
	if err := conf.Load(file.Provider(config_path), yaml.Parser()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	cfg := config{}
	cfg.url = conf.String("protocol") + "://" + conf.String("url") + ":" + conf.String("port")
	cfg.password = conf.String("password")
	cfg.username = conf.String("username")
	return cfg
}

func list(cfg config) string {
	res, err := fetch.Post(cfg.url+"/list", &fetch.Config{
		Query: map[string]string{
			"user":     cfg.username,
			"password": cfg.password,
		},
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return string(res.Body)
}

func add(cfg config, task string) bool {
	res, err := fetch.Post(cfg.url+"/new", &fetch.Config{
		Query: map[string]string{
			"user":     cfg.username,
			"password": cfg.password,
			"task":     task,
		},
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if res.StatusCode() == 200 {
		return true
	} else {
		return false
	}
}

func main() {
	cfg := load_config()
	fmt.Println(list(cfg))
}
