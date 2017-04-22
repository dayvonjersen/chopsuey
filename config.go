package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type clientConfig struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Ssl      bool     `json:"ssl"`
	Nick     string   `json:"nick"`
	Autojoin []string `json:"autojoin"`
}

func getClientConfig() *clientConfig {
	f, err := os.Open("config.json")
	checkErr(err)
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	checkErr(err)

	var cfg *clientConfig
	json.Unmarshal(b, &cfg)

	return cfg
}
