package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestHTTPServerAPI_Errors(t *testing.T) {
	Convey("AssertApp should return error when success=false or non-2xx", t, func() {
		mux := http.NewServeMux()
		// success=false
		mux.HandleFunc("/server/assert", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(CommonResp[int64]{Success: false, Message: "app not found"})
		})
		ts := httptest.NewServer(mux)
		defer ts.Close()
		host := ts.Listener.Addr().String()
		api := NewHTTPServerAPI()
		_, err := api.AssertApp(context.Background(), host, "none")
		So(err, ShouldNotBeNil)
	})

	Convey("Acquire should return error when non-2xx", t, func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/server/acquire", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "bad", http.StatusBadRequest)
		})
		ts := httptest.NewServer(mux)
		defer ts.Close()
		host := ts.Listener.Addr().String()
		api := NewHTTPServerAPI()
		_, err := api.Acquire(context.Background(), host, 1, "", "0.1.0")
		So(err, ShouldNotBeNil)
	})
}
