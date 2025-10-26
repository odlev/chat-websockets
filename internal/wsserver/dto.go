// Package wsserver is a nice package
package wsserver

type wsMessage struct {
	IPAddress string `json:"ip_address"`
	Message   string `json:"message"`
	Time      string `json:"time"`
}
