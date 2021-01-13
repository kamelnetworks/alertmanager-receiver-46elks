// Copyright (C) 2021  Kamel Networks
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

var (
	m = sync.Mutex{}
)

type Alert struct {
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations struct {
		Description string `json:"description"`
		Summary     string `json:"summary"`
	} `json:"annotations"`
}

type Callback struct {
	Alerts []Alert `json:"alerts"`
}

func handle(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()
	var cb Callback
	if err := json.Unmarshal(b, &cb); err != nil {
		log.Printf("Error handling JSON: %+v", err)
		http.Error(w, "Handling error", http.StatusInternalServerError)
		return
	}
	m.Lock()
	defer m.Unlock()

	for _, alert := range cb.Alerts {
		handleAlert(w, r, &alert)
	}
}

func handleAlert(w http.ResponseWriter, r *http.Request, alert *Alert) {
	id := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%+v", alert.Labels))))

	df, err := ioutil.ReadFile("active-alerts.yaml")
	if os.IsNotExist(err) {
		df = []byte("")
	} else if err != nil {
		log.Printf("Error reading active alerts YAML: %+v", err)
		http.Error(w, "Handling error", http.StatusInternalServerError)
		return
	}
	idl := []string{}
	if err := yaml.Unmarshal(df, &idl); err != nil {
		log.Printf("Error parsing active alerts YAML: %+v", err)
		http.Error(w, "Handling error", http.StatusInternalServerError)
		return
	}

	found := false
	for _, v := range idl {
		if v == id {
			found = true
			break
		}
	}
	if found {
		// De-dup!
		return
	}

	idl = append(idl, id)
	yidl, err := yaml.Marshal(idl)
	if err != nil {
		log.Printf("Error creating new active alert YAML: %v", err)
		http.Error(w, "Handling error", http.StatusInternalServerError)
		return
	}

	to := "+" + r.URL.Path[1:]
	if !strings.HasPrefix(to, "+467") {
		log.Printf("Number has to start with +467.., to is: %q", to)
		http.Error(w, "Invalid to number", http.StatusForbidden)
		return
	}

	log.Printf("New alert! To: %q, ID: %q, Data: %+v", to, id, alert)
	message := fmt.Sprintf("%s\n%s\n\n", alert.Annotations.Summary, alert.Annotations.Description)
	for k, v := range alert.Labels {
		message += fmt.Sprintf("%s: %s\n", k, v)
	}
	log.Printf("Message: %s", message)

	data := url.Values{
		"from":    {"KamelNet"},
		"to":      {to},
		"message": {message},
	}

	req, err := http.NewRequest("POST", "https://api.46elks.com/a1/sms", bytes.NewBufferString(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	req.SetBasicAuth(apiUsername, apiPassword)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("SMS sending failure: %+v", err)
		http.Error(w, "SMS error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("SMS sending read failure: %+v", err)
		http.Error(w, "SMS error", http.StatusInternalServerError)
		return
	}

	if err := ioutil.WriteFile("active-alerts.yaml", yidl, 0644); err != nil {
		log.Printf("Error write new active alert YAML: %v", err)
		http.Error(w, "Handling error", http.StatusInternalServerError)
		return
	}
}

func main() {
	log.Printf("Running")
	// Ask soundgoof why the port 1025 was chosen
	log.Fatal(http.ListenAndServe(":1025", http.HandlerFunc(handle)))
}
