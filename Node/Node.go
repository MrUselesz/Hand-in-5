package main

import (
	proto "Replication/grpc"
	"context"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	biddersAmount   int32
	highestBidderID int32
	highestBid      int32
	auctionStarted  bool // False on start, gets set to true once when first leader is initialized
	isLeader        bool
	endTime         int64 //Remaining time of the auction in seconds
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

func (s *ReplicationServer) Result(ctx context.Context, req *proto.Empty) (*proto.ShowResult, error) {

	var resultString string

	if time.Now().Unix() < s.endTime {

		//Show current highest bid and highest bidder Id
		resultString = fmt.Sprintf("Current highest bid is %v from bidder %v", s.highestBid, s.highestBidderID)
	} else {
		resultString = fmt.Sprintf("The auction is over! Winner is ID %v with the amount %v", s.highestBidderID, s.highestBid)
	}
	return &proto.ShowResult{Result: resultString, Nodeports: s.NodeAddresses}, nil
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

	var status string
	if time.Now().Unix() >= s.endTime {

		status = fmt.Sprintf("The auction is over")
		log.Println("THE AUCTION ENDS!")
		return &proto.BidAcknowledgement{Acknowledgement: status}, nil

	}
	var bidderId int32
	if !s.isLeader {

		nodeConnect := s.connect(s.leaderAddress)
		response, err := nodeConnect.Bid(context.Background(), req)
		if err != nil {

			fmt.Println("Error forwarding bid to leader")
			status = "Lost connection, please try again"
		}

		return &proto.BidAcknowledgement{Acknowledgement: response.GetAcknowledgement(), Nodeports: response.GetNodeports(), RegisteredId: response.GetRegisteredId()}, nil

	} else {

		//In case of new, unregistered bidder, gives it a new ID
		if req.GetId() == 0 {

			bidderId = s.biddersAmount + 1
			s.biddersAmount++
			fmt.Printf("Unknown bidder, registering new bidder with ID %v %v", bidderId, newLine)

		} else {

			bidderId = req.GetId()

		}

		//Checks if the received bid is the highest and sets it

		if req.GetBid() > s.highestBid {

			fmt.Printf("Old highest bid was: %v from : %v %v", s.highestBid, s.highestBidderID, newLine)

			s.highestBid = req.GetBid()
			s.highestBidderID = bidderId

			fmt.Printf("New highest bid is: %v from : %v %v", s.highestBid, s.highestBidderID, newLine)

			status = "Success!"

			for _, addresses := range s.NodeAddresses {

				nodeConnect := s.connect(addresses)

				_, err := nodeConnect.Update(context.Background(), &proto.PlaceBid{Id: bidderId, Bid: req.GetBid()})
				if err != nil {

					fmt.Println("Error getting updates from leader")

				}

			}

		} else {

			status = "Failure, bid too low. Check results by typing 'result'"

		}

	}

	return &proto.BidAcknowledgement{Acknowledgement: status, Nodeports: s.NodeAddresses, RegisteredId: bidderId}, nil
}

func (s *ReplicationServer) Update(ctx context.Context, req *proto.PlaceBid) (*proto.Empty, error) {

	s.highestBid = req.GetBid()
	s.highestBidderID = req.GetId()
	return &proto.Empty{}, nil

}

// Return its port for the requesting node.
func (s *ReplicationServer) Discover(ctx context.Context, req *proto.Empty) (*proto.Nodes, error) {

	return &proto.Nodes{Port: s.ownAddress, TimeStamp: s.endTime}, nil
}

// Will search through every known port to find living nodes and find its own port.
func (s *ReplicationServer) FindNodesAndOwnPort() {

	var WG sync.WaitGroup

	for i := 0; i < 5; i++ {

		WG.Add(1)

		portSearch := 5000 + i

		go func() {

			defer WG.Done()

			port := strconv.Itoa(portSearch)
			port = ":" + port
			nodeConnect := s.connect(port)
			fmt.Printf("Checking port %v %v", port, newLine)
			response, err := nodeConnect.Discover(context.Background(), &proto.Empty{})
			if err != nil {
				return
			}
			fmt.Printf("Found existing port %v %v", response.GetPort(), newLine)
			s.NodeAddresses = append(s.NodeAddresses, strings.Trim(response.GetPort(), newLine))

			if response.GetTimeStamp() != 0 {
				s.endTime = response.GetTimeStamp()
			}

		}()
	}

	fmt.Println("Searching for port")

	WG.Wait()

	lowestPort := 5006
	for i := 0; i < 5; i++ {
		currentPort := 5000 + i

		if Contains(s.NodeAddresses, ":"+strconv.Itoa(currentPort)) == false {
			if currentPort < lowestPort {
				lowestPort = currentPort
			}
		}

	}
	if lowestPort == 5006 {
		return
	} else {
		s.ownAddress = ":" + strconv.Itoa(lowestPort)
		log.Println("Initialized node with port ", s.ownAddress)
	}

}

func (s *ReplicationServer) InitiateHeartbeat() {

	var WG sync.WaitGroup

	for i := 0; i < len(s.NodeAddresses); i++ {

		WG.Add(1)

		go func() {

			defer WG.Done()

			nodeConnect := s.connect(s.NodeAddresses[i])
			fmt.Println("Checking if " + s.NodeAddresses[i] + " is alive lol")
			_, err := nodeConnect.Heartbeat(context.Background(), &proto.Nodes{Port: s.ownAddress})
			if err != nil {

				if i != len(s.NodeAddresses)-1 {

					s.NodeAddresses = append(s.NodeAddresses[:i], s.NodeAddresses[i+1:]...)

				} else {

					s.NodeAddresses = s.NodeAddresses[:i]
				}

				return
			}

		}()

	}

	WG.Wait()

	if !Contains(s.NodeAddresses, s.leaderAddress) {

		sort.Strings(s.NodeAddresses)

		if !s.auctionStarted {

			s.auctionStarted = true
			s.endTime = time.Now().Add(2 * time.Minute).Unix()

		}

		if len(s.NodeAddresses) == 0 || s.NodeAddresses[0] >= s.ownAddress {
			s.leaderAddress = s.ownAddress
			s.isLeader = true
		} else {
			s.leaderAddress = s.NodeAddresses[0]

		}

	}

}

func (s *ReplicationServer) Heartbeat(ctx context.Context, req *proto.Nodes) (*proto.Empty, error) {

	if !Contains(s.NodeAddresses, strings.Trim(req.GetPort(), newLine)) {

		s.NodeAddresses = append(s.NodeAddresses, req.GetPort())
		sort.Strings(s.NodeAddresses)
		fmt.Println("Appended a node " + req.GetPort() + " to my known addresses")

	}

	return &proto.Empty{}, nil
}

// starts the server.
func (s *ReplicationServer) start_server() {

	s.FindNodesAndOwnPort()
	s.NodeAddresses = append(s.NodeAddresses, s.ownAddress)
	sort.Strings(s.NodeAddresses)
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

	for {
		s.InitiateHeartbeat()

		fmt.Println("Leader is : " + s.leaderAddress)
		time.Sleep(1 * time.Second)

	}

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
