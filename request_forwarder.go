package main

import (
	"fmt"
	"net/http"
	"io"
	"os"
	"strconv"
	"time"
	"encoding/base64"

	"github.com/gin-gonic/gin"
	"github.com/galdor/go-cmdline"
)

//

type forwardInfo struct {
	URL    string `header:"X-Request-URL"`
	Method string `header:"X-Request-Method"`
	Key    string `header:"X-Request-Key"`
}

//

func main() {
	router := gin.Default()

	cl := cmdline.New()
	cl.AddFlag("s", "https", "use https")
	cl.AddArgument("port", "port to bind")
	cl.Parse(os.Args)

	//if len(os.Args) < 2 {
	//	fmt.Printf("no port specified\n")
	//	os.Exit(1)
	//}
	//p//ort := os.Args[1]
	port := cl.ArgumentValue("port")
	fmt.Printf("port %#v\n", port)
	portNumber, err := strconv.Atoi(port)
	
	if err != nil || portNumber < 1024 || portNumber > 9999 {
		fmt.Printf("invalid port\n")
		os.Exit(2)
	}


//	currentKey := "XXX"

	v1 := router.Group("/v1")
	{

		// ping
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "pong",
			})
		})

		// forward
		v1.POST("/forward", func(c *gin.Context) {
			h := forwardInfo{}

			if err := c.ShouldBindHeader(&h); err != nil {
				c.JSON(http.StatusOK, err)
			}

			fmt.Printf("%#v\n", h)

			currentKey := makeCurrentKey()

			fmt.Printf("current key: %v\n", currentKey)

			if h.Key != currentKey {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid key",
				})
			} else if !isValidMethod(h.Method) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid method",
				})
			} else if !isValidURL(h.URL) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid url",
				})
			} else {
				body := c.Request.Body
				fmt.Printf("%v\n", body)
			
				header := map[string]string {
					    "Content-Type": "application/json",
					    "Authorization": "123",
    		}
				contentType, respBytes, err := forwardRequest(h.Method, h.URL, header, body)
				if err != nil {
					fmt.Printf("Error Returned: %v\n", err)
					c.JSON(400, gin.H { "error": "1" })
				} else {
					fmt.Printf("Resp body type %v\n", contentType)
					fmt.Printf("Resp body %v\n", string(respBytes))
					// TODO: Check for empty type
					c.Data(http.StatusOK, contentType, respBytes)
				}

 			}
		})
	}

	if cl.IsOptionSet("s") {
		router.RunTLS(":" + port, "server.crt", "server.key")
	} else {
	 	router.Run(":" + port)
 	}
}

//

func isValidURL(url string) bool {
	if len(url) == 0 {
		return false
	} else {
		return true
	}
}

func isValidMethod(method string) bool {
	if method == "GET" || method == "POST" || method == "PUT" { // no DELETE
		return true
	} else {
		return false
	}
}

func makeCurrentKey() string {
	stamp := time.Now().UTC().Format(time.DateOnly)
	validKey := []byte(stamp)
	return base64.RawStdEncoding.Strict().EncodeToString(validKey)
}

func forwardRequest(method string, url string, header map[string]string, body io.ReadCloser) (string, []byte, error) {
    req, err := http.NewRequest(method, url, body)
    if err != nil {
    	fmt.Println("Forward Request Create Failed.")
    	return "", nil, err
    }
    for name, value := range header {
    	req.Header.Set(name, value)
    }   
    
    // Modifictations 
//     req.Header.Add("Content-Type", "application/json")
//     req.Header.Add("Authorization", "123")
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
    	fmt.Printf("Request error %v\n", err)
    	return "", nil, err
    } else {
    	 bytes, err := io.ReadAll(resp.Body)
   		 if err != nil {
    	    return "", nil, err
    	 }
    	contentType := resp.Header.Get("Content-Type") // "" if none
    	fmt.Printf("Recieved body: %v\n", string(bytes))
    	return contentType, bytes, nil
    }
}
