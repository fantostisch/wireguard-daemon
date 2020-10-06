package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type UserHandler struct {
	Server *Server
}

// Get all configs of a user.
func (h UserHandler) getConfigs(w http.ResponseWriter, username string) {
	clients := map[string]*ClientConfig{}

	h.Server.mutex.Lock()
	defer h.Server.mutex.Unlock()

	userConfig := h.Server.Config.Users[username]
	if userConfig != nil {
		clients = userConfig.Clients
	}

	if err := json.NewEncoder(w).Encode(clients); err != nil {
		message := fmt.Sprintf("Error encoding response as JSON: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}
}

type createConfigAndKeyPairResponse struct {
	ClientPrivateKey string `json:"clientPrivateKey"`
	IP               net.IP `json:"ip"`
	ServerPublicKey  string `json:"serverPublicKey"`
}

type createConfigResponse struct {
	IP              net.IP `json:"ip"`
	ServerPublicKey string `json:"serverPublicKey"`
}

func (h UserHandler) newConfig(username string, publicKey string, name string) createConfigResponse {
	h.Server.mutex.Lock()
	defer h.Server.mutex.Unlock()

	userConfig := h.Server.Config.GetUserConfig(username)

	ip := h.Server.allocateIP()
	config := NewClientConfig(name, ip)

	userConfig.Clients[publicKey] = &config

	return createConfigResponse{
		IP:              config.IP,
		ServerPublicKey: h.Server.Config.PublicKey,
	}
}

func (h UserHandler) createConfigGenerateKeyPair(w http.ResponseWriter, username string, name string) {
	clientPrivateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		message := fmt.Sprintf("Error generating private key: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}
	clientPublicKey := clientPrivateKey.PublicKey()
	createConfigResponse := h.newConfig(username, clientPublicKey.String(), name)
	response := createConfigAndKeyPairResponse{
		ClientPrivateKey: clientPrivateKey.String(),
		IP:               createConfigResponse.IP,
		ServerPublicKey:  createConfigResponse.ServerPublicKey,
	}

	if err := h.Server.reconfigureWG(); err != nil {
		message := fmt.Sprintf("Error reconfiguring WireGuard: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		message := fmt.Sprintf("Error encoding response as JSON: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}
}

func (h UserHandler) createConfig(w http.ResponseWriter, username string, publicKey string, name string) {
	response := h.newConfig(username, publicKey, name)

	if err := h.Server.reconfigureWG(); err != nil {
		message := fmt.Sprintf("Error reconfiguring WireGuard: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		message := fmt.Sprintf("Error encoding response as JSON: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h UserHandler) deleteConfig(w http.ResponseWriter, username string, publicKey string) {
	h.Server.mutex.Lock()
	defer h.Server.mutex.Unlock()
	userConfig := h.Server.Config.Users[username]
	if userConfig == nil {
		http.Error(w, fmt.Sprintf("User '%s' not found", username), http.StatusNotFound)
		return
	}

	if userConfig.Clients[publicKey] == nil {
		message := fmt.Sprintf("Config with public key '%s' not found for user '%s", publicKey, username)
		http.Error(w, message, http.StatusNotFound)
		return
	}

	delete(userConfig.Clients, publicKey)

	if err := h.Server.reconfigureWG(); err != nil {
		message := fmt.Sprintf("Error reconfiguring WireGuard: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

const form = "application/x-www-form-urlencoded"

func (h UserHandler) ServeHTTP(w http.ResponseWriter, req *http.Request, username string) {
	receivedPublicKey := req.Form.Get("public_key")
	if receivedPublicKey == "" {
		switch req.Method {
		case http.MethodGet:
			h.getConfigs(w, username)
		case http.MethodPost:
			name := req.Form.Get("name")
			if name == "" {
				contentType := req.Header.Get("Content-Type")
				if contentType != form {
					http.Error(w, fmt.Sprintf("Content-Type '%s' was not equal to %s'", contentType, form), http.StatusBadRequest)
					return
				}
				http.Error(w, "No config name supplied.", http.StatusBadRequest)
				return
			}
			h.createConfigGenerateKeyPair(w, username, name)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	} else {
		publicKey, err := wgtypes.ParseKey(receivedPublicKey)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid public key: '%s'. %s", receivedPublicKey, err), http.StatusBadRequest)
			return
		}
		switch req.Method {
		case http.MethodPost:
			//todo: same code as above
			name := req.Form.Get("name")
			if name == "" {
				contentType := req.Header.Get("Content-Type")
				if contentType != form {
					http.Error(w, fmt.Sprintf("Content-Type '%s' was not equal to %s'", contentType, form), http.StatusBadRequest)
					return
				}
				http.Error(w, "No config name supplied.", http.StatusBadRequest)
				return
			}
			h.createConfig(w, username, publicKey.String(), name)
		case http.MethodDelete:
			h.deleteConfig(w, username, publicKey.String())
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
