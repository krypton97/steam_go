package steam_go

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"github.com/valyala/fasthttp"
)

var (
	steam_login = "https://steamcommunity.com/openid/login"

	openId_mode       = "checkid_setup"
	openId_ns         = "http://specs.openid.net/auth/2.0"
	openId_identifier = "http://specs.openid.net/auth/2.0/identifier_select"

	validation_regexp        = regexp.MustCompile("^(http|https)://steamcommunity.com/openid/id/[0-9]{15,25}$")
	digits_extraction_regexp = regexp.MustCompile("\\D+")
)

type OpenId struct {
	root      string
	returnUrl string
	data      *fasthttp.Args
}

func NewOpenId(ctx *fasthttp.RequestCtx) *OpenId {
	id := new(OpenId)

	proto := "http://"
	if ctx.IsTLS() {
		proto = "https://"
	}
	id.root = proto + string(ctx.Host())

	uri := string(ctx.RequestURI())
	if i := strings.Index(uri, "openid"); i != -1 {
		uri = uri[0 : i-1]
	}
	id.returnUrl = id.root + uri

	switch string(ctx.Method()) {
	case "POST":
		id.data = ctx.Request.PostArgs()
	case "GET":
		id.data = ctx.URI().QueryArgs()
	}

	return id
}

func (id OpenId) AuthUrl() string {
	data := map[string]string{
		"openid.claimed_id": openId_identifier,
		"openid.identity":   openId_identifier,
		"openid.mode":       openId_mode,
		"openid.ns":         openId_ns,
		"openid.realm":      id.root,
		"openid.return_to":  id.returnUrl,
	}

	i := 0
	url := steam_login + "?"
	for key, value := range data {
		url += key + "=" + value
		if i != len(data)-1 {
			url += "&"
		}
		i++
	}
	return url
}

func (id *OpenId) ValidateAndGetId() (string, error) {
	if !bytes.Equal(id.Mode(), []byte("id_res") ) {
		return "", errors.New("Mode must equal to \"id_res\".")
	}

	if !bytes.Equal(id.data.Peek("openid.return_to"), []byte(id.returnUrl)) {
		return "", errors.New("The \"return_to url\" must match the url of current request.")
	}

	params := make(url.Values)
	params.Set("openid.assoc_handle", string(id.data.Peek("openid.assoc_handle")))
	params.Set("openid.signed", string(id.data.Peek("openid.signed")))
	params.Set("openid.sig", string(id.data.Peek("openid.sig")))
	params.Set("openid.ns", string(id.data.Peek("openid.ns")))

	split := strings.Split(string(id.data.Peek("openid.signed")), ",")
	for _, item := range split {
		params.Set("openid."+item, string(id.data.Peek("openid."+item)))
	}
	params.Set("openid.mode", "check_authentication")

	resp, err := http.PostForm(steam_login, params)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	response := strings.Split(string(content), "\n")
	if response[0] != "ns:"+openId_ns {
		return "", errors.New("Wrong ns in the response.")
	}
	if strings.HasSuffix(response[1], "false") {
		return "", errors.New("Unable validate openId.")
	}

	openIdUrl := string(id.data.Peek("openid.claimed_id"))
	if !validation_regexp.MatchString(openIdUrl) {
		return "", errors.New("Invalid steam id pattern.")
	}

	return digits_extraction_regexp.ReplaceAllString(openIdUrl, ""), nil
}

func (id OpenId) ValidateAndGetUser(apiKey string) (*PlayerSummaries, error) {
	steamId, err := id.ValidateAndGetId()
	if err != nil {
		return nil, err
	}
	return GetPlayerSummaries(steamId, apiKey)
}

func (id OpenId) Mode() []byte {
	return id.data.Peek("openid.mode")
}
