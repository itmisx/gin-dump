package gindump

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/itmisx/logx"
)

func init() {
	gin.SetMode(gin.TestMode)
	logx.Init(logx.Config{
		Debug:  true,
		Output: "console",
	}, logx.String("service_name", "dev"))
}

func performRequest(r http.Handler, method, contentType string, path string, body io.Reader) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestMIMEPlain(t *testing.T) {
	router := gin.New()
	router.Use(Dump())

	router.POST("/dump", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok":   true,
			"data": "gin-dump",
		})
	})
	body := bytes.NewBuffer([]byte("plain text"))
	performRequest(router, "POST", gin.MIMEPlain, "/dump", body)
	<-make(chan int) // wait loki push
}

func TestMIMEJSON(t *testing.T) {
	router := gin.New()
	router.Use(Dump())

	router.POST("/dump", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok":   true,
			"data": "gin-dump",
		})
	})

	type params struct {
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}

	var httpdata = params{
		StartTime: "2019-03-03",
		EndTime:   "2019-03-03",
	}
	b, err := json.Marshal(httpdata)
	if err != nil {
		fmt.Println("json format error:", err)
		return
	}

	body := bytes.NewBuffer(b)
	performRequest(router, "POST", gin.MIMEJSON, "/dump", body)
	<-make(chan int) // wait loki push
}

func TestMIMEPOSTFORM(t *testing.T) {
	router := gin.New()
	router.Use(Dump())

	router.POST("/dump", func(c *gin.Context) {
		bts, err := httputil.DumpRequest(c.Request, true)
		fmt.Println(string(bts), err)

		c.JSON(http.StatusOK, gin.H{
			"ok": true,
			"data": map[string]interface{}{
				"name": "jfise",
				"addr": "tpkeeper@qq.com",
			},
		})
	})

	form := make(url.Values)
	form.Set("foo", "bar")
	form.Add("foo", "bar2")
	form.Set("bar", "baz")

	body := strings.NewReader(form.Encode())
	performRequest(router, "POST", gin.MIMEPOSTForm, "/dump", body)
	<-make(chan int) // wait loki push
}
