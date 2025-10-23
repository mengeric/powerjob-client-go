package client

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    . "github.com/smartystreets/goconvey/convey"
)

func TestHTTPServerAPI_Basic(t *testing.T) {
    Convey("AssertApp & Acquire should work", t, func() {
        // 准备：模拟 server
        mux := http.NewServeMux()
        mux.HandleFunc("/server/assert", func(w http.ResponseWriter, r *http.Request) {
            _ = json.NewEncoder(w).Encode(CommonResp[int64]{Success:true, Data: 123})
        })
        mux.HandleFunc("/server/acquire", func(w http.ResponseWriter, r *http.Request) {
            _ = json.NewEncoder(w).Encode(CommonResp[string]{Success:true, Data: "127.0.0.1:10010"})
        })
        ts := httptest.NewServer(mux); defer ts.Close()

        // 解析 host
        host := ts.Listener.Addr().String()
        api := NewHTTPServerAPI()

        appID, err := api.AssertApp(context.Background(), host, "demo")
        So(err, ShouldBeNil)
        So(appID, ShouldEqual, 123)

        addr, err := api.Acquire(context.Background(), host, appID, "", "0.1.0")
        So(err, ShouldBeNil)
        So(addr, ShouldEqual, "127.0.0.1:10010")
    })
}

