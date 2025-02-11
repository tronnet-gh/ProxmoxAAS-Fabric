package app

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/luthermonson/go-proxmox"
)

const APIVersion string = "0.0.2"

var client ProxmoxClient

func Run() {
	gob.Register(proxmox.Client{})
	gin.SetMode(gin.ReleaseMode)

	configPath := flag.String("config", "config.json", "path to config.json file")
	flag.Parse()

	config := GetConfig(*configPath)
	log.Println("Initialized config from " + *configPath)

	token := fmt.Sprintf(`%s@%s!%s`, config.PVE.Token.USER, config.PVE.Token.REALM, config.PVE.Token.ID)
	client = NewClient(token, config.PVE.Token.Secret)

	router := gin.Default()

	cluster := Cluster{}
	cluster.Init(client)
	start := time.Now()
	log.Printf("Starting cluster sync\n")
	cluster.Sync()
	log.Printf("Synced cluster in %fs\n", time.Since(start).Seconds())

	// set repeating update for full rebuilds
	ticker := time.NewTicker(time.Duration(config.ReloadInterval) * time.Second)
	log.Printf("Initialized cluster sync interval of %ds", config.ReloadInterval)
	channel := make(chan bool)
	go func() {
		for {
			select {
			case <-channel:
				return
			case <-ticker.C:
				start := time.Now()
				log.Printf("Starting cluster sync\n")
				cluster.Sync()
				log.Printf("Synced cluster in %fs\n", time.Since(start).Seconds())
			}
		}
	}()

	router.GET("/version", func(c *gin.Context) {
		PVEVersion, err := client.Version()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusOK, gin.H{"api-version": APIVersion, "pve-version": PVEVersion})
		}
	})

	router.GET("/nodes/:node", func(c *gin.Context) {
		node := c.Param("node")

		host, err := cluster.GetHost(node)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{"node": host})
			return
		}
	})

	router.GET("/nodes/:node/instances/:instance", func(c *gin.Context) {
		node := c.Param("node")
		vmid, err := strconv.ParseUint(c.Param("instance"), 10, 64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("%s could not be converted to vmid (uint)", c.Param("instance"))})
			return
		}

		host, err := cluster.GetHost(node)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("vmid %s not found in cluster", node)})
			return
		} else {
			instance, err := host.GetInstance(uint(vmid))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("%d not found in %s", vmid, node)})
				return
			} else {
				c.JSON(http.StatusOK, gin.H{"instance": instance})
				return
			}
		}
	})

	router.Run("0.0.0.0:" + strconv.Itoa(config.ListenPort))
}
