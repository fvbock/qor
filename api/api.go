package api

import (
	"github.com/jinzhu/gorm"
	"github.com/qor/qor/auth"
	"github.com/qor/qor/resource"

	"reflect"
	"strings"
)

type API struct {
	Prefix    string
	DB        *gorm.DB
	Resources map[string]*Resource
	auth      auth.Auth
}

func (api *API) SetAuth(auth auth.Auth) {
	api.auth = auth
}

func NewResource(value interface{}, names ...string) *Resource {
	name := strings.ToLower(reflect.Indirect(reflect.ValueOf(value)).Type().Name())
	for _, n := range names {
		name = n
	}

	return &Resource{Name: name, Resource: resource.Resource{Value: value}}
}

func (api *API) NewResource(value interface{}, names ...string) *Resource {
	res := NewResource(value, names...)
	api.Resources[res.Name] = res
	return res
}

func (api *API) UseResource(res *Resource) {
	api.Resources[res.Name] = res
}
