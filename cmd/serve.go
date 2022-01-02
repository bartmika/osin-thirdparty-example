package cmd

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/openshift/osin"
	"github.com/openshift/osincli"
	"github.com/spf13/cobra"

	"github.com/bartmika/osin-thirdparty-example/utils"
)

var (
	ClientID     string
	ClientSecret string
	AuthorizeURL string
	TokenURL     string
	RedirectURL  string
)

func init() {
	serveCmd.Flags().StringVarP(&ClientID, "client_id", "a", "", "-")
	serveCmd.MarkFlagRequired("client_id")
	serveCmd.Flags().StringVarP(&ClientSecret, "client_secret", "b", "", "-")
	serveCmd.MarkFlagRequired("client_secret")
	serveCmd.Flags().StringVarP(&AuthorizeURL, "authorize_uri", "c", "", "-")
	serveCmd.MarkFlagRequired("authorize_uri")
	serveCmd.Flags().StringVarP(&TokenURL, "token_url", "d", "", "-")
	serveCmd.MarkFlagRequired("token_url")
	serveCmd.Flags().StringVarP(&RedirectURL, "redirect_uri", "e", "", "-")
	serveCmd.MarkFlagRequired("redirect_uri")
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Print the version number",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		runServeCmd()
	},
}

func runServeCmd() {
	// create http muxes
	clienthttp := http.NewServeMux()

	// create client
	cliconfig := &osincli.ClientConfig{
		ClientId:                 ClientID,
		ClientSecret:             ClientSecret,
		AuthorizeUrl:             AuthorizeURL,
		TokenUrl:                 TokenURL,
		RedirectUrl:              RedirectURL,
		SendClientSecretInParams: false,
		Scope:                    "all",
	}
	client, err := osincli.NewClient(cliconfig)
	if err != nil {
		panic(err)
	}

	// create a new request to generate the url
	areq := client.NewAuthorizeRequest(osincli.CODE)

	// CLIENT

	// Home
	clienthttp.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		u := areq.GetAuthorizeUrl()

		w.Write([]byte(fmt.Sprintf("<a href=\"%s\">Login</a>", u.String())))
	})

	// Auth endpoint
	clienthttp.HandleFunc("/appauth", func(w http.ResponseWriter, r *http.Request) {
		// parse a token request
		areqdata, err := areq.HandleRequest(r)
		if err != nil {
			w.Write([]byte(fmt.Sprintf("ERROR: %s\n", err)))
			return
		}

		treq := client.NewAccessRequest(osincli.AUTHORIZATION_CODE, areqdata)

		// show access request url (for debugging only)
		u2 := treq.GetTokenUrl()
		w.Write([]byte(fmt.Sprintf("Access token URL: %s\n", u2.String())))

		// exchange the authorize token for the access token
		ad, err := treq.GetToken()
		if err != nil {
			w.Write([]byte(fmt.Sprintf("ERROR: %s\n", err)))
			return
		}
		w.Write([]byte(fmt.Sprintf("Access token: %+v\n", ad)))
	})

	clienthttp.HandleFunc("/appauth/code", func(w http.ResponseWriter, r *http.Request) {
		// Parse the URL path to extract our URL parameters that were attached
		// by the oAuth 2.0 server to this API endpoint.
		r.ParseForm()

		// Extract our `Authorization Code` which we can use to get our first
		// access token / refresh token from the oAuth 2.0 server. Please
		// remember this code can only be called once!
		code := r.FormValue("code")

		w.Write([]byte("<html><body>"))
		w.Write([]byte("APP AUTH - CODE<br/>"))
		defer w.Write([]byte("</body></html>"))

		if code == "" {
			w.Write([]byte("Nothing to do"))
			return
		}

		jr := make(map[string]interface{})

		// build access code url
		aurl := fmt.Sprintf("/token?grant_type=authorization_code&client_id=%s&client_secret=%s&state=xyz&redirect_uri=%s&code=%s",
			ClientID,
			ClientSecret,
			RedirectURL,
			url.QueryEscape(code))

		fullURL := fmt.Sprintf("http://localhost:8000%s", aurl)

		// if parse, download and parse json
		if r.FormValue("doparse") == "1" {
			err := utils.DownloadAccessToken(fullURL, &osin.BasicAuth{ClientID, ClientSecret}, jr)
			if err != nil {
				w.Write([]byte(err.Error()))
				w.Write([]byte("<br/>"))
			}
		}

		// show json error
		if erd, ok := jr["error"]; ok {
			w.Write([]byte(fmt.Sprintf("ERROR: %s<br/>\n", erd)))
		}

		// show json access token
		if at, ok := jr["access_token"]; ok {
			w.Write([]byte(fmt.Sprintf("ACCESS TOKEN: %s<br/>\n", at)))
		}

		w.Write([]byte(fmt.Sprintf("FULL RESULT: %+v<br/>\n", jr)))

		// output links
		w.Write([]byte(fmt.Sprintf("<a href=\"%s\">Goto Token URL</a><br/>", fullURL)))

		cururl := *r.URL
		curq := cururl.Query()
		curq.Add("doparse", "1")
		cururl.RawQuery = curq.Encode()
		w.Write([]byte(fmt.Sprintf("<a href=\"%s\">Download Token</a><br/>", cururl.String())))

	})

	log.Println("Server started running on port 8001")
	http.ListenAndServe(":8001", clienthttp)
}
