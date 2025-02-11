package app

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/luthermonson/go-proxmox"
)

const APIVersion string = "0.0.2"

var client ProxmoxClient

func Run() {
	gob.Register(proxmox.Client{})

	configPath := flag.String("config", "config.json", "path to config.json file")
	flag.Parse()

	config := GetConfig(*configPath)
	log.Println("Initialized config from " + *configPath)

	token := fmt.Sprintf(`%s@%s!%s`, config.PVE.Token.USER, config.PVE.Token.REALM, config.PVE.Token.ID)
	client = NewClient(token, config.PVE.Token.Secret)

	router := gin.Default()

	cluster := Cluster{}
	cluster.Init(client)
	cluster.Rebuild()

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
		Host, ok := cluster.Hosts[node]
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("%s not found in cluster", node)})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{"node": Host})
			return
		}
	})

	router.GET("/nodes/:node/instances/:instance", func(c *gin.Context) {
		host := c.Param("node")
		vmid, err := strconv.ParseUint(c.Param("instance"), 10, 64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("%s could not be converted to vmid (uint)", c.Param("instance"))})
			return
		}
		Node, ok := cluster.Hosts[host]
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("vmid %s not found in cluster", host)})
			return
		} else {
			Instance, ok := Node.Instances[uint(vmid)]
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("%d not found in %s", vmid, host)})
				return
			} else {
				c.JSON(http.StatusOK, gin.H{"instance": Instance})
				return
			}
		}
	})

	router.Run("0.0.0.0:" + strconv.Itoa(config.ListenPort))
}
