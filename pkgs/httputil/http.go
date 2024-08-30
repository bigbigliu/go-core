package httputil

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
)

// IHttpClient ...
type IHttpClient interface {
	// HTTPRequest 发起通用的HTTP请求，并返回响应和状态码
	SendGETRequest(urlStr string, headers map[string]string) ([]byte, int, error)
	// SendJSONPOSTRequest 发起POST请求，Content-Type为application/json，并返回响应和状态码
	SendJSONPOSTRequest(urlStr string, headers map[string]string, requestBody map[string]interface{}) ([]byte, int, error)
	// SendFormPOSTRequest 发起POST请求，Content-Type为application/x-www-form-urlencoded，并返回响应和状态码
	SendFormPOSTRequest(urlStr string, headers map[string]string, formData url.Values) ([]byte, int, error)
	// StructToQueryParams 将结构体字段绑定到 URL 参数
	StructToQueryParams(input interface{}) (string, error)
}

// HttpClientOption ...
type HttpClientOption struct{}

// HTTPRequest 发起通用的HTTP请求，并返回响应和状态码
func HTTPRequest(urlStr, method, contentType string, headers map[string]string, body interface{}) ([]byte, int, error) {
	var requestBody []byte

	// 根据contentType选择请求体格式
	switch contentType {
	case "application/json":
		if jsonBody, ok := body.(map[string]interface{}); ok {
			jsonBytes, err := json.Marshal(jsonBody)
			if err != nil {
				return nil, 0, err
			}
			requestBody = jsonBytes
		}
	case "application/x-www-form-urlencoded":
		if formBody, ok := body.(url.Values); ok {
			requestBody = []byte(formBody.Encode())
		}
	}

	req, err := http.NewRequest(method, urlStr, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, 0, err
	}

	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 发起HTTP请求
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}

	defer resp.Body.Close()
	newBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	newCode := resp.StatusCode
	return newBody, newCode, nil
}

// SendGETRequest 发起GET请求，并返回响应和状态码
func (h *HttpClientOption) SendGETRequest(urlStr string, headers map[string]string) ([]byte, int, error) {
	respByte, statusCode, err := HTTPRequest(urlStr, "GET", "", headers, nil)
	if err != nil {
		return nil, statusCode, err
	}

	return respByte, statusCode, nil
}

// SendJSONPOSTRequest 发起POST请求，Content-Type为application/json，并返回响应和状态码
func (h *HttpClientOption) SendJSONPOSTRequest(urlStr string, headers map[string]string, requestBody map[string]interface{}) ([]byte, int, error) {
	respByte, statusCode, err := HTTPRequest(urlStr, "POST", "application/json", headers, requestBody)
	if err != nil {
		return nil, statusCode, err
	}

	return respByte, statusCode, nil
}

// SendFormPOSTRequest 发起POST请求，Content-Type为application/x-www-form-urlencoded，并返回响应和状态码
func (h *HttpClientOption) SendFormPOSTRequest(urlStr string, headers map[string]string, formData url.Values) ([]byte, int, error) {
	respByte, statusCode, err := HTTPRequest(urlStr, "POST", "application/x-www-form-urlencoded", headers, formData)
	if err != nil {
		return nil, statusCode, err
	}

	return respByte, statusCode, nil
}

// StructToQueryParams 将结构体字段绑定到 URL 参数
func (h *HttpClientOption) StructToQueryParams(input interface{}) (string, error) {
	values := url.Values{}
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return "", fmt.Errorf("input must be a struct or a pointer to a struct")
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Tag.Get("queryparam")
		if fieldName == "" {
			fieldName = field.Name
		}
		value := v.Field(i)

		// 仅在 int 类型的字段不为零时添加到查询字符串中
		if value.Kind() == reflect.Int && value.Int() == 0 {
			continue
		}

		values.Add(fieldName, fmt.Sprintf("%v", value))
	}

	return values.Encode(), nil
}
