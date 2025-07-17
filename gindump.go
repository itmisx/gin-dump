package gindump

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
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
		var strB strings.Builder
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
				dumpError = "parse req header err " + err.Error()
			} else {
				requestHeader = s
			}
		}

		if showReq && showBody {
			//dump req body
			if ctx.Request.ContentLength > 0 {
				buf, err := io.ReadAll(ctx.Request.Body)
				if err != nil {
					strB.WriteString(fmt.Sprintf("\nread bodyCache err \n %s", err.Error()))
					goto DumpRes
				}
				rdr := io.NopCloser(bytes.NewBuffer(buf))
				ctx.Request.Body = io.NopCloser(bytes.NewBuffer(buf))
				ctGet := ctx.Request.Header.Get("Content-Type")
				ct, _, err := mime.ParseMediaType(ctGet)
				if err != nil {
					strB.WriteString(fmt.Sprintf("\ncontent_type: %s parse err \n %s", ctGet, err.Error()))
					goto DumpRes
				}

				switch ct {
				case gin.MIMEJSON:
					bts, err := io.ReadAll(rdr)
					if err != nil {
						strB.WriteString(fmt.Sprintf("\nread rdr err \n %s", err.Error()))
						goto DumpRes
					}

					s, err := FormatJsonBytes(bts, bodyHiddenFields)
					if err != nil {

						dumpError = "parse req body err " + err.Error()
						goto DumpRes
					}
					requestBody = s
				case gin.MIMEPOSTForm:
					bts, err := io.ReadAll(rdr)
					if err != nil {
						dumpError = "read rdr err " + err.Error()
						goto DumpRes
					}
					val, err := url.ParseQuery(string(bts))
					if err != nil {
						dumpError = "parse req body err" + err.Error()
						goto DumpRes
					}
					s, err := FormatToJson(val, bodyHiddenFields)
					if err != nil {
						dumpError = "parse req body err" + err.Error()
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
				dumpError = "parse res header err " + err.Error()
			} else {
				// Response-Header
				responseHeader = s
			}
		}

		if showResp && showBody {
			bw, ok := ctx.Writer.(*bodyWriter)
			if !ok {
				strB.WriteString("\nbodyWriter was override , can not read bodyCache")
				goto End
			}

			//dump res body
			if bodyAllowedForStatus(ctx.Writer.Status()) && bw.bodyCache.Len() > 0 {
				ctGet := ctx.Writer.Header().Get("Content-Type")
				ct, _, err := mime.ParseMediaType(ctGet)
				if err != nil {
					strB.WriteString(fmt.Sprintf("\ncontent-type: %s parse  err \n %s", ctGet, err.Error()))
					goto End
				}
				switch ct {
				case gin.MIMEJSON:

					s, err := FormatJsonBytes(bw.bodyCache.Bytes(), bodyHiddenFields)
					if err != nil {
						dumpError = "parse bodyCache err " + err.Error()
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
		fields = append(fields, logx.String("request url", ctx.Request.URL.String()))
		fields = append(fields, logx.String("request mothod", ctx.Request.Method))
		fields = append(fields, logx.Int64("requst duration[ms]", endTs-startTs))
		fields = append(fields, logx.Any("request header", requestHeader))
		fields = append(fields, logx.Any("request body", requestBody))
		fields = append(fields, logx.Any("response header", responseHeader))
		fields = append(fields, logx.Any("response body", responseBody))
		fields = append(fields, logx.String("dump-error", dumpError))
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
