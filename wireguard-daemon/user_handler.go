package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/skip2/go-qrcode"
)

type UserHandler struct {
	Server *Server
}

// Get all clients of a user.
func (h UserHandler) getClients(w http.ResponseWriter, username string) {
	log.Print("Getting Clients")
	clients := map[string]*ClientConfig{}
	userConfig := h.Server.Config.Users[username]
	if userConfig != nil {
		clients = userConfig.Clients
	} else {
		log.Print("This user does not have clients")
	}

	err := json.NewEncoder(w).Encode(clients)

	w.WriteHeader(http.StatusOK)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h UserHandler) getClient(w http.ResponseWriter, username string, id int) {
	log.Print("Get One client")

	usercfg := h.Server.Config.Users[username]
	if usercfg == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	log.Print("Client :::")
	client := usercfg.Clients[strconv.Itoa(id)]
	if client == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	log.Print("AllowedIP's Config")
	allowedIPs := *wgAllowedIPs + ","

	dns := ""
	if *wgDNS != "" {
		dns = fmt.Sprint("DNS = ", *wgDNS)
	}

	configData := fmt.Sprintf(`[Interface]
%s
Address = %s
PrivateKey = %s
[Peer]
PublicKey = %s
AllowedIPs = %s
Endpoint = %s
`, dns, client.IP.String(), client.PrivateKey, h.Server.Config.PublicKey, allowedIPs, *wgEndpoint)
	log.Print(configData)
	format := "config" //todo

	if format == "qrcode" {
		png, err := qrcode.Encode(configData, qrcode.Medium, 220)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(png)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		return
	}

	if format == "config" {
		filename := fmt.Sprintf("%s.conf", filenameRe.ReplaceAllString(client.Name, "_"))
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		w.Header().Set("Content-Type", "application/config")
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, configData)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	err := json.NewEncoder(w).Encode(client)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

//----------API-Endpoint: Creating client----------
func (h UserHandler) createClient(w http.ResponseWriter, r *http.Request, username string) {
	h.Server.mutex.Lock()
	defer h.Server.mutex.Unlock()

	log.Print("Creating client :: User ", username)
	cli := h.Server.Config.GetUserConfig(username)
	log.Print("User Config: ", cli.Clients)

	//if maxNumberCliConfig > 3 {
	//	if len(cli.Clients) >= maxNumberCliConfig {
	//		log.Errorf("there too many configs ", cli.Name)
	//		e := struct {
	//			Error string
	//		}{
	//			Error: "Max number of configs: " + strconv.Itoa(maxNumberCliConfig),
	//		}
	//		w.WriteHeader(http.StatusBadRequest)
	//		err := json.NewEncoder(w).Encode(e)
	//		if err != nil {
	//			log.Errorf("There was an API ERRROR - CREATE CLIENT ::", err)
	//			w.WriteHeader(http.StatusBadRequest)
	//			err := json.NewEncoder(w).Encode(e)
	//			if err != nil {
	//				log.Errorf("Error enocoding ::", err)
	//				return
	//			}
	//			return
	//		}
	//		log.Print("decoding dthe body")
	decoder := json.NewDecoder(r.Body)
	//w.Header().Set("Content-Type", "application/json"; "charset=UTF-8")
	client := &ClientConfig{}
	err := decoder.Decode(&client)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if client.Name == "" {
		log.Print("No CLIENT NAME found.....USING DEFAULT...\"unnamed Client\"")
		client.Name = "Unnamed Client"
	}
	i := 0
	for k := range cli.Clients {
		n, err := strconv.Atoi(k)
		if err != nil {
			log.Print("THere was an error strc CONV :: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if n > i {
			i = n
		}
	}
	i++
	log.Print("Allocating IP")
	ip := h.Server.allocateIP()
	log.Print("Creating Client Config")
	client = NewClientConfig(ip, client.Name, client.Info)
	cli.Clients[strconv.Itoa(i)] = client
	err = h.Server.reconfiguringWG()
	if err != nil {
		log.Print("error Reconfiguring :: ", err)
	}
	err = json.NewEncoder(w).Encode(client)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h UserHandler) editClient(w http.ResponseWriter, req *http.Request, username string, clientID int) {
	usercfg := h.Server.Config.Users[username]
	if usercfg == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	client := usercfg.Clients[strconv.Itoa(clientID)]
	if client == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	cfg := ClientConfig{}

	if err := json.NewDecoder(req.Body).Decode(&cfg); err != nil {
		log.Print("Error parsing request: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Printf("EditClient: %#v", cfg)

	if cfg.Name != "" {
		client.Name = cfg.Name
	}

	if cfg.Info != "" {
		client.Info = cfg.Info
	}

	client.Modified = time.Now().Format(time.RFC3339)

	reconfigureErr := h.Server.reconfiguringWG()
	if reconfigureErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(client); err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h UserHandler) deleteClient(w http.ResponseWriter, username string, clientID int) {
	usercfg := h.Server.Config.Users[username]
	if usercfg == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	client := strconv.Itoa(clientID)
	if usercfg.Clients[client] == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	delete(usercfg.Clients, client)
	reconfigureErr := h.Server.reconfiguringWG()
	if reconfigureErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Print("user", username, " ::: Deleted client:", client)

	w.WriteHeader(http.StatusOK)
}

func (h UserHandler) ServeHTTP(w http.ResponseWriter, req *http.Request, username string) {
	clients, secondRemaining := ShiftPath(req.URL.Path)
	clientID, _ := ShiftPath(secondRemaining)
	if clients == "clients" {
		if clientID == "" {
			switch req.Method {
			case http.MethodGet:
				h.getClients(w, username)
			case http.MethodPost:
				h.createClient(w, req, username)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else {
			id, err := strconv.Atoi(clientID)
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid user id: '%s'", clientID), http.StatusBadRequest)
				return
			}
			switch req.Method {
			case http.MethodGet:
				h.getClient(w, username, id)
			case http.MethodPut:
				h.editClient(w, req, username, id)
			case http.MethodPost:
				h.deleteClient(w, username, id)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		}
	} else {
		http.NotFound(w, req)
	}
}