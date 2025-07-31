package gindump

import (
	"bytes"
	"context"
	"io"
	"mime"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/itmisx/logx"
)

func Dump() gin.HandlerFunc {
	return DumpWithOptions(true, true, true, true, true)
}

func DumpWithOptions(showReq bool, showResp bool, showBody bool, showHeaders bool, showCookies bool) gin.HandlerFunc {
	headerHiddenFields := []string{}
	bodyHiddenFields := []string{}

	if !showCookies {
		headerHiddenFields = append(headerHiddenFields, "cookie")
	}

	return func(ctx *gin.Context) {
		var startTs, endTs int64
		var requestHeader interface{}
		var requestBody interface{}
		var responseHeader interface{}
		var responseBody interface{}
		var dumpError string
		startTs = time.Now().UnixMilli()
		if showReq && showHeaders {
			//dump req header
			s, err := FormatToJson(ctx.Request.Header, headerHiddenFields)
			if err != nil {
				dumpError = "parse req header err: " + err.Error()
			} else {
				requestHeader = s
			}
		}

		if showReq && showBody {
			//dump req body
			if ctx.Request.ContentLength > 0 {
				buf, err := io.ReadAll(ctx.Request.Body)
				if err != nil {
					dumpError = "read bodyCache err: " + err.Error()
					goto DumpRes
				}
				rdr := io.NopCloser(bytes.NewBuffer(buf))
				ctx.Request.Body = io.NopCloser(bytes.NewBuffer(buf))
				ctGet := ctx.Request.Header.Get("Content-Type")
				ct, _, err := mime.ParseMediaType(ctGet)
				if err != nil {
					dumpError = "content_type  parse err: " + err.Error()
					goto DumpRes
				}

				switch ct {
				case gin.MIMEPlain:
					bts, err := io.ReadAll(rdr)
					if err != nil {
						dumpError = "read rdr err: " + err.Error()
						goto DumpRes
					}
					requestBody = string(bts)
				case gin.MIMEJSON:
					bts, err := io.ReadAll(rdr)
					if err != nil {
						dumpError = "read rdr err: " + err.Error()
						goto DumpRes
					}

					s, err := FormatJsonBytes(bts, bodyHiddenFields)
					if err != nil {

						dumpError = "parse req body err: " + err.Error()
						goto DumpRes
					}
					requestBody = s
				case gin.MIMEPOSTForm:
					bts, err := io.ReadAll(rdr)
					if err != nil {
						dumpError = "read rdr err: " + err.Error()
						goto DumpRes
					}
					val, err := url.ParseQuery(string(bts))
					if err != nil {
						dumpError = "parse req body err: " + err.Error()
						goto DumpRes
					}
					s, err := FormatToJson(val, bodyHiddenFields)
					if err != nil {
						dumpError = "parse req body err: " + err.Error()
						goto DumpRes
					}
					requestBody = s
				case gin.MIMEMultipartPOSTForm:
				default:
				}
			}

		DumpRes:
			ctx.Writer = &bodyWriter{bodyCache: bytes.NewBufferString(""), ResponseWriter: ctx.Writer}
			ctx.Next()
		}
		endTs = time.Now().UnixMilli()
		if showResp && showHeaders {
			//dump res header
			s, err := FormatToJson(ctx.Writer.Header(), headerHiddenFields)
			if err != nil {
				dumpError = "parse res header err: " + err.Error()
			} else {
				// Response-Header
				responseHeader = s
			}
		}

		if showResp && showBody {
			bw, ok := ctx.Writer.(*bodyWriter)
			if !ok {
				dumpError = "bodyWriter was override , can not read bodyCache"
				goto End
			}

			//dump res body
			if bodyAllowedForStatus(ctx.Writer.Status()) && bw.bodyCache.Len() > 0 {
				ctGet := ctx.Writer.Header().Get("Content-Type")
				ct, _, err := mime.ParseMediaType(ctGet)
				if err != nil {
					dumpError = "content-type parse  err: " + err.Error()
					goto End
				}
				switch ct {
				case gin.MIMEJSON:
					s, err := FormatJsonBytes(bw.bodyCache.Bytes(), bodyHiddenFields)
					if err != nil {
						dumpError = "parse bodyCache err: " + err.Error()
						goto End
					}
					// Reponse body
					responseBody = s
				case gin.MIMEHTML:
				default:
				}
			}
		}

	End:
		var msg string
		var fields []logx.Field
		fields = append(fields, logx.String("request url", ctx.Request.URL.String())) // 请求的url
		fields = append(fields, logx.String("request mothod", ctx.Request.Method))    // 请求方法
		fields = append(fields, logx.Int64("elapsed[ms]", endTs-startTs))             // 耗费时长，衣拉普斯特
		fields = append(fields, logx.Any("request header", requestHeader))            // 请求的header
		fields = append(fields, logx.Any("request body", requestBody))                // 请求的body
		fields = append(fields, logx.Any("response header", responseHeader))          // 响应的header
		fields = append(fields, logx.Any("response body", responseBody))              // 响应的body
		if dumpError != "" {
			fields = append(fields, logx.String("dump error ", dumpError)) // 解析过程中的错误信息
		}
		if (endTs - startTs) > 2000 {
			msg = "slow gin request"
		} else {
			msg = "normal gin request"
		}
		logx.Debug(
			context.Background(),
			msg,
			fields...,
		)
	}
}

type bodyWriter struct {
	gin.ResponseWriter
	bodyCache *bytes.Buffer
}

// rewrite Write()
func (w bodyWriter) Write(b []byte) (int, error) {
	w.bodyCache.Write(b)
	return w.ResponseWriter.Write(b)
}

// bodyAllowedForStatus is a copy of http.bodyAllowedForStatus non-exported function.
func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == http.StatusNoContent:
		return false
	case status == http.StatusNotModified:
		return false
	}
	return true
}
