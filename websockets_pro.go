package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tkanos/gonfig"
)

var clients = make(map[*websocket.Conn]string) // connected clients
var httpclients = make(map[string]string) // connected clients
var allowsTokens = make(map[string]bool)	   // allows clients
var broadcast = make(chan string)            // broadcast channel

// Configure the upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}


func setupResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
    (*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
    (*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func api(w http.ResponseWriter, r *http.Request ) {
	setupResponse(&w,r)
	mes := ""
	r.ParseForm()
	method := strings.Join(r.Form["method"],"")
	switch method {
		case "broadcast": {
			mes = strings.Join(r.Form["message"],"")
			broadcast <- string(mes)
			fmt.Println("broadcast:",mes)			
		}
		case "get_clients":{
			theclients := make(map[string]string);
			for key, value := range clients {
				s := fmt.Sprintf("%s", key.RemoteAddr())
				theclients[s] = value
			}			
			jsonString, err := json.Marshal(theclients)
			if err != nil {
				w.Write([]byte("{\"error\":\"json encoding error\"}"))
			} else {
				w.Write([]byte(jsonString))
			}
			fmt.Println("clients list:",clients)
		}
		case "httpClientsList" : {
			jsonString, err := json.Marshal(httpclients)
			if err != nil {
				w.Write([]byte("{\"error\":\"json encoding error\"}"))
			} else {
				w.Write([]byte(jsonString))
			}
			fmt.Println("http clients list:",httpclients)			
		}
		case "registerToken": {
			t := strings.Join(r.Form["token"],"")
			if len(t) >= 1 { allowsTokens[t] = true; }
		}	
		case "unregisterToken": {
			t := strings.Join(r.Form["token"],"")
			if len(t) >= 1 { allowsTokens[t] = false; }
		}	
		case "registeredTokenList": {
			jsonString, err := json.Marshal(allowsTokens)
			if err != nil {
				w.Write([]byte("{\"error\":\"json encoding error\"}"))
			} else {
				w.Write([]byte(jsonString))
			}
		}
		case "registerHttpClient": {
			setConnection := false
			token := "unregstered"
			rtoken := strings.Join(r.Form["token"],"")
			if len(rtoken) >= 1 {
			   token = rtoken
			} 
			if websocketconfig.AllowUnregisteredHttpClients == "yes" { 
				setConnection = true
			} else { 
				if allowsTokens[token] { 
					setConnection = true
				} 
			}
			if setConnection {
				endpoint := strings.Join(r.Form["endpoint"],"") 
				if len(endpoint) >= 1 {
					httpclients[endpoint] = token
					w.Write([]byte("done"))
				} else {
					w.Write([]byte("error"))
				}
			} else {
				w.Write([]byte("error. unregistered token"))
			}
		}	
	}
}

type saiwebsocketconfig struct {
	Host string
	Port string
	Origin string
	Responseheaders string
	AllowUnregisteredClients string
	AllowUnregisteredHttpClients string
	RegisteredTokensUrl string
	Crtfile string
	Keyfile string
	AutoTLS string
}

func generateSelfSignedCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"saiWebSocket"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	ifaces, _ := net.InterfaceAddrs()
	for _, addr := range ifaces {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				template.IPAddresses = append(template.IPAddresses, ip4)
			}
		}
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return tls.X509KeyPair(certPEM, keyPEM)
}

