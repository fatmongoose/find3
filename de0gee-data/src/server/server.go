package server

import (
	"fmt"
	"net/http"

	"github.com/de0gee/de0gee-data/src/database"
	"github.com/de0gee/de0gee-data/src/mqtt"
	"github.com/gin-gonic/gin"
)

var Port = "8003"

// Run will start the server listening on the specified port
func Run() {
	mqtt.Setup() // setup MQTT

	r := gin.Default()
	r.HEAD("/", func(c *gin.Context) { // handler for the uptime robot
		c.String(http.StatusOK, "OK")
	})
	r.GET("/ws", wshandler)             // handler for the web sockets (see websockets.go)
	r.POST("/mqtt", handlerMQTT)        // handler for setting MQTT
	r.POST("/data", handlerData)        // typical data handler
	r.POST("/learn", handlerFIND)       // backwards-compatible with FIND
	r.POST("/track", handlerFIND)       // backwards-compatible with FIND
	r.GET("/location", handlerLocation) // get the latest location
	r.Run(":" + Port)                   // listen and serve on 0.0.0.0:8080
}

func handlerMQTT(c *gin.Context) {
	type Payload struct {
		Family string `json:"family" binding:"required"`
		OTP    string `json:"otp" binding:"required"`
	}
	success := false
	var message string
	var p Payload
	if errBind := c.ShouldBindJSON(&p); errBind == nil {
		// authenticate p.OTP
		passphrase, err := mqtt.AddFamily(p.Family)
		if err != nil {
			message = err.Error()
		} else {
			message = fmt.Sprintf("Added '%s' for mqtt. Your passphrase is '%s'", p.Family, passphrase)
		}
	} else {
		message = errBind.Error()
	}
	c.JSON(http.StatusOK, gin.H{"message": message, "success": success})
}

func handlerLocation(c *gin.Context) {
	AddCORS(c)
	type Payload struct {
		Family string `json:"family" binding:"required"`
		Device string `json:"device" binding:"required"`
	}
	success := false
	var message string
	var p Payload
	if errBind := c.ShouldBindJSON(&p); errBind == nil {
		d, err := database.Open(p.Family, true)
		if err != nil {
			message = err.Error()
		} else {
			defer d.Close()
			s, err := d.GetLatest(p.Device)
			if err != nil {
				message = err.Error()
			} else {
				target, err := d.Classify(s)
				if err != nil {
					message = err.Error()
				} else {
					c.JSON(http.StatusOK, gin.H{"message": "got latest", "success": true, "response": target})
					return
				}
			}
		}
	} else {
		message = errBind.Error()
	}
	c.JSON(http.StatusOK, gin.H{"message": message, "success": success})
}

func handlerData(c *gin.Context) {
	AddCORS(c)
	var err error
	var message string
	var d database.SensorData
	err = c.BindJSON(&d)
	if err == nil {
		err2 := processFingerprint(d)
		if err2 == nil {
			message = "inserted data"
		} else {
			err = err2
		}
	}
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": err.Error(), "success": false})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": message, "success": true})
	}
}

func handlerFIND(c *gin.Context) {
	AddCORS(c)
	var j database.FINDFingerprint
	var err error
	var message string
	err = c.BindJSON(&j)
	if err == nil {
		if c.Request.URL.Path == "/track" {
			j.Location = ""
		}
		d := j.Convert()
		err2 := processFingerprint(d)
		if err2 == nil {
			message = "inserted data"
		} else {
			err = err2
		}
	}
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": err.Error(), "success": false})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": message, "success": true})
	}
}

func processFingerprint(d database.SensorData) (err error) {
	err = d.Save()
	if err != nil {
		return
	}

	// TODO: use MQTT to push the latest classification
	return
}
