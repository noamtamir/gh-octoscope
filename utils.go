package main

func checkErr(err error) {
	if err != nil {
		logger.Fatal().Stack().Err(err).Msg("")
	}
}

// func formatJson(obj interface{}) string {
// 	jsonObj, err := json.Marshal(obj)
// 	checkErr(err)
// 	r := bytes.NewReader(jsonObj)
// 	buf := new(bytes.Buffer)
// 	err = jsonpretty.Format(buf, r, "  ", true)
// 	checkErr(err)
// 	return buf.String()
// }
