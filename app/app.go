package app

import (
	"context"
	"flag"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/luthermonson/go-proxmox"
)

const APIVersion string = "0.0.1"

var client *proxmox.Client = nil

func Run() {
	configPath := flag.String("config", "config.json", "path to config.json file")
	flag.Parse()

	config := GetConfig(*configPath)
	log.Println("Initialized config from " + *configPath)

	client = NewClient(config.PVE.Token.ID, config.PVE.Token.Secret)

	router := gin.Default()

	router.GET("/version", func(c *gin.Context) {
		PVEVersion, err := client.Version(context.Background())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusOK, gin.H{"api-version": APIVersion, "pve-version": PVEVersion})
		}
	})

	router.Run("0.0.0.0:" + strconv.Itoa(config.ListenPort))

}
