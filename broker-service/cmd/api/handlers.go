package main

import (
	"broker/event"
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type Payload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	err := app.pushToQueue("broker_hit", r.RemoteAddr)
	if err != nil {
		log.Println(err)
	}
	var payload struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	payload.Message = "Received request"

	out, _ := json.MarshalIndent(payload, "", "\t")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(out)
}

func (app *Config) BrokerAuth(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	_ = readJSON(w, r, &requestPayload)

	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	request, err := http.NewRequest("POST", "http://authentication-service/authenticate", bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		_ = errorJSON(w, err, http.StatusBadRequest)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		_ = errorJSON(w, errors.New("error calling auth service"), http.StatusBadRequest)
		return
	}

	// variable we'll read the response.Body into
	var jsonFromService jsonResponse

	// decode the json we get from the auth service into our variable
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		_ = errorJSON(w, err, http.StatusBadRequest)
		return
	}

	// send json back to our end user
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Authenticated!"
	payload.Data = jsonFromService.Data

	out, _ := json.MarshalIndent(payload, "", "\t")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write(out)
}

func (app *Config) pushToQueue(name, msg string) error {
	emitter, err := event.NewEventEmitter(app.Rabbit)
	if err != nil {
		log.Println(err)
		return err
	}

	payload := Payload{
		Name: name,
		Data: msg,
	}

	j, _ := json.MarshalIndent(&payload, "", "    ")
	err = emitter.Push(string(j), "log.INFO")
	if err != nil {
		return err
	}
	return nil
}