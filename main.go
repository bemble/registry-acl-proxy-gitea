package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type GiteaUser struct {
	ID        int       `json:"id"`
	Login     string    `json:"login"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatar_url"`
	Language  string    `json:"language"`
	IsAdmin   bool      `json:"is_admin"`
	LastLogin time.Time `json:"last_login"`
	Created   time.Time `json:"created"`
	Username  string    `json:"username"`
}

var allowAnonymous = getEnv("ALLOW_ANONYMOUS_READ", "0") == "1"
var debugMode = getEnv("DEBUG", "0") == "1"
var giteaHost = getEnv("GITEA_HOST", "")
var readOnlyUsers = strings.Split(getEnv("READ_ONLY_USERNAMES", ""), ",")
var realm = strings.Split(getEnv("REALM", "Registry authentication"), ",")

func debug(v string) {
	if debugMode {
		log.Println(v)
	}
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		originalMethod := r.Header.Get("X-Original-Method")
		originalURI := r.Header.Get("X-Original-Uri")
		isOriginalMethodRead := contains([]string{"GET", "HEAD"}, originalMethod)
		authProvided := r.Header.Get("Authorization") != ""

		debug(fmt.Sprintf("[%s] %s, auth: %v", originalMethod, originalURI, authProvided))

		responseCode := http.StatusUnauthorized
		if allowAnonymous && isOriginalMethodRead {
			responseCode = http.StatusOK
		}

		if responseCode != http.StatusOK && originalURI == "/v2/" && isOriginalMethodRead {
			w.Header().Add("WWW-Authenticate", fmt.Sprintf("Basic realm=\"%s\", charset=\"UTF-8\"", realm))
		}

		if responseCode != http.StatusOK && authProvided && giteaHost != "" {
			req, err := http.NewRequest("GET", giteaHost+"/api/v1/user", nil)
			if err != nil {
				log.Fatalln("Could not create request")
			}
			client := &http.Client{}
			req.Header.Set("Authorization", r.Header.Get("Authorization"))
			resp, err := client.Do(req)
			if err != nil {
				log.Fatalln(err)
			}

			responseCode = resp.StatusCode
			if responseCode == 200 && !isOriginalMethodRead {
				var user GiteaUser
				json.NewDecoder(resp.Body).Decode(&user)
				// Handle read only users
				if contains(readOnlyUsers, user.Username) {
					responseCode = http.StatusUnauthorized
				} else {
					// check repo name
					matched, _ := regexp.MatchString(fmt.Sprintf("^/v2/%s/", user.Username), originalURI)
					if !matched {
						responseCode = http.StatusUnauthorized
					}
				}
			}
		}
		debug(fmt.Sprintf("=> %v", responseCode))
		w.WriteHeader(responseCode)
	})

	port := "8787"
	debug(fmt.Sprintf("Gitea host: %s", giteaHost))
	debug(fmt.Sprintf("Allow anonymous: %v", allowAnonymous))
	debug(fmt.Sprintf("Read only usernames: %v", readOnlyUsers))
	log.Println("Starting proxy on 0.0.0.0:" + port)
	http.ListenAndServe(":"+port, nil)
}

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}
