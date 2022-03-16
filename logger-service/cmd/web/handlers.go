package main

import (
	"bytes"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"log"
	"log-service/data"
	"net/http"
	"time"
)

const authServiceURL = "http://authentication-service/authenticate"

type JSONPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) WriteLog(w http.ResponseWriter, r *http.Request) {
	// read json into var
	var requestPayload JSONPayload
	_ = readJSON(w, r, &requestPayload)

	// create the data we'll save to mongo in
	// the correct format
	entry := data.LogEntry{
		Name:      requestPayload.Name,
		Data:      requestPayload.Data,
		CreatedAt: time.Now(),
	}

	// insert the data, ignoring returned id
	_, err := app.Models.LogEntry.Insert(entry)
	if err != nil {
		log.Println(err)
		_ = errorJSON(w, err, http.StatusBadRequest)
		return
	}

	// create the response we'll send back as JSON
	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	resp.Message = "logged"
	_ = writeJSON(w, http.StatusAccepted, resp)
}

// Logout logs the user out and redirects them to the login page
func (app *Config) Logout(w http.ResponseWriter, r *http.Request) {
	_ = app.Session.Destroy(r.Context())
	_ = app.Session.RenewToken(r.Context())

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// LoginPage displays the login page
func (app *Config) LoginPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "login.page.gohtml", nil)
}

// LoginPagePost handles user login. Note that it calls the authentication microservice
func (app *Config) LoginPagePost(w http.ResponseWriter, r *http.Request) {
	// it's always good to regenerate the session on login/logout
	_ = app.Session.RenewToken(r.Context())

	// parse the posted form data
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")

	// create a variable we'll post to the auth service as JSON
	var requestPayload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	requestPayload.Email = email
	requestPayload.Password = password

	// create json we'll send to the authentication-service
	jsonData, _ := json.MarshalIndent(requestPayload, "", "\t")

	// call the authentication-service
	request, err := http.NewRequest("POST", authServiceURL, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		_ = errorJSON(w, err, http.StatusBadRequest)
		return
	}
	defer response.Body.Close()

	// make sure we get back the right status code
	if response.StatusCode == http.StatusUnauthorized {
		log.Println("wrong status code")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	} else if response.StatusCode != http.StatusAccepted {
		log.Println("did not get status accepted")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var user struct {
		ID        int       `json:"id"`
		Email     string    `json:"email"`
		FirstName string    `json:"first_name,omitempty"`
		LastName  string    `json:"last_name,omitempty"`
		Password  string    `json:"-"`
		Active    int       `json:"active"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	_ = readJSON(w, r, &user)

	// set up session & log user in
	app.Session.Put(r.Context(), "userID", user.ID)
	http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
}

// Dashboard displays the dashboard page
func (app *Config) Dashboard(w http.ResponseWriter, r *http.Request) {
	// get the list of all log entries from mongo
	logs, err := app.Models.LogEntry.All()
	if err != nil {
		log.Println("Error getting all log entries")
		app.clientError(w, http.StatusBadRequest)
	}

	templateData := make(map[string]interface{})
	templateData["logs"] = logs

	app.render(w, r, "dashboard.page.gohtml", &TemplateData{
		Data: templateData,
	})
}

func (app *Config) DisplayOne(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	entry, err := app.Models.LogEntry.GetOne(id)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	templateData := make(map[string]interface{})
	templateData["entry"] = entry

	app.render(w, r, "entry.page.gohtml", &TemplateData{
		Data: templateData,
	})
}