var websocketconfig saiwebsocketconfig
func main() {
 	config_err := gonfig.GetConf("saiwebsocket.config", &websocketconfig)
	if config_err != nil {
		fmt.Println("Config missed!! ")
		panic(config_err)
	}
	fmt.Println(websocketconfig)
	// Reading preregistered allows tokens
	if len(websocketconfig.RegisteredTokensUrl) > 0 {
		resp, err := http.Get(websocketconfig.RegisteredTokensUrl)
		if err != nil {
			fmt.Println("Corrupted URL. Registered tokens can not be imported")
		} else { 
			
			defer resp.Body.Close() 

			tokensJsonString, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("Corrupted URL data. Registered tokens can not be imported")
			}
			if len(tokensJsonString) > 0 {
				err := json.Unmarshal(tokensJsonString, &allowsTokens)
				if err != nil {
					fmt.Println("Corrupted JSON data. Registered tokens can not be imported \n Url output example {\"abc1\":true,\"abc2\":true,\"abc3\":true,\"group/item1\":true,\"group/item2\":true}")
				}			
			}
		}
	}
	if len(allowsTokens) > 0 {
		fmt.Println("Registered tokens imported ",allowsTokens)
	}
	// Configure routes
	http.HandleFunc("/", api)
	http.HandleFunc("/ws", handleConnections)

	// Start listening for incoming messages
	go handleMessages()

	addr := websocketconfig.Host + ":" + websocketconfig.Port
	fmt.Println("http server started on " + addr)

	if len(websocketconfig.Crtfile) > 0 {
		fmt.Println("Serve wss (cert files)..")
		err := http.ListenAndServeTLS(addr, websocketconfig.Crtfile, websocketconfig.Keyfile, nil)
		if err != nil {
			fmt.Println("ListenAndServeTLS:", err)
		}
	} else if websocketconfig.AutoTLS == "yes" {
		fmt.Println("Serve wss (auto self-signed)..")
		cert, err := generateSelfSignedCert()
		if err != nil {
			panic(err)
		}
		server := &http.Server{
			Addr:      addr,
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
		}
		if err := server.ListenAndServeTLS("", ""); err != nil {
			fmt.Println("ListenAndServeTLS:", err)
		}
	} else {
		fmt.Println("Serve ws")
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			fmt.Println("ListenAndServe:", err)
		}
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	setConnection := false
	token := "unregstered"
	rtoken, ok := r.URL.Query()["RegisterToken"]
	if ok && len(rtoken[0]) >= 1 {
       token = strings.Join(rtoken," ")
    } 
    if websocketconfig.AllowUnregisteredClients == "yes" { 
		setConnection = true
	} else { 
		if allowsTokens[token] { 
			setConnection = true
		} 
	}
	if setConnection {
		// Upgrade initial GET request to a websocket ======================
		upgrader.CheckOrigin = func(r *http.Request) bool {
			if ( websocketconfig.Origin == "*") {return true}
			if ( websocketconfig.Origin == r.URL.String() ) {return true}
			return false
		}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
		}
		defer ws.Close()
		// ==== Upgrade initial GET request to a websocket ======================		
		clients[ws] = token  
		fmt.Println("Register client", token)	
		// loop =====
		for {
			// Read message from browser
			msgType, msg, err := ws.ReadMessage()
			_ = msgType
			if err != nil {
				return
			}
			// Print the message to the console
			fmt.Printf("%s sent: %s\n", ws.RemoteAddr(), string(msg))
			if strings.HasPrefix(string(msg), "RegisterToken:") {
				token := strings.Split(string(msg), ":")
				clients[ws] = token[1];
			} else {
				if strings.HasPrefix(string(msg), "Echo:") {
					echomessage := strings.Split(string(msg), ":")
					ws.WriteJSON(strings.TrimPrefix(strings.Join(echomessage,""), "Echo"))
				} else {
					// Send the newly received message to the broadcast channel
					broadcast <- string(msg)
				}
			}
		}
		// === loop =====			
	} else {
		fmt.Println("Connection refused")
	}
}

func handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		// Send it out to every client that is currently connected
		for client,k := range clients {
			tokens := strings.Split(string(msg), "|")
			if len(tokens) >= 1 {
				if strings.Contains(tokens[0], k) {  // OR tokens[0] == "TokenToBroadcastToAllClients"
					fmt.Println("Now send ", msg, " To:", k)
					//~ err := client.WriteJSON(msg)
					err := client.WriteJSON(strings.TrimPrefix(msg, tokens[0]+"|"))
					time.Sleep(3 * time.Millisecond)
					if err != nil {
						fmt.Println("error: %v", err)
						client.Close()
						delete(clients, client)
					}
				}
			}
		}
		// Send it out to every http client that is currently connected
		for httpclient,k := range httpclients {
			tokens := strings.Split(string(msg), "|")
			if len(tokens) >= 1 {
				if strings.Contains(tokens[0], k) {  // OR tokens[0] == "TokenToBroadcastToAllClients"
					fmt.Println("Now send ", msg, " To HTTP:", k)
					_, err := http.PostForm(httpclient,url.Values{"message": {strings.TrimPrefix(msg, tokens[0]+"|")}})
					time.Sleep(3 * time.Millisecond)
					if err != nil {
						fmt.Println("http cient error: %v", err)
						delete(httpclients, httpclient)
					}
				}
			}
		}
	}
}
