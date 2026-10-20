package main

import (
	"fmt"
	"net/http/httptest"
	"os"
	"time"

	"github.com/bigbigliu/go-core/fileutil"
	"github.com/bigbigliu/go-core/idgen"
	"github.com/bigbigliu/go-core/netutil"
	"github.com/bigbigliu/go-core/strutil"
	"github.com/bigbigliu/go-core/timeutil"
	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("snowflake id:", idgen.GenUniversalID())

	uuid7, err := idgen.UUIDv7()
	if err != nil {
		fmt.Fprintln(os.Stderr, "uuidv7:", err)
	} else {
		fmt.Println("uuid v7:", uuid7)
	}
	fmt.Println("ints:", strutil.IntsToString([]int64{1, 2, 3}))
	fmt.Println("time:", timeutil.Time2String(time.Now()))

	greater, err := fileutil.IsFileGreaterThan(os.Args[0], 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "file size:", err)
	} else {
		fmt.Println("binary larger than 0MB:", greater)
	}

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "203.0.113.9:1234"
	fmt.Println("remote ip:", netutil.GetRemoteIP(c))
}
