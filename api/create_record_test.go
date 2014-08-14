package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestCreateRecord(t *testing.T) {
	json := `{"Name":"api_create_record", "Role":"admin"}`
	buf := strings.NewReader(json)

	if req, err := http.Post(server.URL+"/api/user", "application/json", buf); err == nil {
		fmt.Println(req)
		fmt.Println(err)
		t.Errorf("sss")
	}
}
