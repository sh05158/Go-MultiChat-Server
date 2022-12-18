/**
 * 20176342 Song Min Joon
 * ChatTCPServer.go
 **/
package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var MAX_USER int = 8

var totalRequests int = 0       // total Request count global variable for server.
var startTime time.Time         // for saving server start time
var serverPort string = "26342" // for server port
var uniqueID int = 1
var clientMap map[int]client

type client struct {
	nickname string
	uniqueID int
	conn     net.Conn
	ip       string
	port     string
}

func main() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM) // for exit the program gracefully
	go func() {
		for sig := range c {
			// sig is a ^C, handle it
			_ = sig
			byebye() // if this program is interrupt by Ctrl-c, print Bye bye and exits gracefully
		}
	}()

	startTime = time.Now() // records server start time for server running time
	listener, _ := net.Listen("tcp", ":"+serverPort)
	fmt.Printf("Server is ready to receive on port %s\n", serverPort)

	clientMap = make(map[int]client) //make client map for record clients

	for {
		//listener is waiting for tcp connection of clients.
		conn, err := listener.Accept()
		if err != nil {
			handleError(conn, err, "server accept error..")
		}

		fmt.Printf("Connection request from %s\n", conn.RemoteAddr().String())

		nickBuffer := make([]byte, 1024)

		count, err := conn.Read(nickBuffer)

		if err != nil || count == 0 {
			continue
			return
		}

		targetNick := string(nickBuffer[:count])

		_, isDuplicate := getClientByNickname(targetNick)

		remoteAddr := conn.RemoteAddr().String()
		lastIdx := strings.LastIndex(remoteAddr, ":")

		newIP := remoteAddr[0:lastIdx]
		newPort := remoteAddr[lastIdx+1:]

		newClient := client{targetNick, uniqueID, conn, newIP, newPort}

		if len(clientMap) >= MAX_USER {
			sendPacket(newClient, "full")
			conn.Close()
			continue
		}

		if isDuplicate {
			sendPacket(newClient, "duplicated")
			conn.Close()
			continue
		}

		sendPacket(newClient, "success")

		registerClient(newClient, uniqueID)

		broadCastToAll(1, fmt.Sprintf("[Welcome %s to CAU network class chat room at %s.]", newClient.nickname, remoteAddr))
		broadCastToAll(1, fmt.Sprintf("[There are %d users connected.]", len(clientMap)))
		go handleMsg(newClient, uniqueID) // when client is connect to server, make go-routine to communicate with client.
		uniqueID++

	}

	defer byebye() // although when client gets panic, defer should disconnect socket gracefully

}

func getClientByNickname(nick string) (client, bool) {
	for _, v := range clientMap {
		if v.nickname == nick {
			return v, true
		}
	}
	return client{}, false
}

func broadCastToAll(route int, msg string) {
	for _, v := range clientMap {
		sendPacket(v, strconv.Itoa(route)+"|"+msg)
	}
	// fmt.Println(msg)
}

func broadCastExceptMe(route int, msg string, client client) {
	for _, v := range clientMap {
		if client.uniqueID != v.uniqueID {
			//do not send to myself
			broadCastStr := strconv.Itoa(route) + "|" + msg
			// fmt.Printf("broadcast msg : %s send to %s\n", broadCastStr, v.nickname)
			sendPacket(v, broadCastStr)
		}
	}
	// fmt.Println(msg)
}

func registerClient(client client, uniqueID int) int {
	clientMap[uniqueID] = client
	return uniqueID
}

func unregisterClient(uniqueID int) {
	if _, ok := clientMap[uniqueID]; ok {
		delete(clientMap, uniqueID)
	}
}

func byebye() {
	fmt.Printf("Bye bye~\n")
	os.Exit(0)
}

func handleError(conn net.Conn, err error, errmsg string) {
	//handle error and print
	if conn != nil {
		conn.Close()
	}
	// fmt.Println(err)
	// fmt.Println(errmsg)
}
func handleError2(conn net.Conn, errmsg string) {
	//handle error and print
	if conn != nil {
		conn.Close()
	}

	// fmt.Println(errmsg)
}

func handleMsg(client client, cid int) {
	for {
		buffer := make([]byte, 1024)

		count, err := client.conn.Read(buffer)

		//when client sends packet

		if err != nil {
			unregisterClient(cid)
			broadCastToAll(6, fmt.Sprintf("%s is disconnected. There are %d users in the chat room", client.nickname, len(clientMap)))
			handleError(client.conn, err, "client disconnected!")
			return
		}

		if count == 0 {
			// fmt.Printf("return ! \n")
			return
		}

		_ = count

		totalRequests++

		/*
			client packet form

			(isCommand | Command Option)

			isCommand => 1   not command( normal chat )
			isCommand => 2    is command


			Command Option => 1 (list) show the nickname IP Port of all connected users
			Command Option => 2 (dm)  dm destination message
			Command Option => 3 (exit) disconnect
			Command Option => 4 (ver) show version
			Command Option => 5 (rtt) show rtt

			2|2|hulk hello there



		*/

		tempStr := string(buffer[:count])

		if strings.Contains(strings.ToUpper(tempStr), "I HATE PROFESSOR") {
			//if client message includes 'i hate professor' disconnect socket
			client.conn.Close()
			unregisterClient(client.uniqueID)
			broadCastToAll(6, fmt.Sprintf("%s is disconnected. There are %d users in the chat room", client.nickname, len(clientMap)))
			sendPacket(client, "5|")
			continue
		}

		fmt.Printf("%s client msg %s\n", client.nickname, tempStr)

		requestOption, _ := strconv.Atoi(strings.Split(tempStr, "|")[0]) // split client packet by '|' and takes option and convert to Integer.

		time.Sleep(time.Millisecond * 1) // minimum delay to deliver packet to client.

		switch requestOption {
		case 1:
			//  is not command (normal message)
			message := strings.Split(tempStr, "|")[1]
			formatMessage := client.nickname + "> " + message

			broadCastExceptMe(0, formatMessage, client)

			break
		case 2:
			//Option 2
			commandOption, _ := strconv.Atoi(strings.Split(tempStr, "|")[1])

			switch commandOption {
			case 1:
				sendPacket(client, "1|"+getClientListString())
				break
			case 2:

				// dmOption := strings.Split(tempStr, "|")[2]
				dmTarget := strings.Split(tempStr, "|")[2]
				dmMessage := strings.Split(tempStr, "|")[3]

				targetClient, success := getClientByNickname(dmTarget)

				if success {
					sendPacket(targetClient, "2|"+client.nickname+"|"+dmMessage)

				}
				break

			case 3:
				sendPacket(client, "3|")
				break
			case 4:
				sendPacket(client, "4|Chat TCP Version 0.1")
				break
			case 5:
				sendPacket(client, "5|")
				break
			}
		default:
			break
			//Option default
			// do nothing
			// conn.Close()
		}
	}

}

func getClientListString() string {
	returnStr := ""

	for _, v := range clientMap {
		returnStr += "\n<" + v.nickname + ", " + v.ip + ", " + v.port + ">"
	}

	fmt.Printf("return Str %s \n", returnStr)
	return returnStr
}

func sendPacket(client client, serverMsg string) {
	//send packet to client
	time.Sleep(time.Millisecond * 1)// minimum delay to deliver packet to client.
	client.conn.Write([]byte(serverMsg))
	fmt.Printf("send packet %s => %s \n",client.nickname,serverMsg)
}
