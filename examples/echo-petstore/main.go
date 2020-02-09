package main

import (
	"github.com/labstack/echo/v4"
	"github.com/tamasfe/repose/examples/echo-petstore/src/api"
	"github.com/tamasfe/repose/examples/echo-petstore/src/server"
)

func main() {
	e := echo.New()
	e.Server.Addr = ":8080"

	api.RegisterEchoServer(e, &server.ServerImpl{})

	panic(e.Server.ListenAndServe())
}
