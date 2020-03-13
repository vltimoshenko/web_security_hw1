package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	. "web_security_hw1/internal/repeater"
	"web_security_hw1/pkg/connector"

	"github.com/spf13/viper"
)

func main() {
	var cfgPath string

	flag.StringVar(&cfgPath, "c", "./configs/config.yaml", "config path")
	flag.Parse()

	viper.SetConfigFile(cfgPath)

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Fatal error config file: %s", err)
	}

	c := connector.Connector{
		DBHostname:     viper.GetString("db.hostname") + ":" + viper.GetString("db.port"),
		DBName:         viper.GetString("db.name"),
		DBUser:         viper.GetString("db.user"),
		DBUserPassword: viper.GetString("db.password"),
		DBMaxConns:     viper.GetInt("db.max_conns"),
	}

	fmt.Println("Starting http repeater")
	repeater, err := InitRepeater(&c, ":"+strconv.Itoa(viper.GetInt("repeater.port")), viper.GetString("main.schema"))

	if err != nil {
		log.Fatalf("Failed to start db %s", err.Error())

	}
	defer repeater.DB.Close()

	log.Fatal(repeater.Server.ListenAndServe())
}
