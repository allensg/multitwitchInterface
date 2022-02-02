package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"

	"github.com/gorilla/pat"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/twitch"
)

func main() {
	clientId := os.Getenv("MULTITWITCH_CLIENT_ID")
	secret := os.Getenv("MULTITWITCH_CLIENT_SECRET")
	redirect := os.Getenv("MULTITWITCH_CLIENT_REDIRECT")

	goth.UseProviders(
		twitch.New(clientId, secret, redirect),
	)

	m := make(map[string]string)
	m["twitch"] = "Twitch"

	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	providerIndex := &ProviderIndex{Providers: keys, ProvidersMap: m}

	// userRetrieved := goth.User{}

	p := pat.New()
	p.Get("/auth/{provider}/callback", func(res http.ResponseWriter, req *http.Request) {

		user, err := gothic.CompleteUserAuth(res, req)
		if err != nil {
			fmt.Fprintln(res, err)
			return
		}
		// userRetrieved = user
		t, _ := template.New("foo").Parse(userTemplate)
		t.Execute(res, user)
	})

	p.Get("/logout/{provider}", func(res http.ResponseWriter, req *http.Request) {
		gothic.Logout(res, req)
		res.Header().Set("Location", "/")
		res.WriteHeader(http.StatusTemporaryRedirect)
	})

	p.Get("/auth/{provider}", func(res http.ResponseWriter, req *http.Request) {
		// try to get the user without re-authenticating
		if gothUser, err := gothic.CompleteUserAuth(res, req); err == nil {
			t, _ := template.New("foo").Parse(userTemplate)
			t.Execute(res, gothUser)
		} else {
			gothic.BeginAuthHandler(res, req)
		}
	})

	p.Get("/", func(res http.ResponseWriter, req *http.Request) {
		t, _ := template.New("foo").Parse(indexTemplate)
		t.Execute(res, providerIndex)

	})

	p.Get("/display", func(res http.ResponseWriter, req *http.Request) {
		following, _, _ := getUsersFollowing(clientId, secret)
		fmt.Println("=====================================")
		for _, k := range following.Data {
			fmt.Println(k.fromName)
		}
		fmt.Println("=====================================")
		t, _ := template.New("foo").Parse(followingTemplate)
		t.Execute(res, following)
	})

	log.Println("listening on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", p))
}

func getUsersFollowing(clientId string, secret string) (following TVDBTokenResponse, authToken string, err error) {
	// POST to tvdb
	tokenRequest := &TVDBTokenRequest{
		Authorization: "Bearer " + secret,
		ClientId:      clientId,
	}

	payload, err := json.Marshal(&tokenRequest)
	if err != nil {
		return TVDBTokenResponse{}, "payload marshall", err
	}

	response, err := http.Post("https://api.twitch.tv/helix/users/follows?from_id=126178180", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return TVDBTokenResponse{}, "post error", err
	}

	if response.StatusCode != 200 {
		return TVDBTokenResponse{}, "not 200", nil
	}

	decoder := json.NewDecoder(response.Body)
	decodedResponse := &TVDBTokenResponse{}
	err = decoder.Decode(&decodedResponse)
	if err != nil {
		return TVDBTokenResponse{}, "decode error", err
	}
	return *decodedResponse, "", nil
}

type ProviderIndex struct {
	Providers    []string
	ProvidersMap map[string]string
}

// var indexTemplate = `{{range $key,$value:=.Providers}}
//     <p><a href="/auth/{{$value}}">Log in with {{index $.ProvidersMap $value}}</a></p>
// {{end}}`

type TVDBTokenRequest struct {
	Authorization string `json:"Authorization"`
	ClientId      string `json:"Client-Id"`
}

type FollowUser struct {
	fromId     string `json:"from_id"`
	fromLogin  string `json:"from_login"`
	fromName   string `json:"from_name"`
	toId       string `json:"to_id"`
	toLogin    string `json:"to_login"`
	toName     string `json:"to_name"`
	followedAt string `json:"followed_at"`
}

type TVDBTokenResponse struct {
	Total string       `json:"total"`
	Data  []FollowUser `json:"data"`
}

var followingTemplate = `{{range $key,$value:=.Data}}
     <p>Add {{$value.fromName}}</p>{{end}}`

var indexTemplate = `<p><a href="/auth/twitch">Log in with Twitch</a></p>`

var userTemplate = `
<p><a href="/logout/{{.Provider}}">logout</a></p>
<p>Name: {{.Name}} [{{.LastName}}, {{.FirstName}}]</p>
<p>Email: {{.Email}}</p>
<p>NickName: {{.NickName}}</p>
<p>Location: {{.Location}}</p>
<p>AvatarURL: {{.AvatarURL}} <img src="{{.AvatarURL}}"></p>
<p>Description: {{.Description}}</p>
<p>UserID: {{.UserID}}</p>
<p>AccessToken: {{.AccessToken}}</p>
<p>ExpiresAt: {{.ExpiresAt}}</p>
<p>RefreshToken: {{.RefreshToken}}</p>
<p><a href="/display">display</a></p>
`
