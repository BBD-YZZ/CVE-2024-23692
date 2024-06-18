package newclient

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

type ProxyInfo struct {
	ProxyType string
	UserName  string
	PassWord  string
	IP        string
	Port      string
}

const (
	HTTP_PREFIX   = "http://"
	HTTPS_PREFIX  = "https://"
	SOCKS5_PREFIX = "socks5://"
	BASE_STRING   = "ABCDEFGHIJKLMNOPQRETUVWXYZabcdefghijklmnopqrstuvwxyz1234567890"
)

func Standar_URL(url_str string) string {
	if !strings.HasPrefix(url_str, HTTP_PREFIX) && !strings.HasPrefix(url_str, HTTPS_PREFIX) {
		url_str = fmt.Sprintf("%s%s", HTTP_PREFIX, url_str)
	}
	parse_url, err := url.ParseRequestURI(url_str)
	if err != nil {
		fmt.Println("URL parse error: ", err)
		return ""
	}

	port := parse_url.Port()
	if port == "" {
		return fmt.Sprintf("%s://%s", parse_url.Scheme, parse_url.Host)
	} else if port == "443" {
		return fmt.Sprintf("%s://%s", "https", parse_url.Host)
	} else {
		return fmt.Sprintf("%s://%s:%s", parse_url.Scheme, parse_url.Hostname(), port)
	}
}

func validateProxyFormat(info []string, filedName string, expectedLength int) error {
	if len(info) != expectedLength {
		return errors.New("Invalid proxy string format for [" + filedName + "]")
	}

	return nil
}

func proxyStringParse(proxyStr string) (ProxyInfo, error) {
	var proxyInfo ProxyInfo

	parts := strings.SplitN(proxyStr, "://", 2)
	if err := validateProxyFormat(parts, proxyStr, 2); err != nil {
		return ProxyInfo{}, err
	}

	proxyInfo.ProxyType = parts[0]

	if strings.Contains(parts[1], "@") {
		info := strings.Split(parts[1], "@")
		if err := validateProxyFormat(info, proxyStr, 2); err != nil {
			return ProxyInfo{}, err
		}

		if strings.Contains(info[0], ":") {
			userInfo := strings.Split(info[0], ":")
			if err := validateProxyFormat(userInfo, proxyStr, 2); err != nil {
				return ProxyInfo{}, err
			}
			proxyInfo.UserName = userInfo[0]
			proxyInfo.PassWord = userInfo[1]
		} else {

			if err := validateProxyFormat([]string{}, proxyStr, 2); err != nil {
				return ProxyInfo{}, err
			}
		}

		if strings.Contains(info[1], ":") {
			ipInfo := strings.Split(info[1], ":")

			if err := validateProxyFormat(ipInfo, proxyStr, 2); err != nil {
				return ProxyInfo{}, err
			}

			proxyInfo.IP = ipInfo[0]
			proxyInfo.Port = ipInfo[1]
		} else {
			proxyInfo.IP = info[1]
			proxyInfo.Port = ""
		}
	} else {
		proxyInfo.UserName = ""
		proxyInfo.PassWord = ""
		if strings.Contains(parts[1], ":") {
			ipInfo := strings.Split(parts[1], ":")

			if err := validateProxyFormat(ipInfo, proxyStr, 2); err != nil {
				return ProxyInfo{}, err
			}

			proxyInfo.IP = ipInfo[0]
			proxyInfo.Port = ipInfo[1]
		} else {
			proxyInfo.IP = parts[1]
			proxyInfo.Port = ""
		}
	}

	return proxyInfo, nil
}

func createHTTPProxy(scheme, ipPort string, userInfo *url.Userinfo) (*url.URL, error) {

	proxyURL := &url.URL{
		Scheme: scheme,
		Host:   ipPort,
		User:   userInfo,
	}

	return proxyURL, nil
}

func createSocksProxy(scheme, ipPort string, userInfo *url.Userinfo) (proxy.Dialer, error) {

	proxyURL := &url.URL{
		Scheme: scheme,
		Host:   ipPort,
		User:   userInfo,
	}

	dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
	if err != nil {
		return nil, err
	}

	return dialer, nil
}

