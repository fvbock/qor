package api

import (
	"github.com/qor/qor"

	"net/http"
	"path"
	"regexp"
	"strings"
)

func (api *API) generateContext(w http.ResponseWriter, r *http.Request) *qor.Context {
	context := qor.Context{Writer: w, Request: r, DB: api.DB}
	if api.auth != nil {
		context.CurrentUser = api.auth.GetCurrentUser(&context)
	}
	return &context
}

func (api *API) AddToMux(prefix string, mux *http.ServeMux) {
	prefix = regexp.MustCompile("//(//)*").ReplaceAllString("/"+prefix+"/", "/")
	api.Prefix = prefix

	pathMatch := regexp.MustCompile(path.Join(prefix, `(\w+)(?:/(\w+))?[^/]*/?$`))
	mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
		var isIndexURL, isShowURL bool
		context := api.generateContext(w, r)

		if len(r.Form["_method"]) > 0 {
			r.Method = strings.ToUpper(r.Form["_method"][0])
		}

		matches := pathMatch.FindStringSubmatch(r.URL.Path)
		if len(matches) == 0 {
			http.NotFound(w, r)
			return
		}

		if _, ok := api.Resources[matches[1]]; matches[1] != "" && ok {
			isIndexURL = true
			context.ResourceName = matches[1]

			if matches[2] != "" { // "/admin/user/1234"
				context.ResourceID = matches[2]
				isIndexURL = false
				isShowURL = true
			}
		}

		switch {
		case r.Method == "GET" && isIndexURL:
			api.Index(context)
		case r.Method == "GET" && isShowURL:
			api.Show(context)
		case r.Method == "POST" && isShowURL:
			api.Update(context)
		case r.Method == "POST" && isIndexURL:
			api.Create(context)
		case r.Method == "DELETE" && isShowURL:
			api.Delete(context)
		default:
			http.NotFound(w, r)
		}
	})
}
