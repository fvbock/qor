package api

import (
	"fmt"

	"github.com/qor/qor"
	"github.com/qor/qor/resource"

	"net/http"
	"path"
)

func (api *API) Index(context *qor.Context) {
	resource := api.Resources[context.ResourceName]
	result := resource.NewSlice()
	api.DB.Find(result)
}

func (api *API) Show(context *qor.Context) {
	resource := api.Resources[context.ResourceName]

	result := resource.NewStruct()
	api.DB.First(result, context.ResourceID)
}

func (api *API) Create(context *qor.Context) {
	res := api.Resources[context.ResourceName]

	result := res.NewStruct()
	resource.DecodeToResource(res, result, ConvertJSONToMetaValues(context, "QorResource.", res), context).Start()
	api.DB.Save(result)

	primaryKey := fmt.Sprintf("%v", api.DB.NewScope(result).PrimaryKeyValue())
	http.Redirect(context.Writer, context.Request, path.Join(context.Request.RequestURI, primaryKey), http.StatusFound)
}

func (api *API) Update(context *qor.Context) {
	res := api.Resources[context.ResourceName]
	result := res.NewStruct()

	if !api.DB.First(result, context.ResourceID).RecordNotFound() {
		// resource.DecodeToResource(res, result, ConvertFormToMetaValues(context, "QorResource.", res), context).Start()
		api.DB.Save(result)
		http.Redirect(context.Writer, context.Request, context.Request.RequestURI, http.StatusFound)
	}
}

func (api *API) Delete(context *qor.Context) {
	res := api.Resources[context.ResourceName]
	result := res.NewStruct()

	if api.DB.Delete(result, context.ResourceID).RowsAffected > 0 {
		http.Redirect(context.Writer, context.Request, path.Join(api.Prefix, res.Name), http.StatusFound)
	} else {
		http.Redirect(context.Writer, context.Request, path.Join(api.Prefix, res.Name), http.StatusNotFound)
	}
}
