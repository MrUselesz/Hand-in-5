package main

import (
	proto "Replication/grpc"
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// all Global values that we need for the node.
type ReplicationServer struct {
	proto.UnimplementedReplicationServer

	ownAddress      string
	leaderAddress   string
	NodeAddresses   []string
	highestBidderID int32
	highestBid      int32
	auctionState    bool //false over true is ongoing
	isLeader        bool
}

var newLine string

func main() {
	if runtime.GOOS == "windows" {
		newLine = "\r\n"
	} else {
		newLine = "\n"
	}

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

// Return its port for the requesting node.
func (s *ReplicationServer) Discover(ctx context.Context, req *proto.Empty) (*proto.Nodes, error) {
	return &proto.Nodes{Port: s.ownAddress}, nil
}

// Will search through every known port to find living nodes and find its own port.
func (s *ReplicationServer) FindNodesAndOwnPort() {
	for i := 0; i < 5; i++ {
		portSearch := 5000 + i

		go func() {
			port := strconv.Itoa(portSearch)
			port = ":" + port
			nodeConnect := s.connect(port)
			fmt.Println(port)
			response, err := nodeConnect.Discover(context.Background(), &proto.Empty{})
			if err != nil {
				return
			}
			fmt.Println("port is ", response.GetPort())
			s.NodeAddresses = append(s.NodeAddresses, strings.Trim(response.GetPort(), newLine))
		}()
	}
	fmt.Println("Searching for port")
	time.Sleep(10 * time.Second)
	lowestPort := 5006
	for i := 0; i < 5; i++ {
		currentPort := 5000 + i

		if Contains(s.NodeAddresses, ":" + strconv.Itoa(currentPort)) == false {
			if currentPort < lowestPort {
				lowestPort = currentPort
			}
		}

	}
	if lowestPort == 5006 {
		return
	} else {
		s.ownAddress = ":" + strconv.Itoa(lowestPort)
	}
	fmt.Println("Your port is ", lowestPort)
	fmt.Println(s.NodeAddresses)

}

// starts the server.
func (s *ReplicationServer) start_server() {

	s.FindNodesAndOwnPort()
	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", s.ownAddress)
	if err != nil {
		log.Fatalf("Failed to listen on port: ", s.ownAddress, err)

	}
	fmt.Println("Server is active")
	proto.RegisterReplicationServer(grpcServer, s)
	go func() {
		err = grpcServer.Serve(listener)

		if err != nil {
			log.Fatalf("Did not work")
		}
	}()

	time.Sleep(1 * time.Hour)

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
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
