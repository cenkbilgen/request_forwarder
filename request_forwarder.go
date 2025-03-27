package main
import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/galdor/go-cmdline"
	"github.com/gin-gonic/gin"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type forwardInfo struct {
	URL    string `header:"X-Request-URL"`
	Method string `header:"X-Request-Method"`
	Key    string `header:"X-Request-Key"`
}

type KeyValuesMap map[string]string

type HttpRequestOptions struct {
	Method      string
	URL         string
	Headers     map[string]string
	Body        io.ReadCloser
	Client      *http.Client
}

func main() {
	router := gin.Default()
	cl := cmdline.New()
	cl.AddFlag("s", "https", "use https")
	cl.AddOption("p", "port", "value", "port to bind")
	cl.AddOption("h", "headers-file", "filename", "JSON file containing key ID to value mappings")
	cl.AddOption("k", "validate-key-seed", "seed", "enable request validation with given seed (disabled by default)")
	cl.SetOptionDefault("p", "9100")
	cl.Parse(os.Args)
	
	port := cl.OptionValue("port")
	fmt.Printf("port %#v\n", port)
	portNumber, err := strconv.Atoi(port)
	
	if err != nil || portNumber < 1024 || portNumber > 9999 {
		fmt.Printf("invalid port\n")
		os.Exit(2)
	}

	validateRequests := cl.IsOptionSet("v")
	var validationSeed string
	
	if validateRequests {
		validationSeed = cl.OptionValue("validate")
		if validationSeed == "" {
			fmt.Println("Error: A seed value must be provided with the -v option")
			os.Exit(5)
		}
		fmt.Println("Request key validation is enabled")
	} else {
		fmt.Println("Warning: Request key validation is disabled")
	}

	keyValuesMap := make(KeyValuesMap)

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

	v1 := router.Group("/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "pong",
			})
		})
		
		v1.POST("/forward", func(c *gin.Context) {
			h := forwardInfo{}
			if err := c.ShouldBindHeader(&h); err != nil {
				c.JSON(http.StatusOK, err)
			}
			fmt.Printf("%#v\n", h)
			
			if validateRequests {
				currentKey := makeCurrentKey(validationSeed)
				fmt.Printf("current key: %v\n", currentKey)
				if h.Key != currentKey {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "invalid key",
					})
					return
				}
			}
			
			if !isValidMethod(h.Method) {
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
				
				headers := extractMappedHeaders(c.Request.Header, "X-Request-Key-", keyValuesMap)
				
				if _, ok := headers["Content-Type"]; !ok {
					headers["Content-Type"] = "application/json"
				}
				
				options := HttpRequestOptions{
					Method:  h.Method,
					URL:     h.URL,
					Headers: headers,
					Body:    body,
					Client:  &http.Client{},
				}
				
				contentType, respBytes, err := sendRequest(options)
				if err != nil {
					fmt.Printf("Error Returned: %v\n", err)
					c.JSON(400, gin.H { "error": "1" })
				} else {
					fmt.Printf("Resp body type %v\n", contentType)
					fmt.Printf("Resp body %v\n", string(respBytes))
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

func isValidURL(url string) bool {
	return len(url) > 0
}

func isValidMethod(method string) bool {
	return method == "GET" || method == "POST" || method == "PUT" // no DELETE
}

// CRC-32 is widely available in standard libraries on both iOS and Android
func makeCurrentKey(seed string) string {
	stamp := time.Now().UTC().Format(time.DateOnly)
	input := stamp + seed
	
	hashValue := crc32.ChecksumIEEE([]byte(input))
	
	hashBytes := make([]byte, 4)
	for i := 0; i < 4; i++ {
		hashBytes[i] = byte(hashValue >> (i * 8))
	}
	
	return base64.RawURLEncoding.EncodeToString(hashBytes)
}

func extractMappedHeaders(headers http.Header, prefix string, keyValuesMap KeyValuesMap) map[string]string {
	result := make(map[string]string)
	
	for name, values := range headers {
		if strings.HasPrefix(name, prefix) {
			headerName := strings.TrimPrefix(name, prefix)
			if len(values) > 0 && headerName != "" {
				keyID := values[0]
				if value, ok := keyValuesMap[keyID]; ok {
					result[headerName] = value
					fmt.Printf("Adding header %s: %s from keyID %s\n", headerName, value, keyID)
				}
			}
		}
	}
	
	return result
}

func sendRequest(options HttpRequestOptions) (string, []byte, error) {
    req, err := http.NewRequest(options.Method, options.URL, options.Body)
    if err != nil {
    	fmt.Println("Forward Request Create Failed.")
    	return "", nil, err
    }
    
    for name, value := range options.Headers {
    	req.Header.Set(name, value)
    }   
    
    client := options.Client
    if client == nil {
        client = &http.Client{}
    }
    
    resp, err := client.Do(req)
    if err != nil {
    	fmt.Printf("Request error %v\n", err)
    	return "", nil, err
    } 
    
    defer resp.Body.Close()
    bytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", nil, err
    }
    
    contentType := resp.Header.Get("Content-Type")
    fmt.Printf("Received body: %v\n", string(bytes))
    return contentType, bytes, nil
}
