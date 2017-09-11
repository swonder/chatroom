package main

import (
	"bufio"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strings"
)

type Nothing struct{}

func CheckMessages(client *rpc.Client, user string) {
	var messagesReply []string
	for {
		//Check for messages
		if err := client.Call("ChatServer.CheckMessages", user, &messagesReply); err != nil {
			log.Fatalf("Error calling ChatServer.CheckMessages: %v", err)
		}
		for i := 0; i < len(messagesReply); i++ {
			fmt.Println(messagesReply[i])
			break
		}
	}
}

func ListUsers(client *rpc.Client, user string) {
	var err error
	var listReply []string
	if err = client.Call("ChatServer.List", user, &listReply); err != nil {
		log.Fatalf("Error calling Server.List: %v", err)
	}
	fmt.Println("Users currently logged on")
	for i := 0; i < len(listReply); i++ {
		fmt.Printf("  %s\n", listReply[i])
	}
	fmt.Printf("Total users: %d\n", len(listReply))
}

func main() {
	var nothing Nothing
	var user, address string

	/*if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <serveraddress>", os.Args[0])
	}*/
	if len(os.Args) == 2 {
		user = os.Args[1]
		address = "localhost:3410"
	} else if len(os.Args) == 3 {
		user = os.Args[1]
		address = os.Args[2]
		if strings.HasPrefix(address, ":") {
			address = "localhost" + address
		} else if !strings.Contains(address, ":") {
			address = address + ":3410"
		}
	}	else {
    log.Fatal("Number of command line arguments provided is incorrect")
	}

	//No user specified - set default user to guest
	if len(user) == 0 {
		user = "Guest"
	}

	client, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		log.Fatalf("Error connecting to server at %s: %v", address, err)

	}

	//Register the user, print greeting and list users currently logged on
	if err = client.Call("ChatServer.Register", user, &nothing); err != nil {
		log.Fatalf("Error calling ChatServer.Register: %v", err)
	}
	fmt.Printf("Welcome %s\n", user)
	ListUsers(client, user)

	go CheckMessages(client, user)

	//Start reading lines of text that the user inputs
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		command := scanner.Text()
		//List all the users on the server
		if strings.HasPrefix(command, "list") {
			ListUsers(client, user)
			//Broadcast a message to all users
		} else if strings.HasPrefix(command, "say") {
			var userMessage [2]string
			userMessage[0] = user
			userMessage[1] = command[4:len(command)]
			if err = client.Call("ChatServer.Say", userMessage, &nothing); err != nil {
				log.Fatalf("Error calling ChatServer.Say: %v", err)
			}
			//Send message to a specific user
		} else if strings.HasPrefix(command, "tell") {
			commandTokens := strings.Split(command, " ")
			var serverParams [3]string
			serverParams[0] = user
			//Recipient of message
			serverParams[1] = commandTokens[1]
			//Message
			messageSlice := commandTokens[2:len(commandTokens)]
			serverParams[2] = strings.Join(messageSlice, " ")
			if err = client.Call("ChatServer.Tell", serverParams, &nothing); err != nil {
				log.Fatalf("Error calling ChatServer.Tell: %v", err)
			}
			//Display help
		} else if strings.HasPrefix(command, "help") {
			fmt.Println("--List of chat commands--")
			fmt.Println("  tell <user> some message:  Sends 'some message' to a specific user")
			fmt.Println("  say some other message:    Sends 'some other message' to all users")
			fmt.Println("  list:                      Lists all users currently logged in")
			fmt.Println("  quit:                      Logs you out")
			fmt.Println("  shutdown <password>:       Shuts down the server if supplied password is correct")
			//Logout the current user
		} else if strings.HasPrefix(command, "quit") {
			if err = client.Call("ChatServer.Logout", user, &nothing); err != nil {
				log.Fatalf("Error calling ChatServer.Logout: %v", err)
			}
			fmt.Println("You are now logged out")
			return
			//Shutdown the server if the password is right
		} else if strings.HasPrefix(command, "shutdown") {
			commandTokens := strings.Split(command, " ")
			var serverParams [2]string
			serverParams[0] = user
			serverParams[1] = commandTokens[1]
			if err = client.Call("ChatServer.Shutdown", serverParams, &nothing); err != nil {
				log.Fatalf("Error calling ChatServer.Shutdown: %v", err)
			}
			fmt.Println("Server is now shutdown\n")
		}
	}

	//Error handling
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

}
