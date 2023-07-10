package context

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Context struct {
	*gin.Context
	*RequestInfo
	prefix       string
	reqIDEnabled bool
}

type RequestInfo struct {
	RequestID string `json:"requestID"`
	ClientIP  string `json:"clientIP"`
	Host      string `json:"host"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Referer   string `json:"referer"`
	UserAgent string `json:"userAgent"`
}

func New(c *gin.Context) *Context {
	savedCtx, _ := c.Get("Context")
	ctx, ok := savedCtx.(*Context)

	if !ok {
		ctx = &Context{Context: c, RequestInfo: &RequestInfo{}, reqIDEnabled: true}

		if c.Request != nil {
			ctx.RequestInfo = &RequestInfo{
				RequestID: c.Request.Header.Get("X-Request-ID"),
				ClientIP:  c.ClientIP(),
				Host:      c.Request.Host,
				Method:    c.Request.Method,
				Path:      c.Request.URL.Path,
				Referer:   c.Request.Referer(),
				UserAgent: c.Request.UserAgent(),
			}
		}

		if ctx.RequestInfo.RequestID == "" {
			ctx.RequestInfo.RequestID = uuid.New().String()
		}

		c.Set("Context", &ctx)
	}

	return ctx
}

func NewDefault() *Context {
	return New(&gin.Context{})
}

func NewTestContext() *Context {
	return New(&gin.Context{})
}

func (c Context) WithLogPrefix(prefix string) Context {
	c.prefix = prefix
	return c
}

func (c Context) AppendLogPrefix(prefix string) Context {
	c.prefix = c.prefix + ":" + prefix
	return c
}

func (c Context) WithRequestID(enable bool) Context {
	c.reqIDEnabled = enable
	return c
}
