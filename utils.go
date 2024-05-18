package main

import (
	"bytes"
	"encoding/json"

	"github.com/cli/go-gh/pkg/jsonpretty"
)

func checkErr(err error) {
	if err != nil {
		logger.Fatal().Stack().Err(err).Msg("")
	}
}

func formatJson(obj interface{}) string {
	jsonObj, err := json.Marshal(obj)
	checkErr(err)
	r := bytes.NewReader(jsonObj)
	buf := new(bytes.Buffer)
	logger.Debug().Msg("")
	err = jsonpretty.Format(buf, r, "  ", true)
	checkErr(err)
	return buf.String()
}
