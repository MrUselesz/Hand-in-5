package main

import (
	proto "Replication/grpc"
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	

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

	connecter = Connect()
	PlaceBid()
	Result()
}
//open or creates a new log file
func openLogFile(path string) (*os.File, error) {
	logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Println("Failed")
	}
	return logFile, nil
}
//Prints the current state of the auction, and who won or who the higest bidder is. 
func Result() {
	result, err := connecter.Result(context.Background(), &proto.Empty{})
	if err != nil {
		log.Fatalf("failed response on: ", err)
	}

	fmt.Println(result.Result)
}
	
//Sends the clients new bid, and prints the outcome.
func PlaceBid() {
	
	bidAcknowledgement, err := connecter.Bid(context.Background(), &proto.PlaceBid{Id: 1, Bid: 100000000})
	if err != nil {
		log.Fatalf("failed response on: ", err)
	}

	fmt.Println(bidAcknowledgement.Acknowledgement, " ", bidAcknowledgement.Nodeports)
}

func Connect() (connection proto.ReplicationClient) {
	conn, err := grpc.NewClient("localhost:5000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Not working")
	}
	
	return proto.NewReplicationClient(conn)
}
