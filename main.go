package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"

	"github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
)

const (
	version = "0.1"
)

var token = os.Getenv("VAULT_TOKEN")
var vault_addr = os.Getenv("VAULT_ADDR")
var client *api.Client

func main() {
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.DebugLevel)
	log.Infof("Starting version %s, listening on port 8090", version)
	var err error

	config := &api.Config{
		Address: vault_addr,
	}
	client, err = api.NewClient(config)
	if err != nil {
		fmt.Println(err)
		return
	}
	client.SetToken(token)

	rtr := mux.NewRouter()
	rtr.HandleFunc("/", Index)
	rtr.HandleFunc("/{endpoint:.*}", Index)
	srv := &http.Server{
		Handler: rtr,
		Addr:    "0.0.0.0:8090",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
func Index(w http.ResponseWriter, r *http.Request) {
	var endpoint string
	var ok bool
	vars := mux.Vars(r)
	log.Debugf("Vars=%#v", vars)

	if endpoint, ok = vars["endpoint"]; ok {
		log.Debugf("Found endpoint %s", endpoint)
		endpoint = "sys/raw/" + endpoint
	} else {
		endpoint = "sys/raw"
	}

	var index string
	if endpoint[len(endpoint)-1:] == "/" || endpoint == "sys/raw" {
		log.Debugf("Getting list of %s", endpoint)
		index = getList(endpoint)
	} else {
		log.Debugf("Getting read of %s", endpoint)
		w.Header().Set("Content-Type", "application/json")
		index = getRead(endpoint)
	}
	fmt.Fprintf(w, index)
}
func getRead(endpoint string) string {
	resp, err := client.Logical().Read(endpoint)
	if err != nil {
		log.Debugf("ERROR: %s", err)
		return fmt.Sprintf("%s", err)
	}
	//log.Debugf("READ raw %s", prettyPrint(resp))

	if resp.Data == nil {
		errStr := fmt.Sprintf("READ failed to read value for %s", endpoint)
		log.Fatal(errStr)
		return errStr
	}
	log.Debugf("READ Data: %s\n", prettyPrint(fmt.Sprintf("%s", resp.Data["value"])))

	return fmt.Sprintf("%s", resp.Data["value"])

}
func getList(endpoint string) string {
	resp, err := client.Logical().List(endpoint)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	log.Debugf("raw %#v", resp)
	log.Debugf("Data: %v\n", prettyPrint(fmt.Sprintf("%s", resp.Data["value"])))

	m, ok := resp.Data["keys"].([]interface{})
	if !ok {
		log.Fatal(fmt.Sprintf("failed to read value for key: sys/raw"))
	}
	var buff bytes.Buffer
	for _, v := range m {
		log.Debugf("V=%#v", v)

		buff.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a><br>\n", v, v))

	}
	return buff.String()
}

func prettyPrint(body string) string {
	var prettyJSON bytes.Buffer
	error := json.Indent(&prettyJSON, []byte(body), "", "  ")
	if error != nil {
		log.Println("JSON parse error: ", error)
		return fmt.Sprintf("%s", error)
	}
	return prettyJSON.String()
}
