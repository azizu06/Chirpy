package main

import (
	"encoding/json"
	"net/http"
)

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errResp struct {
		Error string `json:"error"`
	}
	resp := errResp{
		Error: msg,
	}
	respondWithJSON(w, code, resp)
}
