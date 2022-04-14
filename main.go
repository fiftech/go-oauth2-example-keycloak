package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

func main() {
	clientPort := 9094
	oauth2cli := oauth2.Config{
		ClientID: "test-id",
		Scopes: []string{
			"openid",
		},
		RedirectURL: fmt.Sprintf("http://localhost:%d/callback", clientPort),
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://localhost:8080/realms/master/protocol/openid-connect/auth",
			TokenURL: "http://localhost:8080/realms/master/protocol/openid-connect/token",
		},
	}
	var clientState = randomStr(8)
	var codeVerifier string
	var codeChallenge string

	// Authorization Code Grant Flow
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		codeVerifier = genCodeVerifier()
		codeChallenge = genCodeChallengeS256(codeVerifier)
		u := oauth2cli.AuthCodeURL(clientState,
			oauth2.SetAuthURLParam("code_challenge", codeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
		http.Redirect(w, r, u, http.StatusFound)
	})

	// Authorization Code Grant Callback
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		state := r.Form.Get("state")
		if state != clientState {
			http.Error(w, "State invalid", http.StatusBadRequest)
			return
		}
		code := r.Form.Get("code")
		if code == "" {
			http.Error(w, "Code not found", http.StatusBadRequest)
			return
		}
		token, err := oauth2cli.Exchange(context.Background(), code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, token)
	})

	// Running Client
	log.Println(fmt.Sprintf("Client is running at %d port.", clientPort))
	log.Println(fmt.Sprintf("Please open http://localhost:%d/", clientPort))
	log.Println("===== IMPORTANT!=================================================================================================")
	log.Println("Remember imports test-id.json file as client at http://localhost:8080/admin/master/console/#/create/client/master")
	log.Println("=================================================================================================================")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", clientPort), nil))
}

// Util functions
var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomStr(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func base64Encode(data []byte) string {
	s := base64.URLEncoding.EncodeToString(data)
	return strings.ReplaceAll(s, "=", "")
}

func genCodeVerifier() string {
	str := randomStr(32)
	return base64Encode([]byte(str))
}

func genCodeChallengeS256(codeVerifier string) string {
	s256 := sha256.Sum256([]byte(codeVerifier))
	return base64Encode(s256[:])
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	_ = e.Encode(data)
}
