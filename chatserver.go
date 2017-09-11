package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"sync"
)

var chatRoomServer sync.RWMutex

//Used to return nothing on a RPC call
type Nothing struct{}

type ChatServer struct {
	Clients []Client
	Kill    chan string
}

type Client struct {
	Name     string
	Messages []string
}

//Add a user to the slice of users
func (cs *ChatServer) Register(username string, nothing *Nothing) error {
	chatRoomServer.RLock()
	defer chatRoomServer.RUnlock()

	log.Printf("%s logs on", username)

	//Add to the Clients list before the user gets added to clients so the newly logged on user
	//doesn't recieve the message that he/she has logged on
	for i := 0; i < len(cs.Clients); i++ {
		cs.Clients[i].Messages = append(cs.Clients[i].Messages, username+" has logged on")
	}

	newClient := Client{Name: username}
	cs.Clients = append(cs.Clients, newClient)

	return nil
}

func (cs *ChatServer) List(username string, users *[]string) error {
	log.Printf("%s requests a list of users", username)
	for i := 0; i < len(cs.Clients); i++ {
		*users = append(*users, cs.Clients[i].Name)
	}
	return nil
}

func (cs *ChatServer) CheckMessages(username string, messages *[]string) error {
	chatRoomServer.RLock()
	defer chatRoomServer.RUnlock()

	for i := 0; i < len(cs.Clients); i++ {
		//User found - return messages
		if cs.Clients[i].Name == username {
			*messages = cs.Clients[i].Messages
			//Empty users message queue now that all messages are returned
			cs.Clients[i].Messages = cs.Clients[i].Messages[:0]
		}
	}
	return nil
}

func (cs *ChatServer) Tell(userMessage [3]string, nothing *Nothing) error {
	userFound := false
	for i := 0; i < len(cs.Clients); i++ {
		//User found - append to message queue
		if cs.Clients[i].Name == userMessage[1] {
			userFound = true
			log.Printf("%s tells %s \"%s\"", userMessage[0], userMessage[1], userMessage[2])
			message := userMessage[0] + " tells you: " + userMessage[2]
			cs.Clients[i].Messages = append(cs.Clients[i].Messages, message)
		}
	}
	if !userFound {
		for i := 0; i < len(cs.Clients); i++ {
			if cs.Clients[i].Name == userMessage[0] {
				log.Printf("%s attempts to tell %s \"%s\" but %s is not logged on", userMessage[0], userMessage[1], userMessage[2], userMessage[1])
				message := userMessage[1] + " is not logged on"
				cs.Clients[i].Messages = append(cs.Clients[i].Messages, message)
			}
		}
	}

	return nil
}

func (cs *ChatServer) Say(userMessage [2]string, nothing *Nothing) error {
	chatRoomServer.RLock()
	defer chatRoomServer.RUnlock()
	log.Printf("%s says \"%s\"", userMessage[0], userMessage[1])
	//Add new message to every users message queue
	for i := 0; i < len(cs.Clients); i++ {
		cs.Clients[i].Messages = append(cs.Clients[i].Messages, userMessage[0]+" says: "+userMessage[1])
	}
	return nil
}

func (cs *ChatServer) Logout(username string, nothing *Nothing) error {
	chatRoomServer.RLock()
	defer chatRoomServer.RUnlock()
	d := -1
	for i := 0; i < len(cs.Clients); i++ {
		if cs.Clients[i].Name == username {
			d = i
		}
	}
	if d > -1 {
		log.Printf("%s logs out", username)
		cs.Clients = append(cs.Clients[:d], cs.Clients[d+1:]...)
		for i := 0; i < len(cs.Clients); i++ {
			cs.Clients[i].Messages = append(cs.Clients[i].Messages, username+" has logged off")
		}
	} else {
		log.Println("Error deleting the user")
	}
	return nil
}

//Shut the chat server down
func (cs *ChatServer) Shutdown(userMessage [2]string, nothing *Nothing) error {
	chatRoomServer.RLock()
	defer chatRoomServer.RUnlock()
	//Check to make sure the password sent is right
	if userMessage[1] == "12345" {
		log.Printf("Server is being shutdown by %s", userMessage[0])
		cs.Kill <- "true"
	} else {
		log.Printf("Server shutdown attempted by %s - password incorrect", userMessage[0])
	}
	return nil
}

func main() {
	//Handles command line -port option
	var port string
	flag.StringVar(&port, "port", "3410", "port to listen on")
	flag.Parse()
	log.Printf("Chat server is listening on port %s\n", port)

	server := new(ChatServer)
	server.Kill = make(chan string, 1)

	rpc.Register(server)
	rpc.HandleHTTP()

	l, e := net.Listen("tcp", ":"+port)
	if e != nil {
		log.Fatal("listen error:", e)
	}

	go http.Serve(l, nil)

	<-server.Kill
}
