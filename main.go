package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/unrolled/secure"
	"log"
	"net/http"
	"strings"
	"sync"
)

var LocalPort=":12080"
var sslPort = ":12443"

type Clients struct {
	clientGroup string
	clientName  string
	//Action      map[string]string
	Data     map[string]chan string
	clientWs *websocket.Conn
}

var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var hlClients sync.Map

func NewClient(group string, name string, ws *websocket.Conn) *Clients {
	
	client := &Clients{
		clientGroup: group,
		clientName:  name,
		Data:        make(map[string]chan string, 1),
		//Action:      make(map[string]string),
		clientWs: ws,
	}
	return client

}

func ws(c *gin.Context) {
	getGroup, getName := c.Query("group"), c.Query("name")
	if getGroup == "" || getName == "" {
		return
	}
	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("websocket err:", err)
		return
	}
	client := NewClient(getGroup, getName, ws)
	//message := []byte("hello," + getGroup + "->" + getName)
	//err = ws.WriteMessage(1, message)
	hlClients.Store(getGroup+"->"+getName, client)
	for {
		
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}
		msg := string(message)
		//check:=[]uint8{104,108,65,99,116,105,111,110,58,94,95,94}
		////fmt.Println(msg,string(check)==msg[:12])
		//if len(msg)>12{
		//	if string(check)==msg[:12]{
		//		action:=msg[12:]
		//		//client.Action[action]=""
		//		hlClients.Store(getGroup + "->" + getName, client)
		//	}
		//}else{
		//	fmt.Println(msg)
		//}

		check := []uint8{104, 108, 94, 95, 94}

		strIndex := strings.Index(msg, string(check))
		if strIndex >= 1 {
			action := msg[:strIndex]
			//fmt.Println(action,"save msg")
			if client.Data[action] == nil {
				client.Data[action] = make(chan string, 1)

			}
			client.Data[action] <- msg[strIndex+5:]

			
			hlClients.Store(getGroup+"->"+getName, client)
		} else {
			fmt.Println(msg)
		}
		
	}
	hlClients.Delete(getGroup + "->" + getName)
	defer ws.Close()
}

func QueryFunc(client *Clients, funcName string, param string) {
	var WriteDate string
	if param == "" {
		WriteDate = "{\"action\":\"" + funcName + "\"}"
	} else {
		WriteDate = "{\"action\":\"" + funcName + "\",\"param\":\"" + param + "\"}"
	}
	fmt.Println(WriteDate)
	ws := client.clientWs
	err := ws.WriteMessage(1, []byte(WriteDate))
	if err != nil {
		fmt.Println(err)
	}

}

func Go(c *gin.Context) {
	getGroup, getName, getAction, getParam := c.Query("group"), c.Query("name"), c.Query("action"), c.Query("param")
	if getGroup == "" || getName == "" {
		c.String(200, "input group and name")
		return
	}
	//fmt.Println(getGroup, getName)
	fmt.Println(getGroup + "->" + getName)
	clientName, ok := hlClients.Load(getGroup + "->" + getName)
	fmt.Println(clientName)
	if ok == false {
		c.String(200, "注入了ws？没有找到当前组和名字")
		return
	}
	if getAction == "" {
		c.JSON(200, gin.H{"group": getGroup, "name": getName})
		return
	}

	value, ko := clientName.(*Clients)
	if value.Data[getAction] == nil {
		value.Data[getAction] = make(chan string, 1)
	}



	QueryFunc(value, getAction, getParam)
	//time.Sleep(time.Second)
	//data:=value.Action[getAction]

	data := <-value.Data[getAction]

	if ko {
		c.JSON(200, gin.H{"status": "200", "group": value.clientGroup, "name": value.clientName, getAction: data})
	} else {
		c.JSON(666, gin.H{"message": "?"})
	}

}

func getList(c *gin.Context) {
	resList := "hliang:\r\n"
	hlClients.Range(func(key, value interface{}) bool {
		resList += key.(string) + "\r\n\t"
		
		return true
	})
	c.String(200, resList)
}

func TlsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		secureMiddleware := secure.New(secure.Options{
			SSLRedirect: true,
			SSLHost:     sslPort,
		})
		err := secureMiddleware.Process(c.Writer, c.Request)
		if err != nil {
			c.Abort()
			return
		}
		c.Next()
	}
}

func main() {
	r := gin.Default()
	r.GET("/go", Go)
	r.GET("/ws", ws)
	r.GET("/list", getList)
	r.Use(TlsHandler())
	r.Run(LocalPort)
	// r.RunTLS(sslPort, "zhengshu.pem", "zhengshu.key")

}