func proxyFunc(proxyStr string) (func(*http.Request) (*url.URL, error), func(string, string) (net.Conn, error)) {

	if proxyStr == "" {
		return nil, nil
	}

	proxyInfo, err := proxyStringParse(proxyStr)
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}

	var proxyURL *url.URL
	var dialer proxy.Dialer

	switch proxyInfo.ProxyType {
	case "http":
		userInfo := url.UserPassword(proxyInfo.UserName, proxyInfo.PassWord)
		proxyURL, err = createHTTPProxy("http", fmt.Sprintf("%s:%s", proxyInfo.IP, proxyInfo.Port), userInfo)
		if err != nil {
			fmt.Printf("Error creating HTTP proxy: %v\n", err)
			return nil, nil
		}
		return http.ProxyURL(proxyURL), nil
	case "socks5":
		userInfo := url.UserPassword(proxyInfo.UserName, proxyInfo.PassWord)
		dialer, err = createSocksProxy("socks5", fmt.Sprintf("%s:%s", proxyInfo.IP, proxyInfo.ProxyType), userInfo)
		if err != nil {
			fmt.Printf("Error creating SOCKS proxy: %v\n", err)
			return nil, nil
		}
		return nil, dialer.Dial
	default:
		fmt.Println("Unsupported proxy type")
		return nil, nil
	}

}

func clientFUNC(proxy_str string, i int) *http.Client {
	proxy, dail := proxyFunc(proxy_str)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Proxy:           proxy,
		Dial:            dail,
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * time.Duration(i),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 如果你想要允许重定向，返回nil；如果不想要，返回错误
			fmt.Println("Redirect from", via[0].URL, "to", req.URL)
			return http.ErrUseLastResponse // 返回这个错误会停止重定向， 禁止重定向
		},
	}

	return client
}

func ReturnUA() string {
	rand.Seed(time.Now().UnixNano())
	user_agent := []string{
		"Mozilla/5.0 (Windows; U; Win98; en-US; rv:1.8.1) Gecko/20061010 Firefox/2.0",
		"Mozilla/5.0 (Windows; U; Windows NT 5.0; en-US) AppleWebKit/532.0 (KHTML, like Gecko) Chrome/3.0.195.6 Safari/532.0",
		"Mozilla/5.0 (Windows; U; Windows NT 5.1 ; x64; en-US; rv:1.9.1b2pre) Gecko/20081026 Firefox/3.1b2pre",
		"Opera/10.60 (Windows NT 5.1; U; zh-cn) Presto/2.6.30 Version/10.60", "Opera/8.01 (J2ME/MIDP; Opera Mini/2.0.4062; en; U; ssr)",
		"Mozilla/5.0 (Windows; U; Windows NT 5.1; ; rv:1.9.0.14) Gecko/2009082707 Firefox/3.0.14",
		"Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.106 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/57.0.2987.133 Safari/537.36",
		"Mozilla/5.0 (Windows; U; Windows NT 6.0; fr; rv:1.9.2.4) Gecko/20100523 Firefox/3.6.4 ( .NET CLR 3.5.30729)",
		"Mozilla/5.0 (Windows; U; Windows NT 6.0; fr-FR) AppleWebKit/528.16 (KHTML, like Gecko) Version/4.0 Safari/528.16",
		"Mozilla/5.0 (Windows; U; Windows NT 6.0; fr-FR) AppleWebKit/533.18.1 (KHTML, like Gecko) Version/5.0.2 Safari/533.18.5",
		"Mozilla/5.0 (Windows; U; Windows NT 6.1; en-US) AppleWebKit/533.20.25 (KHTML, like Gecko) Version/5.0.4 Safari/533.20.27",
		"Mozilla/5.0 (Windows NT 6.1; WOW64; rv:23.0) Gecko/20130406 Firefox/23.0",
		"Opera/9.80 (Windows NT 5.1; U; zh-sg) Presto/2.9.181 Version/12.00",
		"Mozilla/5.0 (Linux; U; Android 3.0; en-us; Xoom Build/HRI39) AppleWebKit/534.13 (KHTML, like Gecko) Version/4.0 Safari/534.13",
		"Opera/9.80 (Android 2.3.4; Linux; Opera Mobi/build-1107180945; U; en-GB) Presto/2.8.149 Version/11.10",
		"Mozilla/5.0 (Linux; U; Android 2.3.7; en-us; Nexus One Build/FRF91) AppleWebKit/533.1 (KHTML, like Gecko) Version/4.0 Mobile Safari/533.1",
		"Mozilla/5.0 (iPhone; U; CPU iPhone OS 4_3_3 like Mac OS X; en-us) AppleWebKit/533.17.9 (KHTML, like Gecko) Version/5.0.2 Mobile/8J2 Safari/6533.18.5",
		"MQQBrowser/26 Mozilla/5.0 (Linux; U; Android 2.3.7; zh-cn; MB200 Build/GRJ22; CyanogenMod-7) AppleWebKit/533.1 (KHTML, like Gecko) Version/4.0 Mobile Safari/533.1",
	}

	randomIndex := rand.Intn(len(user_agent))
	return user_agent[randomIndex]
}

func SendRequest(method, url, proxy string, i int, body io.Reader, headers map[string]string) (*http.Response, error) {
	client := clientFUNC(proxy, i)
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		request.Header.Set(k, v)
	}
	return client.Do(request)
}
