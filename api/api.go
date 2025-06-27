package api

import (
	"dvecdb/kvstore"
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

func PostData(e echo.Context) error {
	if err := e.Bind(&dataSingle); err != nil {
		log.Printf("PostData handler: Error binding request body: %v", err)
		// Return a Bad Request (400) status if the binding fails
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body. Please ensure 'key' and 'value' are provided and correctly formatted.",
		})
	}
	fmt.Println(dataSingle)
	err := kvstore.KV.Put(dataSingle.Key, dataSingle.Value)
	if err != nil {
		return err
	}
	return nil
}

func GetData(e echo.Context) error {
	err := e.Bind(&dataSingle)
	fmt.Println(dataSingle.Key)
	value, err := kvstore.KV.Get(dataSingle.Key)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, map[string]string{"key": dataSingle.Key, "value": value})
}

func GetAllData(c echo.Context) error {
	data, err := kvstore.KV.GetAll()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, data)
}
