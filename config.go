package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type clientConfig struct {
	Host          string   `json:"host"`
	Port          int      `json:"port"`
	Ssl           bool     `json:"ssl"`
	Nick          string   `json:"nick"`
	Autojoin      []string `json:"autojoin"`
	HideJoinParts bool     `json:"hidejoinparts"`
}

func (cfg *clientConfig) ServerString() string {
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
