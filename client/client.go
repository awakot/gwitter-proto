package main

import (
	"fmt"
	"io"
	"log"

	"context"

	"github.com/waytkheming/grpc-go-course/gwitter/gwitterpb"
	"google.golang.org/grpc"
)

func main() {
	fmt.Println("Hello from Gwitter client")
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect:%v", err)
	}

	defer conn.Close()

	c := gwitterpb.NewGweetServiceClient(conn)

	gweet := &gwitterpb.Gweet{
		UserId:  "waytkheming",
		Content: "First Gweet",
	}

	createGweetRes, err := c.PostGwitter(context.Background(), &gwitterpb.PostGwitterRequest{Gweet: gweet})
	if err != nil {
		log.Fatalf("Unexpected Error %v: \n", err)
	}

	fmt.Printf("Gweet has been gweeted: %v \n", createGweetRes)
	gweetID := createGweetRes.GetGweet().GetId()
	// read gwitter
	fmt.Println("Reading the gwitter")
	_, err2 := c.ReadGwitter(context.Background(), &gwitterpb.ReadGwitterRequest{GweetId: "waytkheming"})
	if err2 != nil {
		fmt.Printf("Error happened WHILE READING: %v \n", err2)
	}

	readGweetReq := &gwitterpb.ReadGwitterRequest{GweetId: gweetID}
	readGweetRes, readGweetError := c.ReadGwitter(context.Background(), readGweetReq)
	if readGweetError != nil {
		fmt.Printf("Error happened WHILE READING: %v \n", readGweetError)
	}
	fmt.Printf("Gweet was read: %v \n", readGweetRes)

	newGweet := &gwitterpb.Gweet{
		Id:      gweetID,
		UserId:  "changeMan",
		Content: "Editted content",
	}
	updateRes, updateErr := c.UpdateGwitter(context.Background(), &gwitterpb.UpdateGwitterRequest{Gweet: newGweet})
	if updateErr != nil {
		fmt.Printf("Error happened WHILE updateting: %v \n", readGweetError)
	}
	fmt.Printf("Gweet was updated: %v \n", updateRes)

	fmt.Println("Deleting the gwitter")
	deleteGweetRes, deleteGweetError := c.DeleteGwitter(context.Background(), &gwitterpb.DeleteGwitterRequest{GweetId: gweetID})
	if deleteGweetError != nil {
		fmt.Printf("Error happened WHILE READING: %v \n", deleteGweetError)
	}
	fmt.Printf("Gweet was deleted: %v \n", deleteGweetRes)

	stream, err := c.ListGwitter(context.Background(), &gwitterpb.ListGwitterRequest{})
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("somethig wrong things happened: %v", err)
		}
		fmt.Println(res.GetGweet())
	}

}
