package server

import (
	"fmt"
	"github.com/valyala/fastjson"
	"sync"
	"sync/atomic"

	"github.com/valyala/fasthttp"

	"../prefork"
)

var (
	nextResponseId uint64 = 0
	countPool             = sync.Pool{
		New: func() interface{} {
			atomic.AddUint64(&nextResponseId, 1)
			newId := nextResponseId
			return RequestPack{
				ReqId:       newId,
				RecvChannel: make(chan IncomingResponse),
				ModRequest: OutgoingRequest{
					Op:        1,
					RequestId: newId,
				},
			}
		},
	}
)

/*
	startMainServer (public) starts the pre-forking FastHTTP server binding to the
	set address of `mainHost`
*/
func StartMainServer(mainHost string, workerCount int) {
	server := &fasthttp.Server{
		Handler: anyHTTPHandler,
	}

	preforkServer := prefork.New(server, workerCount)

	if !prefork.IsChild() {
		fmt.Printf("Server started server on http://%s\n", mainHost)
	}

	if err := preforkServer.ListenAndServe(mainHost); err != nil {
		panic(err)
	}
}

func anyHTTPHandler(ctx *fasthttp.RequestCtx) {
	reqHelper := countPool.Get().(RequestPack)

	reqHelper.ModRequest.Method = string(ctx.Method())
	reqHelper.ModRequest.Remote = ctx.RemoteAddr().String()
	reqHelper.ModRequest.Path = string(ctx.Path())
	reqHelper.ModRequest.Headers = ctx.Request.Header.String()
	reqHelper.ModRequest.Version = "HTTP/1.1"
	reqHelper.ModRequest.Body = ""
	reqHelper.ModRequest.Query = ctx.QueryArgs().String()

	var shardId uint64
	shardId = 1

	exists := shardManager.SubmitToShard(shardId, &reqHelper.ModRequest, reqHelper.RecvChannel)

	if !exists {
		ctx.SetStatusCode(503)
		_, _ = fmt.Fprintf(ctx, "Internal Server error: Shard with Id: %v does not exist.", shardId)
		return
	}

	response := <-reqHelper.RecvChannel

	countPool.Put(reqHelper)

	ctx.SetStatusCode(response.Status)
	ctx.SetBody(response.Body)

	var head []*fastjson.Value
	for _, head = range response.Headers {
		ctx.Response.Header.SetBytesKV(
			head[0].GetStringBytes(), head[1].GetStringBytes())
	}
}
