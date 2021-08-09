package main

import (
	"bufio"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gosrc.io/xmpp"
	"gosrc.io/xmpp/stanza"
)

const (
	infoFormat       = "====== "
	configContactSep = ";"
)

type appStateShape struct {
	contacts       []string // Contacts list
	currentContact string   // Contact we are currently messaging
}

var (
	CorrespChan = make(chan string, 1)
	textChan    = make(chan string, 5)
	rawTextChan = make(chan string, 5)
	killChan    = make(chan error, 1)
	rosterChan  = make(chan struct{})

	logger        *log.Logger
	disconnectErr = errors.New("disconnecting client")
	appState      = appStateShape{}

	//set by user at start
	username     = ""
	password     = ""
	serverDomain = ""
)

func buildMessage(message string, author string, channelName string) string {
	return author + " -> " + channelName + " :" + message
}

func main() {

	reader := bufio.NewReader(os.Stdin)
	log.Println("Enter your user: ")
	username, _ = reader.ReadString('\n')
	log.Println("---------------------")
	log.Println("Enter your password: ")
	password, _ = reader.ReadString('\n')
	log.Println("---------------------")
	log.Println("Enter the servers host: (use TLS port)")
	serverDomain, _ = reader.ReadString('\n')
	log.Println("---------------------")
	log.Println("Attempting to contact host...")

	username = strings.TrimRight(username, "\r\n")
	password = strings.TrimRight(password, "\r\n")
	serverDomain = strings.TrimRight(serverDomain, "\r\n")

	config := xmpp.Config{
		TransportConfiguration: xmpp.TransportConfiguration{
			Address: serverDomain,
		},
		Jid:        username,
		Credential: xmpp.Password(password),
		Insecure:   true,
	}

	router := xmpp.NewRouter()
	router.HandleFunc("message", handleMessage)

	client, err := xmpp.NewClient(&config, router, errorHandler)
	if err != nil {
		log.Panicln(fmt.Sprintf("Could not create client, reason: %s", err))
	} else {
		log.Println("Connection OK")
	}

	if err = client.Connect(); err != nil {
		fmt.Println("Failed to connect to server. Exiting...")
		return
	}

	//first ation is getting the roster
	getUserRoster(client)

	// start channel router
	go messageActionRouter(client)
}

func handleMessage(_ xmpp.Sender, pkg stanza.Packet) {
	msg, ok := pkg.(stanza.Message)
	if logger != nil {
		m, _ := xml.Marshal(msg)
		logger.Println(string(m))
	}

	if !ok {
		fmt.Printf("%sIgnoring packet: %T\n", infoFormat, pkg)
	}
	if msg.Error.Code != 0 {
		_, err := fmt.Printf("Error from server : %s : %s \n", msg.Error.Reason, msg.XMLName.Space)
		if err != nil {
			fmt.Printf("Error happened: %s", err)
		}
	}
	if len(strings.TrimSpace(msg.Body)) != 0 {
		_, err := fmt.Printf("%s : %s \n", msg.From, msg.Body)
		if err != nil {
			fmt.Printf("Error happened: %s", err)
		}
	}
}

func errorHandler(err error) {
	killChan <- err
}

func messageActionRouter(client xmpp.Sender) {
	var text string
	var correspondent string
	for {
		select {
		//on error or close req, close loop
		case err := <-killChan:
			if err == disconnectErr {
				sc := client.(xmpp.StreamClient)
				sc.Disconnect()
			} else {
				logger.Println(err)
			}
			return

		//send message
		case text = <-textChan:
			reply := stanza.Message{Attrs: stanza.Attrs{To: correspondent, Type: stanza.MessageTypeChat}, Body: text}
			if logger != nil {
				raw, _ := xml.Marshal(reply)
				logger.Println(string(raw))
			}
			err := client.Send(reply)
			if err != nil {
				fmt.Printf("There was a problem sending the message : %v", reply)
				return
			}

		//raw message
		case text = <-rawTextChan:
			if logger != nil {
				logger.Println(text)
			}
			err := client.SendRaw(text)
			if err != nil {
				fmt.Printf("There was a problem sending the message : %v", text)
				return
			}

		//set corresponder from chanel
		case crrsp := <-CorrespChan:
			correspondent = crrsp

		//get roster req
		case <-rosterChan:
			getUserRoster(client)
		}

	}
}
func getUserRoster(client xmpp.Sender) {
	// Create request
	req, _ := stanza.NewIQ(stanza.Attrs{From: username, Type: stanza.IQTypeGet})
	req.RosterItems()

	if logger != nil {
		m, _ := xml.Marshal(req)
		logger.Println(string(m))
	}
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)

	// Send request
	c, err := client.SendIQ(ctx, req)
	if err != nil {
		logger.Panicln(err)
	}

	// spawn goroutine to update with srvr response to not block client
	go func() {
		serverResp := <-c
		if logger != nil {
			m, _ := xml.Marshal(serverResp)
			logger.Println(string(m))
		}

		// Update contacts
		if rosterItems, ok := serverResp.Payload.(*stanza.RosterItems); ok {
			appState.contacts = []string{}
			for _, item := range rosterItems.Items {
				appState.contacts = append(appState.contacts, item.Jid)
			}

			fmt.Printf(infoFormat + "Contacts list updated !")
			return
		}
		fmt.Printf(infoFormat + "Failed to update contact list !")
	}()
}
