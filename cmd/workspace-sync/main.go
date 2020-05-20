package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	wssync "github.com/tmitchel/workspace-sync"
)

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath("./")
	if err := viper.ReadInConfig(); err != nil {
		logrus.Fatal("Fatal error config file: %s \n", err)
	}

	config := wssync.Config{
		IceURL:      viper.GetString("iceURL"),
		ChannelName: viper.GetString("channel"),
		Addr:        viper.GetString("port"),
		Ignore:      viper.GetStringSlice("ignore"),
	}

	if viper.GetBool("local") {
		config.WatchDir = viper.GetString("directories")
		l, err := wssync.NewLocal(config)
		if err != nil {
			logrus.Fatal(err)
		}
		go l.Watch()
	} else {
		_, err := wssync.NewRemote(config)
		if err != nil {
			logrus.Fatal(err)
		}
	}

	// Block forever
	select {}
}
