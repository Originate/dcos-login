// Package dcoslogin provides a way to login to a Community Edition DC/OS cluster unattended
package dcoslogin

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Debug can be set to true for (very) verbose output, helpful for troubleshooting OAuth issues
var Debug = false

// Options has the parameters needed to login to DC/OS
type Options struct {
	ClusterURL *string

	Username *string
	Password *string

	AllowInsecureTLS *bool
}

// Login simulates a user loging in to a Community Edition DC/OS cluster using Github credentials
func Login(o *Options) error {
	client, err := httpClient(*o.AllowInsecureTLS)
	if err != nil {
		return err
	}

	// Hit the DC/OS login endpoint to retrieve the clusterID and the clientID
	clusterID, clientID, err := client.initiateLogin(*o.ClusterURL)
	if err != nil {
		return err
	}

	// DC/OS uses Auth0, initiate the session to get the CSRF token
	csrfToken, err := client.initiateAuth0(clusterID, clientID)
	if err != nil {
		return err
	}

	// Authenticate with Github, get the ACS token back
	acsToken, err := client.githubAuthenticate(csrfToken, *o.Username, *o.Password)
	if err != nil {
		return err
	}

	fmt.Println(acsToken)

	return nil
}

type client struct {
	http.Client
}

func httpClient(allowInsecureTLS bool) (*client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	return &client{
		http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: allowInsecureTLS},
			},
			Jar: jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				debug(req.Method, req.URL)
				return nil
			},
		},
	}, nil
}

func (c *client) Get(endpoint string, query url.Values) (*http.Response, error) {
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	debug("GET", endpoint)

	res, err := c.Client.Get(endpoint)
	if err != nil {
		return nil, err
	}

	if err := checkStatus(res); err != nil {
		return nil, err
	}

	return res, nil
}

func (c *client) PostForm(endpoint string, data url.Values) (*http.Response, error) {
	res, err := c.Client.Post(endpoint, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	if err := checkStatus(res); err != nil {
		return nil, err
	}

	return res, nil
}

func (c *client) initiateLogin(clusterURL string) (string, string, error) {
	res, err := c.Get(clusterURL+"/login", url.Values{
		"redirect_uri": []string{"urn:ietf:wg:oauth:2.0:oob"},
	})
	if err != nil {
		return "", "", err
	}

	clusterID := res.Request.URL.Query().Get("cluster_id")
	clientID := res.Request.URL.Query().Get("client")

	return clusterID, clientID, nil
}

func (c *client) initiateAuth0(clusterID, clientID string) (string, error) {
	res, err := c.Get("https://dcos.auth0.com/authorize", url.Values{
		"scope":         []string{"openid email"},
		"response_type": []string{"token"},
		"connection":    []string{"github"},
		"cluster_id":    []string{clusterID},
		"client_id":     []string{clientID},
		"owp":           []string{"true"},
	})
	if err != nil {
		return "", err
	}

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		return "", err
	}

	csrf, found := doc.Find(`input[name="authenticity_token"]`).Attr("value")
	if !found {
		return "", errors.New("Unable to extract CSRF token from response")
	}

	return csrf, nil
}

func (c *client) githubAuthenticate(csrfToken, username, password string) (string, error) {
	ghRes, err := c.PostForm("https://github.com/session", url.Values{
		"login":              []string{username},
		"password":           []string{password},
		"authenticity_token": []string{csrfToken},
	})
	if err != nil {
		return "", err
	}

	tokenRes, err := c.followLoginRedirect(ghRes)
	if err != nil {
		return "", err
	}

	return getLoginToken(tokenRes)
}

func (c *client) followLoginRedirect(res *http.Response) (*http.Response, error) {
	dump, err := httputil.DumpResponse(res, true)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		return nil, err
	}

	redirectURL, found := doc.Find(".container div p a").Attr("href")
	// Easy path, no re-authorization
	if found {
		return c.Get(redirectURL, nil)
	}

	// Check if Github is simply asking for re-authorization
	authorizeForm := doc.Find(`form[action="/login/oauth/authorize"]`)
	if authorizeForm.Length() != 1 {
		debug(string(dump))
		return nil, errors.New("Unexpected Github response")
	}

	// Re-authorize
	q := url.Values{
		"authorize": []string{"1"},
	}
	authorizeForm.Find("input").Each(func(_ int, input *goquery.Selection) {
		name, _ := input.Attr("name")
		value, _ := input.Attr("value")
		q.Add(name, value)
	})

	return c.PostForm("https://github.com/login/oauth/authorize", q)
}

// The DC/OS frontend exposes the token as a variable in a JS script. This (somewhat hackily) extracts it.
func getLoginToken(res *http.Response) (string, error) {
	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		return "", err
	}

	script := doc.Find(`script[type="text/javascript"]`).Last().Text()

	tokenMatcher := regexp.MustCompile(`var value [^"]+"([^"]+)"[^;]+;`)

	var base64Token string
	if matches := tokenMatcher.FindStringSubmatch(script); len(matches) == 2 {
		base64Token = matches[1]
	} else {
		return "", errors.New("Couldn't extract ACS token from response")
	}

	jsonToken, err := base64.StdEncoding.DecodeString(base64Token)
	if err != nil {
		return "", err
	}

	var jwt struct {
		IDToken string `json:"id_token"`
	}
	err = json.Unmarshal(jsonToken, &jwt)
	if err != nil {
		return "", err
	}

	return jwt.IDToken, nil
}

func checkStatus(res *http.Response) error {
	if res.StatusCode < 200 || res.StatusCode > 206 {
		dump, err := httputil.DumpResponse(res, true)
		if err == nil {
			return fmt.Errorf("Expected status 200 <= code <= 206, got %v\n%s", res.StatusCode, string(dump))
		}

		return fmt.Errorf("Expected status 200 <= code <= 206, got %v", res.StatusCode)
	}

	return nil
}

func debug(a ...interface{}) {
	if Debug {
		log.Println(a...)
	}
}
