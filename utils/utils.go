package utils

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/qor/qor"

	"strings"
)

// Humanize separates string based on capitalizd letters
// e.g. "OrderItem" -> "Order Item"
func HumanizeString(str string) string {
	var human []rune
	for i, l := range str {
		if i > 0 && isUppercase(byte(l)) {
			if i > 0 && !isUppercase(str[i-1]) || i+1 < len(str) && !isUppercase(str[i+1]) {
				human = append(human, rune(' '))
			}
		}
		human = append(human, l)
	}
	return strings.Title(string(human))
}

func isUppercase(char byte) bool {
	return 'A' <= char && char <= 'Z'
}

// ToParamString replaces spaces and separates words (by uppercase letters) with
// underscores in a string, also downcase it
// e.g. ToParamString -> to_param_string, To ParamString -> to_param_string
func ToParamString(str string) string {
	if len(str) <= 1 {
		return strings.ToLower(str)
	}

	str = strings.Replace(str, " ", "_", -1)
	result := []rune{rune(str[0])}
	for _, l := range str[1:] {
		if rune('A') <= l && l <= rune('Z') {
			if lr := len(result); lr == 0 || result[lr-1] != '_' {
				result = append(result, rune('_'), l)
				continue
			}
		}

		result = append(result, l)
	}

	return strings.ToLower(string(result))
}

// PatchURL updates the query part of the current request url. You can
// access it in template by `patch_url`.
//     patch_url "key" "value"
func PatchURL(originalURL string, params ...interface{}) (patchedURL string, err error) {
	url, err := url.Parse(originalURL)
	if err != nil {
		return
	}

	query := url.Query()
	for i := 0; i < len(params)/2; i++ {
		// Check if params is key&value pair
		key := fmt.Sprintf("%v", params[i*2])
		value := fmt.Sprintf("%v", params[i*2+1])

		if value == "" {
			query.Del(key)
		} else {
			query.Set(key, value)
		}
	}

	url.RawQuery = query.Encode()
	patchedURL = url.String()
	return
}

func GetLocale(context *qor.Context) string {
	var locale = context.Request.URL.Query().Get("locale")

	if locale == "" {
		locale = context.Request.Form.Get("locale")
	}

	if locale != "" {
		if context.Writer != nil {
			cookie := http.Cookie{Name: "locale", Value: locale, Expires: time.Now().AddDate(1, 0, 0), Path: "/"}
			http.SetCookie(context.Writer, &cookie)
		}
		return locale
	}

	if locale, err := context.Request.Cookie("locale"); err == nil {
		return locale.Value
	}

	return ""
}

func Stringify(object interface{}) string {
	if obj, ok := object.(interface {
		Stringify() string
	}); ok {
		return obj.Stringify()
	}

	scope := gorm.Scope{Value: object}
	for _, column := range []string{"Name", "Title"} {
		if field, ok := scope.FieldByName(column); ok {
			return fmt.Sprintf("%v", field.Field.Interface())
		}
	}
	return fmt.Sprintf("%v", object)
}

func ParseTagOption(str string) map[string]string {
	tags := strings.Split(str, ";")
	setting := map[string]string{}
	for _, value := range tags {
		v := strings.Split(value, ":")
		k := strings.TrimSpace(strings.ToUpper(v[0]))
		if len(v) == 2 {
			setting[k] = v[1]
		} else {
			setting[k] = k
		}
	}
	return setting
}
