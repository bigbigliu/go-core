package httputil

import (
	"fmt"
	"net/url"
)

func example() {
	// 示例用法1: 发起GET请求
	getURL := "https://example.com/api/get_endpoint"
	getHeaders := map[string]string{
		"Authorization": "Bearer your_access_token",
		"Other-Header":  "other-value",
		// 添加其他请求头
	}
	query := &HttpClientOption{}
	bodyGET, statusCodeGET, err := query.SendGETRequest(getURL, getHeaders)
	if err != nil {
		fmt.Printf("GET Request Failed with Status Code: %d, Error: %v\n", statusCodeGET, err)
		return
	}
	fmt.Printf("GET Response with Status Code: %d: %s\n", statusCodeGET, string(bodyGET))

	// 示例用法2: 发起POST请求，Content-Type为application/json
	postJSONURL := "https://example.com/api/post_json_endpoint"
	jsonHeaders := map[string]string{
		"Authorization": "Bearer your_access_token",
		"Other-Header":  "other-value",
		// 添加其他请求头
	}
	requestBody := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	bodyJSON, statusCodeJSON, err := query.SendJSONPOSTRequest(postJSONURL, jsonHeaders, requestBody)
	if err != nil {
		fmt.Printf("POST JSON Request Failed with Status Code: %d, Error: %v\n", statusCodeJSON, err)
		return
	}
	fmt.Printf("POST JSON Response with Status Code: %d: %s\n", statusCodeJSON, string(bodyJSON))

	// 示例用法3: 发起POST请求，Content-Type为application/x-www-form-urlencoded
	postFormURL := "https://example.com/api/post_form_endpoint"
	formHeaders := map[string]string{
		"Authorization": "Bearer your_access_token",
		"Other-Header":  "other-value",
		// 添加其他请求头
	}
	formData := url.Values{
		"key1": []string{"value1"},
		"key2": []string{"value2"},
	}

	bodyForm, statusCodeForm, err := query.SendFormPOSTRequest(postFormURL, formHeaders, formData)
	if err != nil {
		fmt.Printf("POST Form Request Failed with Status Code: %d, Error: %v\n", statusCodeForm, err)
		return
	}
	fmt.Printf("POST Form Response with Status Code: %d: %s\n", statusCodeForm, string(bodyForm))
}
