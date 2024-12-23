package main

import (
	proto "Replication/grpc"
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var id int32
var currentConnection string
var ports []string
var connecter proto.ReplicationClient
var newLine string

func main() {

	if runtime.GOOS == "windows" {
		newLine = "\r\n"
	} else {
		newLine = "\n"
	}

	//calls the openLogFile and set all log outpout to be written into the file.
	file, err := openLogFile("../mylog.log")
	if err != nil {
		log.Fatalf("Not working")
	}
	log.SetOutput(file)

	fmt.Print("Starting address is localhost:")
	fmt.Scan(&currentConnection)
	currentConnection = ":" + currentConnection
	ports = append(ports, currentConnection)
	//fmt.Println(ports)

	for {

		var command string
		fmt.Println("Enter your bid, or type 'result' to check the current auction status", newLine)
		fmt.Scan(&command)

		connecter = Connect()

		if connecter == nil {

			fmt.Println("All known servers down")
			log.Fatal("All known servers down")
			break
		}

		if command == "result" {

			Result()

		} else {

			var bid, err = strconv.ParseInt(command, 10, 32)

			if err != nil {

				fmt.Println("You messed something up: %v", err)

			} else {

				PlaceBid(int32(bid))

			}

		}
	}
}

// open or creates a new log file
func openLogFile(path string) (*os.File, error) {
	logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Println("Failed")
	}
	return logFile, nil
}

// Prints the current state of the auction, and who won or who the higest bidder is.
func Result() {
	result, err := connecter.Result(context.Background(), &proto.Empty{})
	if err != nil {
		log.Fatalf("failed response on: %v", err)
	}

	for _, port := range result.GetNodeports() {

		if !Contains(ports, port) {

			ports = append(ports, port)

		}

	}

	fmt.Printf("Reminder that your ID is %v %v", id, newLine)

	fmt.Println(result.Result, ports)
	log.Printf("Client %v asked for auction result and got the message: %v", id, result.GetResult())
}

// Sends the clients new bid, and prints the outcome.
func PlaceBid(enteredBid int32) {

	bidAcknowledgement, err := connecter.Bid(context.Background(), &proto.PlaceBid{Id: id, Bid: enteredBid})
	if err != nil {
		log.Fatalf("failed response on: ", err)
	}

	log.Printf("Client %v made a bid of %v through port %v", id, enteredBid, currentConnection)

	//Sets its new id if it does not have one yet
	if id == 0 {
		id = bidAcknowledgement.GetRegisteredId()
		fmt.Printf("You have been assigned an ID! %v", newLine)
		log.Printf("Registered new node with ID %v %v", id, newLine)
	}

	for _, port := range bidAcknowledgement.GetNodeports() {

		if !Contains(ports, port) {

			ports = append(ports, port)

		}

	}

	fmt.Printf("Reminder that your ID is %v %v", id, newLine)

	fmt.Println(bidAcknowledgement.Acknowledgement)
	log.Printf("Client %v bid resulted in : %v", id, bidAcknowledgement.GetAcknowledgement())
}

func Connect() (connection proto.ReplicationClient) {

	for addressIndex, address := range ports {

		address = "localhost" + address

		fmt.Println("Trying to connect to", address)
		log.Println("Trying to connect to", address)
		conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))

		client := proto.NewReplicationClient(conn)

		_, err = client.Discover(context.Background(), &proto.Empty{})

		if err != nil {

			fmt.Printf("Could not connect to port %v, trying next known port %v", address, newLine)
			log.Printf("Could not connect to port %v, trying next known port %v", address, newLine)

			if addressIndex != len(ports)-1 {

				ports = append(ports[:addressIndex], ports[addressIndex+1:]...)

			} else {

				ports = ports[:addressIndex]
			}

			continue

		}

		fmt.Printf("Client %v Connected to server %v ! %v", id, address, newLine)
		log.Printf("Client %v Connected to server %v ! %v", id, address, newLine)

		return client

	}

	fmt.Println("Could not connect to any ports")
	log.Printf("Client %v could not connect to any port %v", id, newLine)
	return nil

}

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
