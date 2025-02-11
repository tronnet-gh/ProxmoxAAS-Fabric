package app

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/luthermonson/go-proxmox"
)

const APIVersion string = "0.0.1"

var client ProxmoxClient

func Run() {
	gob.Register(proxmox.Client{})

	configPath := flag.String("config", "config.json", "path to config.json file")
	flag.Parse()

	config := GetConfig(*configPath)
	log.Println("Initialized config from " + *configPath)

	token := fmt.Sprintf(`%s@%s!%s`, config.PVE.Token.USER, config.PVE.Token.REALM, config.PVE.Token.ID)
	client = NewClient(token, config.PVE.Token.Secret)

	//router := gin.Default()

	start := time.Now()
	cluster := Cluster{}
	cluster.Init(client)
	cluster.Rebuild()
	elapsed := time.Since(start)

	fmt.Println(cluster)
	fmt.Println(elapsed)

	/*
		router.GET("/version", func(c *gin.Context) {
			PVEVersion, err := client.Version()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusOK, gin.H{"api-version": APIVersion, "pve-version": PVEVersion})
			}
		})

		router.GET("/nodes/:node", func(c *gin.Context) {
			Node, err := client.Node(c.Param("node"))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusOK, gin.H{"node": Node})
			}
		})

		router.Run("0.0.0.0:" + strconv.Itoa(config.ListenPort))
	*/
}
