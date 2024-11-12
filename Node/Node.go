package main

import (
	proto "Replication/grpc"
	"context"
	"fmt"

	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	
)

// all Global values that we need for the node.
type ReplicationServer struct {
	proto.UnimplementedReplicationServer

	ownAddress string
	leaderAddress string
	NodeAddresses []string
	highestBidderID int32
	highestBid int32
	auctionState bool //false over true is ongoing
	isLeader bool


}

func main() {
	/* here if needed
	 	if runtime.GOOS == "windows" {
			newLine = "\r\n"
		} else {
			newLine = "\n"
		}
	*/

	file, err := openLogFile("../mylog.log")
	if err != nil {
		log.Fatalf("Not working")
	}
	log.SetOutput(file)

	server := &ReplicationServer{}
	server.start_server()

}

// this is the code that responds to talkToHost.
func (s *ReplicationServer) Result(ctx context.Context, req *proto.Empty) (*proto.ShowResult, error) {
	return &proto.ShowResult{Result: "Loser!"}, nil
}

/* check if relavant later.... 
* establishes connection and returns it
*
* @param address - address to which connection is to be established
* @returns a proto client
 */
func (s *ReplicationServer) connect(address string) (connection proto.ReplicationClient) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed connection on: " + address)
	}

	client := proto.NewReplicationClient(conn)

	return client
}




func (s *ReplicationServer) Bid(ctx context.Context, req *proto.PlaceBid) (*proto.BidAcknowledgement, error) {

	var test [3]string
	test[0] = "1"
	test[1] = "2"
	test[2] = "3"
	return &proto.BidAcknowledgement{Acknowledgement: "Nemt", Nodeports: test[:]}, nil
}


// starts the server.
func (s *ReplicationServer) start_server() {

	if os.Args[0] == "host" {

	}
	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":5000")
	if err != nil {
		log.Fatalf("Failed to listen on port: ", ":5000", err)

	}
	fmt.Println("Server is active")
	proto.RegisterReplicationServer(grpcServer, s)
	//go func() {
		err = grpcServer.Serve(listener)

		if err != nil {
			log.Fatalf("Did not work")
		}
	//}()

}


// this open the log
func openLogFile(path string) (*os.File, error) {
	logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Println("Log failed")
	}
	return logFile, nil
}

// simple util method for wheter a slice contains a
func Contains(s []int32, e int32) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
