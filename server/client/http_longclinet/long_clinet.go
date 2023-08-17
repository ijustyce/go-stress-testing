package httplongclinet

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/link1st/go-stress-testing/model"
	"golang.org/x/net/http2"
)

var (
	mutex   sync.RWMutex
	clients = make(map[uint64]*http.Client, 0)
)

// NewClient new
func NewClient(i uint64, request *model.Request) *http.Client {
	client := getClient(i)
	if client != nil {
		return client
	}
	return setClient(i, request)
}

func getClient(i uint64) *http.Client {
	mutex.RLock()
	defer mutex.RUnlock()
	return clients[i]
}

func setClient(i uint64, request *model.Request) *http.Client {
	mutex.Lock()
	defer mutex.Unlock()
	client := createLangHttpClient(request)
	clients[i] = client
	return client
}

type MyConnectPool struct {
	conn *http2.ClientConn
	tr   *http2.Transport
	lock sync.Mutex
}

func (m *MyConnectPool) GetClientConn(req *http.Request, addr string) (*http2.ClientConn, error) {
	if m.conn != nil {
		return m.conn, nil
	}

	m.lock.Lock()
	defer func() {
		m.lock.Unlock()
	}()

	if m.conn != nil {
		return m.conn, nil
	}

	tmp, _ := m.tr.DialTLSContext(req.Context(), "tcp", addr, nil)
	m.conn, _ = m.tr.NewClientConn(tmp)
	return m.conn, nil
}

func (m *MyConnectPool) MarkDead(conn *http2.ClientConn) {
	//m.conn = nil
}

var index = 1

// createLangHttpClient 初始化长连接客户端参数
func createLangHttpClient(request *model.Request) *http.Client {
	if request.HTTP2 {
		var tr3 = &http2.Transport{
			AllowHTTP: true, //充许非加密的链接
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
			DialTLSContext: func(_ context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				log.Default().Println("dia ", index)
				index++
				conn, err := net.DialTimeout(network, addr, 30*time.Second)
				if err != nil {
					panic(err)
				}
				return conn, err
			},
			MaxReadFrameSize:   1024 * 256,
			DisableCompression: true,
		}

		tr3.ConnPool = &MyConnectPool{tr: tr3}
		return &http.Client{Transport: tr3}
	} else {
		// 跳过证书验证
		tr := &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        0,                // 最大连接数,默认0无穷大
			MaxIdleConnsPerHost: request.MaxCon,   // 对每个host的最大连接数量(MaxIdleConnsPerHost<=MaxIdleConns)
			IdleConnTimeout:     90 * time.Second, // 多长时间未使用自动关闭连接
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		}
		tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			log.Default().Println("dia ", index)
			index++
			tmp := &net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}
			return tmp.DialContext(ctx, network, addr)
		}
		return &http.Client{
			Transport: tr,
		}
	}
}
