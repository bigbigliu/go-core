package response

const (
	// CodeSuccess 成功响应码
	CodeSuccess = "0"
	// CodeFailure 失败响应码
	CodeFailure = "-1"
)

// ResultInfo http响应消息结构体
type ResultInfo struct {
	Code  string `json:"code"`
	Msg   string `json:"msg"`
	Total int64  `json:"total,omitempty"`
	Data  any    `json:"data"`
}

// OK 构造成功响应
func OK(data any) ResultInfo {
	return ResultInfo{Code: CodeSuccess, Msg: "success", Data: data}
}

// OKWithTotal 构造带总数的成功响应
func OKWithTotal(data any, total int64) ResultInfo {
	return ResultInfo{Code: CodeSuccess, Msg: "success", Total: total, Data: data}
}

// Fail 构造失败响应
func Fail(code, msg string) ResultInfo {
	return ResultInfo{Code: code, Msg: msg}
}

// FailMsg 使用默认失败码构造失败响应
func FailMsg(msg string) ResultInfo {
	return Fail(CodeFailure, msg)
}

// ReqQuery 列表查询参数
type ReqQuery struct {
	Key    string `json:"key"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}
