package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type connectionConfig struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Ssl      bool     `json:"ssl"`
	Nick     string   `json:"nick"`
	AutoJoin []string `json:"autojoin"`
}

type clientConfig struct {
	AutoConnect     []*connectionConfig `json:"autoconnect"`
	HideJoinParts   bool                `json:"hidejoinparts"`
	ChatLogsEnabled bool                `json:"chatlogs"`
	TimeFormat      string              `json:"timeformat"`
	Version         string              `json:"version"`
}

func (cfg *connectionConfig) ServerString() string {
	if cfg.Ssl {
		return fmt.Sprintf("%s:+%d", cfg.Host, cfg.Port)
	}
	return fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
}

func getClientConfig() (*clientConfig, error) {
	f, err := os.Open("config.json")
	checkErr(err)
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	checkErr(err)

	var cfg *clientConfig
	err = json.Unmarshal(b, &cfg)

	return cfg, err
}
