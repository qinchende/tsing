package tsing

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// 连接上下文
type Context struct {
	app     *App
	Request *http.Request
	// responseWriter customResponseWriter  // 自定义response
	// ResponseWriter _CustomResponseWriter // 用自定义resp替代http.resp
	ResponseWriter http.ResponseWriter
	routerParams   httprouter.Params // 路由参数
	next           bool              // 继续执行下一个中间件或处理器
	parsed         bool              // 是否已解析body
}

// 继续执行下一个中间件或处理器
func (ctx *Context) Next() {
	ctx.next = true
}

// 在ctx里存储值，如果key存在则替换值
func (ctx *Context) SetValue(key string, value interface{}) {
	ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), key, value))
}

// 获取ctx里的值，取出后根据写入的类型自行断言
func (ctx *Context) GetValue(key string) interface{} {
	return ctx.Request.Context().Value(key)
}

// 向客户端发送重定向响应
func (ctx *Context) Redirect(code int, url string) error {
	if code < 300 || code > 308 {
		return errors.New("状态码只能是300-308之间的值")
	}
	ctx.ResponseWriter.Header().Set("Location", url)
	ctx.ResponseWriter.WriteHeader(code)
	return nil
}

// 透过nginx反向代理获得客户端真实IP
func (ctx *Context) RemoteIP() string {
	ra := ctx.Request.RemoteAddr
	if ip := ctx.Request.Header.Get("X-Forwarded-For"); ip != "" {
		ra = strings.Split(ip, ", ")[0]
	} else if ip := ctx.Request.Header.Get("X-Real-IP"); ip != "" {
		ra = ip
	} else {
		var err error
		ra, _, err = net.SplitHostPort(ra)
		if err != nil {
			return ""
		}
	}
	return ra
}

// 解析body数据
func (ctx *Context) parseBody() error {
	// 判断是否已经解析过body
	if ctx.parsed {
		return nil
	}
	if strings.HasPrefix(ctx.Request.Header.Get("Content-Type"), "multipart/form-data") {
		if err := ctx.Request.ParseMultipartForm(http.DefaultMaxHeaderBytes); err != nil {
			return err
		}
	} else {
		if err := ctx.Request.ParseForm(); err != nil {
			return err
		}
	}
	// 标记该context中的body已经解析过
	ctx.parsed = true
	return nil
}

// 获取所有路由参数值
func (ctx *Context) RouteValues() []httprouter.Param {
	return ctx.routerParams
}

// 获取路由参数值
func (ctx *Context) Param(key string) (string, bool) {
	for i := range ctx.routerParams {
		if ctx.routerParams[i].Key == key {
			return ctx.routerParams[i].Value, false
		}
	}
	return "", true
}

// 获取某个路由参数值的string类型
func (ctx *Context) ParamValue(key string) string {
	return ctx.routerParams.ByName(key)
}

// 获取所有GET参数值
func (ctx *Context) Querys() url.Values {
	return ctx.Request.URL.Query()
}

// 获取某个GET参数值
func (ctx *Context) Query(key string) (string, bool) {
	if len(ctx.Request.URL.Query()[key]) == 0 {
		return "", false
	}
	return ctx.Request.URL.Query()[key][0], true
}

// 获取某个GET参数值的string类型
func (ctx *Context) QueryValue(key string) string {
	if len(ctx.Request.URL.Query()[key]) == 0 {
		return ""
	}
	return ctx.Request.URL.Query()[key][0]
}

// 获取所有POST参数值
func (ctx *Context) Posts() url.Values {
	err := ctx.parseBody()
	if err != nil {
		return url.Values{}
	}
	return ctx.Request.PostForm
}

// 获取某个POST参数值
func (ctx *Context) Post(key string) (string, bool) {
	if err := ctx.parseBody(); err != nil {
		return "", false
	}
	vs := ctx.Request.Form[key]
	if len(vs) == 0 {
		return "", false
	}
	return ctx.Request.Form[key][0], true
}

// 获取某个POST参数值的string类型
func (ctx *Context) PostValue(key string) string {
	if err := ctx.parseBody(); err != nil {
		return ""
	}
	vs := ctx.Request.Form[key]
	if len(vs) == 0 {
		return ""
	}
	return ctx.Request.Form[key][0]
}
