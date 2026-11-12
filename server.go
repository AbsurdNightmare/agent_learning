package main

import (
	"encoding/json"
	"fmt"
	"learning/agent"
	"learning/model"
	"net/http"
)

func handleChat(w http.ResponseWriter, r *http.Request) {
	setupSSE(w, r)
	if r.Method == http.MethodOptions {
		return
	}

	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	a := agent.NewAgent(llmClient, agentTools)
	eventChan := make(chan model.AgentEvent)

	go func() {
		_ = a.Run(r.Context(), req.Query, eventChan)
		close(eventChan)
	}()

	streamEvents(w, eventChan)
}

func handleResume(w http.ResponseWriter, r *http.Request) {
	setupSSE(w, r)
	if r.Method == http.MethodOptions {
		return
	}

	var req struct {
		Decision string `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	a := agent.NewAgent(llmClient, agentTools)
	eventChan := make(chan model.AgentEvent)

	go func() {
		_ = a.Resume(r.Context(), req.Decision, eventChan)
		close(eventChan)
	}()

	streamEvents(w, eventChan)
}

func setupSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
}

func streamEvents(w http.ResponseWriter, ch <-chan model.AgentEvent) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}
	for evt := range ch {
		data, _ := json.Marshal(evt)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}
