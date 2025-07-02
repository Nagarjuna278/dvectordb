package api

import (
	"github.com/labstack/echo/v4"
)

func SetApiRoutes(group *echo.Group) {
	group.POST("/put", PostData)
	group.GET("/get", GetData)
	group.GET("/getall", GetAllData)
	group.POST("/dbstore", PushKVData)
}
