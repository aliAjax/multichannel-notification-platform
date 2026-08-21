package main

import "net/http"

func newConsoleHandler(dir string) http.Handler { return http.NotFoundHandler() }
