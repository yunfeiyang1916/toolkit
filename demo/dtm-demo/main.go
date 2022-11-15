package main

import (
	"log"
	"time"

	"github.com/dtm-labs/dtmcli"
	"github.com/lithammer/shortuuid/v3"

	"github.com/gin-gonic/gin"
)

const (
	Success = "success"
	BusiSrv = "http://localhost:8002"
)

func main() {
	app := gin.New()
	addRoute(app)
	log.Print("开始启动服务")
	go func() {
		if err := app.Run(":8002"); err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(1 * time.Second)

	req := &gin.H{"stock": 1}
	saga := dtmcli.NewSaga("http://localhost:36789/api/dtmsvr", shortuuid.New())
	saga.Add(BusiSrv+"/api/product/decr_stock", BusiSrv+"/api/product/decr_stock_compensate", req)
	saga.Add(BusiSrv+"/api/order/create", BusiSrv+"/api/order/create_compensate", req)
	err := saga.Submit()
	if err != nil {
		log.Printf("submit error,err=%s", err)
		return
	}
	log.Printf("gid=%s", saga.Gid)
	select {}
}

func addRoute(app *gin.Engine) {
	app.POST("/api/product/decr_stock", func(c *gin.Context) {
		log.Print("减少库存")
		c.JSON(200, Success)
	})
	// 扣减库存失败后的补偿接口
	app.POST("/api/product/decr_stock_compensate", func(c *gin.Context) {
		log.Print("扣减库存失败，开始补偿操作")
		c.JSON(200, Success)
	})

	// 创建订单
	app.POST("/api/order/create", func(c *gin.Context) {
		log.Print("创建订单")
		c.JSON(200, Success)
	})
	// 创建订单失败后的补偿接口
	app.POST("/api/order/create_compensate", func(c *gin.Context) {
		log.Print("创建订单失败，开始补偿操作")
		c.JSON(200, Success)
	})
}
