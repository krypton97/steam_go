package steam_go

import (
	"bytes"
	"errors"
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
	
	if ctx.IsPost() {
		id.data = ctx.Request.PostArgs()
	} else if ctx.IsGet() {
		id.data = ctx.Request.QueryArgs()
	}

	return id
}

func (id OpenId) AuthUrl() string {
	keys := [...]string{ "openid.claimed_id", "openid.identity", "openid.mode", "openid.ns", "openid.realm", "openid.return_to"}
	values := [...]string{ openId_identifier, openId_identifier, openId_mode, openId_ns, id.root, id.returnUrl }

	url := steam_login + "?"
	for i := 0; i < len(keys); i++ {
	    url += keys[i] + "=" + values[i]
		if i != len(keys) - 1 {
			url += "&"
		}
	}
	return url
}

func (id *OpenId) ValidateAndGetId() ([]byte, error) {
	if !bytes.Equal(id.Mode(), []byte("id_res") ) {
		return []byte{}, errors.New("Mode must equal to \"id_res\".")
	}

	if !bytes.Equal(id.data.Peek("openid.return_to"), []byte(id.returnUrl)) {
		return []byte{}, errors.New("The \"return_to url\" must match the url of current request.")
	}
	
	params := fasthttp.AcquireArgs()
	params.AddBytesV("openid.assoc_handle", id.data.Peek("openid.assoc_handle"))
	params.AddBytesV("openid.signed", id.data.Peek("openid.signed"))
	params.AddBytesV("openid.sig", id.data.Peek("openid.sig"))
	params.AddBytesV("openid.ns", id.data.Peek("openid.ns"))
	
	split := bytes.Split(id.data.Peek("openid.signed"), []byte{','})
	for _, item := range split {
		params.AddBytesV("openid."+ string(item), id.data.Peek("openid."+string(item)))
	}
	params.AddBytesV("openid.mode", []byte("check_authentication"))

	_, content, err := fasthttp.Post(nil, steam_login, params)
	if err != nil {
		return []byte{}, err
	}

	response := bytes.Split(content, []byte{'\n'})
	if !bytes.Equal(response[0], []byte("ns:" + openId_ns)) {
		return []byte{}, errors.New("Wrong ns in the response.")
	}
	if bytes.HasSuffix(response[1], []byte("false")) {
		return []byte{}, errors.New("Unable validate openId.")
	}

	openIdUrl := id.data.Peek("openid.claimed_id")
	if !validation_regexp.Match(openIdUrl) {
		return []byte{}, errors.New("Invalid steam id pattern.")
	}

	return digits_extraction_regexp.ReplaceAll(openIdUrl, []byte("")), nil
}

func (id OpenId) ValidateAndGetUser(apiKey []byte) (*PlayerSummaries, error) {
	steamId, err := id.ValidateAndGetId()
	if err != nil {
		return nil, err
	}
	return GetPlayerSummaries(steamId, apiKey)
}

func (id OpenId) Mode() []byte {
	return id.data.Peek("openid.mode")
}
