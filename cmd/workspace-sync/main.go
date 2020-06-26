package main

import (
	"flag"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	wssync "github.com/tmitchel/workspace-sync"
)

func main() {
	endpoint := flag.String("endpoint", "", "[remote, local] which end is this?")
	flag.Parse()

	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath("./")
	if err := viper.ReadInConfig(); err != nil {
		logrus.Fatal("Fatal error config file: %s \n", err)
	}

	// build config from config.json
	config := wssync.Config{
		IceURL: viper.GetString("iceURL"),
		Addr:   viper.GetString("port"),
		Ignore: viper.GetStringSlice("ignore"),
	}

	if *endpoint == "local" {
		config.WatchDir = viper.GetString("directories")
		l, err := wssync.NewLocal(config)
		if err != nil {
			logrus.Fatal(err)
		}
		go l.Watch()
	} else if *endpoint == "remote" {
		_, err := wssync.NewRemote(config)
		if err != nil {
			logrus.Fatal(err)
		}
	} else {
		logrus.Fatalf("endpoint must be 'local' or 'remote' not %s", *endpoint)
	}

	// Block forever
	select {}
}
