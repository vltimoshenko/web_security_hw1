package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"

	"web_security_hw1/internal/proxy"
	"web_security_hw1/pkg/connector"

	"github.com/spf13/viper"
)

func main() {
	var pemPath string
	flag.StringVar(&pemPath, "pem", "server.pem", "path to pem file")
	var keyPath string
	flag.StringVar(&keyPath, "key", "server.key", "path to key file")
	var cfgPath string
	flag.StringVar(&cfgPath, "c", "./configs/config.yaml", "config path")

	flag.Parse()

	viper.SetConfigFile(cfgPath)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Fatal error config file: %s", err)
	}

	proto := viper.GetString("main.schema")
	if proto != "http" && proto != "https" {
		log.Fatal("Protocol must be either http or https")
	}

	fmt.Println("Starting http proxy")

	c := connector.Connector{
		DBHostname:     viper.GetString("db.hostname") + ":" + viper.GetString("db.port"),
		DBName:         viper.GetString("db.name"),
		DBUser:         viper.GetString("db.user"),
		DBUserPassword: viper.GetString("db.password"),
		DBMaxConns:     viper.GetInt("db.max_conns"),
	}

	p, err := proxy.InitProxy(&c, ":"+strconv.Itoa(viper.GetInt("main.port")), proto)
	if err != nil {
		log.Fatalf("Fatal error proxy initialization: %s", err)
	}

	if proto == "http" {
		log.Fatal(p.Server.ListenAndServe())
	} else {
		log.Fatal(p.Server.ListenAndServeTLS(pemPath, keyPath))
	}
}
