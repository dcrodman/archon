package main

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed "templates/index.html"
var uiTemplate string

// startManageServer starts the HTTP server for the UI.
func startManageServer(serverAddr string, managePort int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/ui", uiHandler)

	addr := fmt.Sprintf("%s:%d", serverAddr, managePort)
	fmt.Println("manage API is listening on", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Println(err)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		sessionName := r.FormValue("session")
		if sessionName != "" {
			for _, p := range packetQueues[sessionName] {
				if err := writePacketToFile(bufio.NewWriter(w), &p); err != nil {
					fmt.Printf("unable to write packet to %s: %s\n", sessionName, err)
				}
			}
		} else {
			var sessionNames []string
			for k := range packetQueues {
				sessionNames = append(sessionNames, k)
			}
			names, _ := json.Marshal(sessionNames)
			if _, err := w.Write(names); err != nil {
				fmt.Println(err)
			}
		}
	case "DELETE":
		packetQueues = make(map[string][]Packet)
	default:
		fmt.Fprintf(w, "Sorry, only GET and DELETE methods are supported.")
	}
}

func uiHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.New("index").Parse(uiTemplate)
	var sessionNames []string
	for k := range packetQueues {
		sessionNames = append(sessionNames, k)
	}
	err := tmpl.Execute(w, sessionNames)
	if err != nil {
		fmt.Printf("unable to execute template %v", err)
	}
}
