# steam_go
> Simple steam auth util

### Installation
```
$ go get github.com/krypton97/steam_go
```
### Usage
Just <code>go run main.go</code> in example dir and open [localhost:8081/login](http://localhost:8081/login) link to see how it works

Code from ./example/main.go:
```
package main

import (
	"bytes"
	"net/http"

	"github.com/krypton97/steam_go"
	"github.com/valyala/fasthttp"
)

var apiKey = []byte("75BEBCDB358BDE8BF6CA916938F12231")

func loginHandler(ctx *fasthttp.RequestCtx) {
	opID := steam_go.NewOpenId(ctx)
	switch true {
	case bytes.Equal(opID.Mode(), []byte("")):
		ctx.Redirect(opID.AuthUrl(), 301)
	case bytes.Equal(opID.Mode(), []byte("cancel")):
		ctx.Write([]byte("Authorization cancelled"))
	default:
		steamID, err := opID.ValidateAndGetId()
		if err != nil {
			ctx.Error(err.Error(), http.StatusInternalServerError)
		}
		// Do whatever you want with steam id
		user, err := steam_go.GetPlayerSummaries(steamID, apiKey)
		if err != nil {
			ctx.Write([]byte("No user found!"))
		} else {
			ctx.Write([]byte(user.PersonaName))
		}
	}
}


func main() {
	fasthttp.ListenAndServe(":3000", loginHandler)
}

```
