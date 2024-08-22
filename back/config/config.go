package config

import (
	"evanBlog/entity/configobj"
	"fmt"
	"github.com/spf13/viper"
)

type Configuration struct {
	Mysql configobj.Mysql
}

var Config *Configuration

func init() {
	Config = &Configuration{}
	viper.SetConfigFile("./config.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %v \n", err))
	}
	err = viper.Unmarshal(&Config)
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %v \n", err))
	}
}
