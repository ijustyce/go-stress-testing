// Package client http 客户端
package client

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/link1st/go-stress-testing/model"
	httplongclinet "github.com/link1st/go-stress-testing/server/client/http_longclinet"
	"golang.org/x/net/http2"

	"github.com/link1st/go-stress-testing/helper"
)

// logErr err
var logErr = log.New(os.Stderr, "", 0)

// HTTPRequest HTTP 请求
// method 方法 GET POST
// url 请求的url
// body 请求的body
// headers 请求头信息
// timeout 请求超时时间
func HTTPRequest(chanID uint64, request *model.Request) (resp *http.Response, requestTime uint64, err error) {
	method := request.Method
	url := request.URL
	body := request.GetBody()
	timeout := request.Timeout
	headers := request.Headers

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return
	}

	// 在req中设置Host，解决在header中设置Host不生效问题
	if _, ok := headers["Host"]; ok {
		req.Host = headers["Host"]
	}
	// 设置默认为utf-8编码
	if _, ok := headers["Content-Type"]; !ok {
		if headers == nil {
			headers = make(map[string]string)
		}
		headers["Content-Type"] = "application/x-www-form-urlencoded; charset=utf-8"
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	var client *http.Client
	if request.Keepalive {
		client = httplongclinet.NewClient(chanID, request)
		startTime := time.Now()
		resp, err = client.Do(req)
		requestTime = uint64(helper.DiffNano(startTime))
		if err != nil {
			logErr.Println("请求失败:", err)

			return
		}
		return
	} else {
		req.Close = true
		if request.HTTP2 {
			// 使用真实证书 验证证书 模拟真实请求
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			}
			tr2, err2 := http2.ConfigureTransports(tr)
			if err2 != nil {
				return
			}
			tr2.AllowHTTP = true
			tr2.DialTLSContext = func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				conn, err := net.DialTimeout(network, addr, 30*time.Second)
				if err != nil {
					panic(err)
				}
				return conn, err
			}
			client = &http.Client{
				Transport: tr2,
				Timeout:   timeout,
			}
		} else {
			// 跳过证书验证
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client = &http.Client{
				Transport: tr,
				Timeout:   timeout,
			}
		}
	}

	startTime := time.Now()
	resp, err = client.Do(req)
	requestTime = uint64(helper.DiffNano(startTime))
	if err != nil {
		logErr.Println("请求失败:", err)

		return
	}
	return
}
