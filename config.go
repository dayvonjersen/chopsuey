package main

import (
	"encoding/json"
	"io"
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
	QuitMessage     string              `json:"quitmessage"`
}

func defaultClientConfig() *clientConfig {
	return &clientConfig{
		AutoConnect: []*connectionConfig{},
		TimeFormat:  "15:04",
		Version:     "chopsuey IRC " + VERSION_STRING + " github.com/generaltso/chopsuey",
		QuitMessage: "|･ω･｀)",
	}
}

func getClientConfig() (*clientConfig, error) {
	cfg := defaultClientConfig()

	f, err := os.Open("config.json")
	defer f.Close()
	if err != nil {
		return cfg, err
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return cfg, err
	}

	err = json.Unmarshal(b, &cfg)

	return cfg, err
}

func writeClientConfig() error {
	f, err := os.Create("config.json")
	if err == nil {
		b, err := json.MarshalIndent(clientCfg, "", "    ")
		if err != nil {
			return err
		}
		io.WriteString(f, string(b))
		f.Close()
	}
	return err
}
