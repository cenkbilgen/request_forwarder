package main
import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/galdor/go-cmdline"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)
//
type forwardInfo struct {
	URL    string `header:"X-Request-URL"`
	Method string `header:"X-Request-Method"`
	Key    string `header:"X-Request-Key"`
}

// KeyValuesMap stores a mapping of key IDs to their values
type KeyValuesMap map[string]string

//
func main() {
	router := gin.Default()
	cl := cmdline.New()
	cl.AddFlag("s", "https", "use https")
	cl.AddOption("p", "port", "value", "port to bind")
	cl.AddOption("h", "headers-file", "filename", "JSON file containing key ID to value mappings")
	cl.SetOptionDefault("p", "9100")
	cl.Parse(os.Args)
	//if len(os.Args) < 2 {
	//	fmt.Printf("no port specifiedn")
	//	os.Exit(1)
	//}
	//p//ort := os.Args[1]
	port := cl.OptionValue("port")
	fmt.Printf("port %#v\n", port)
	portNumber, err := strconv.Atoi(port)
	
	if err != nil || portNumber < 1024 || portNumber > 9999 {
		fmt.Printf("invalid port\n")
		os.Exit(2)
	}

	// Initialize key values map
	keyValuesMap := make(KeyValuesMap)

	// Read key values from file if provided
	if cl.IsOptionSet("h") {
		headersFile := cl.OptionValue("headers-file")
		file, err := os.Open(headersFile)
		if err != nil {
			fmt.Printf("Failed to open headers file: %v\n", err)
			os.Exit(3)
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&keyValuesMap); err != nil {
			fmt.Printf("Failed to parse headers file: %v\n", err)
			os.Exit(4)
		}

		fmt.Printf("Loaded %d key-value pairs from %s\n", len(keyValuesMap), headersFile)
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
				
				// Extract X-Request-Key-* headers from the request
				headers := make(map[string]string)
				
				// Process X-Request-Key-* headers
				for name, values := range c.Request.Header {
					if strings.HasPrefix(name, "X-Request-Key-") {
						headerName := strings.TrimPrefix(name, "X-Request-Key-")
						// The header value contains the keyID to look up
						if len(values) > 0 && headerName != "" {
							keyID := values[0]
							// Look up the value for this key ID
							if value, ok := keyValuesMap[keyID]; ok {
								headers[headerName] = value
								fmt.Printf("Adding header %s: %s from keyID %s\n", headerName, value, keyID)
							}
						}
					}
				}
				
				// Add default Content-Type if not already set
				if _, ok := headers["Content-Type"]; !ok {
					headers["Content-Type"] = "application/json"
				}
				
				contentType, respBytes, err := forwardRequest(h.Method, h.URL, headers, body)
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
    	fmt.Printf("Received body: %v\n", string(bytes))
    	return contentType, bytes, nil
    }
}
