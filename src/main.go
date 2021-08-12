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
	separator        = "\n= = = = = =\n"
	configContactSep = ";"
)

type appStateShape struct {
	contacts       []string // Contacts list
	currentContact string   // Contact we are currently messaging
}

var (
	correspChan = make(chan string, 1)
	textChan    = make(chan string, 5)
	killChan    = make(chan error, 1)
	rosterChan  = make(chan struct{})

	logger   *log.Logger
	appState = appStateShape{}

	//set by user at start
	username     = ""
	password     = ""
	serverDomain = ""
)

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
		log.Println("Client creation OK")
	}

	if err = client.Connect(); err != nil {
		log.Println("Failed to connect to server. Exiting...")
		return
	}

	//first ation is getting the roster
	getUserRoster(client)

	// start channel router
	go getUserAction()
	initChannelActionManager(client)
}

func handleMessage(_ xmpp.Sender, pkg stanza.Packet) {
	msg, ok := pkg.(stanza.Message)
	if logger != nil {
		m, _ := xml.Marshal(msg)
		logger.Println(string(m)) //output message to console
	}

	if !ok {
		fmt.Printf("%sIgnoring packet: %T\n", separator, pkg)
	}
	//some messages can be errors too
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

func initChannelActionManager(client xmpp.Sender) {
	var text string
	var correspondent string
	fmt.Printf("\nChannel action manager started\n")

	for {
		select {
		//on error or close req, close loop
		case err := <-killChan:
			sc := client.(xmpp.StreamClient)
			sc.Disconnect()
			logger.Println(err)
			return

		//send message
		case text = <-textChan:
			log.Println(separator + "Got a message Request")

			reply := stanza.Message{Attrs: stanza.Attrs{To: correspondent, Type: stanza.MessageTypeChat}, Body: text}
			if logger != nil {
				raw, _ := xml.Marshal(reply)
				logger.Println(string(raw))
			}
			err := client.Send(reply)
			if err != nil {
				fmt.Printf("\nThere was a problem sending the message : %v\n", reply)
				return
			}else{
				log.Println("Message to "+correspondent+" has been sent")
			}

		//set corresponder from chanel
		case crrsp := <-correspChan:
			log.Println(separator + "Now sending messages to : " + crrsp + " in a private conversation")

			correspondent = crrsp

		//get roster req
		case <-rosterChan:
			log.Println( "Roster event recv." )

			getUserRoster(client)
		}

	}
}
func getUserRoster(client xmpp.Sender) {

	log.Println("Attempting to fetch user roster...")

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
	fmt.Printf("Request for user roster sent")

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

		fmt.Printf(separator + "Contacts list updated !")
		return
	}
	fmt.Printf(separator + "Failed to update contact list !")
}

func getUserAction() error {
	reader := bufio.NewReader(os.Stdin)
	printMenu()

	for {
		userOption, _ := reader.ReadString('\n')
		userOption = strings.TrimRight(userOption, "\r\n")

		switch userOption {
		case "1":
			printContactsToWindow()
		case "2":
			{
				//send channel roster update request
				log.Println(separator + "Asking server for contact list...")
				rosterChan <- struct{}{}
			}
		case "3":
			{
				log.Println(separator + "What user do you want to message?")

				targetUser, _ := reader.ReadString('\n')
				targetUser = strings.TrimRight(targetUser, "\r\n")
				appState.currentContact = targetUser
				correspChan <- targetUser // update correspondant
			}
		case "4":
			{
				log.Println("What do you want to say to " + appState.currentContact + "?")

				outgoingMessage, _ := reader.ReadString('\n')
				outgoingMessage = strings.TrimRight(outgoingMessage, "\r\n")

				textChan <- outgoingMessage
			}
		case "5":
			{
				//send kill signal and output kill message, return gorutine
				killChan <- errors.New("user disconect")
				log.Println(separator + "You disconnected from the server.")
				return nil
			}
		case "help":
			printMenu()
		}
	}
}

func printMenu() {
	log.Println("1) View contacts")
	log.Println("2) Refresh/Fetch contacts")
	log.Println("3) Change current contact")
	log.Println("4) Send a message to contact")
	log.Println("5) Exit")
	log.Println("'help') Print this menu")

}
func printContactsToWindow() {
	for _, c := range appState.contacts {
		c = strings.ReplaceAll(c, " *", "")
		if c == appState.currentContact {
			log.Println(c + "*\n")
		} else {
			log.Println(c + "\n")
		}
	}
}
