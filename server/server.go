package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"os/signal"

	"github.com/waytkheming/grpc-go-course/gwitter/gwitterpb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var collection *mongo.Collection

type server struct {
}

type gwitterItem struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	UserID  string             `bson:"user_id"`
	Content string             `bson:"content"`
}

func (*server) PostGwitter(ctx context.Context, req *gwitterpb.PostGwitterRequest) (*gwitterpb.PostGwitterResponse, error) {
	fmt.Println("Post Gweet invoked")
	gweet := req.GetGweet()

	data := gwitterItem{
		UserID:  gweet.GetUserId(),
		Content: gweet.GetContent(),
	}

	res, err := collection.InsertOne(context.Background(), data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannnot convert to OID	"),
		)
	}

	return &gwitterpb.PostGwitterResponse{
		Gweet: &gwitterpb.Gweet{
			Id:      oid.Hex(),
			UserId:  gweet.GetUserId(),
			Content: gweet.GetContent(),
		},
	}, nil
}

func (*server) ReadGwitter(ctx context.Context, req *gwitterpb.ReadGwitterRequest) (*gwitterpb.ReadGwitterResponse, error) {
	fmt.Println("Read Gweet invoked")
	gweetID := req.GetGweetId()

	oid, err := primitive.ObjectIDFromHex(gweetID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Cannot parse ID"))
	}

	// create empty struct
	data := &gwitterItem{}

	filter := bson.M{"_id": oid}
	res := collection.FindOne(context.Background(), filter)
	if err := res.Decode(data); err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("cannnot fing gweet with this id: %v", err),
		)
	}

	return &gwitterpb.ReadGwitterResponse{
		Gweet: dataToGweetPb(data),
	}, nil
}

func (*server) DeleteGwitter(ctx context.Context, req *gwitterpb.DeleteGwitterRequest) (*gwitterpb.DeleteGwitterResponse, error) {
	fmt.Println("Delete Gweet invoked")
	oid, err := primitive.ObjectIDFromHex(req.GetGweetId())
	if err != nil {
		return nil,
			status.Error(
				codes.InvalidArgument,
				fmt.Sprintf("Cannnot parse your gweet id"))
	}

	filter := bson.M{"_id": oid}
	res, err := collection.DeleteOne(context.Background(), filter)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}

	if res.DeletedCount == 0 {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}

	return &gwitterpb.DeleteGwitterResponse{GweetId: req.GetGweetId()}, nil
}

func (*server) UpdateGwitter(ctx context.Context, req *gwitterpb.UpdateGwitterRequest) (*gwitterpb.UpdateGwitterResponse, error) {
	fmt.Println("Update Gweet invoked")
	gweet := req.GetGweet()
	oid, err := primitive.ObjectIDFromHex(gweet.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Cannot parse ID"))
	}
	// create empty struct
	data := &gwitterItem{}
	filter := bson.M{"_id": oid}

	res := collection.FindOne(context.Background(), filter)
	if err := res.Decode(data); err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("cannnot fing gweet with this id: %v", err),
		)
	}

	data.UserID = gweet.GetUserId()
	data.Content = gweet.GetContent()

	_, updateErr := collection.ReplaceOne(context.Background(), filter, data)
	if updateErr != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot update object in MongoDB: %v", updateErr),
		)
	}

	return &gwitterpb.UpdateGwitterResponse{
		Gweet: dataToGweetPb(data),
	}, nil
}

func (*server) ListGwitter(req *gwitterpb.ListGwitterRequest, stream gwitterpb.GweetService_ListGwitterServer) error {
	fmt.Println("List gwitter request")

	list, err := collection.Find(context.Background(), primitive.D{{}})
	if err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", err),
		)
	}
	defer list.Close(context.Background())

	for list.Next(context.Background()) {
		data := &gwitterItem{}
		err := list.Decode(data)
		if err != nil {
			return status.Errorf(codes.Internal,
				fmt.Sprintf("Error while decoding data from MongoDB: %v", err),
			)
		}
		stream.Send(&gwitterpb.ListGwitterResponse{Gweet: dataToGweetPb(data)})
		if err := list.Err(); err != nil {
			return status.Errorf(codes.Internal,
				fmt.Sprintf("Error while decoding data from MongoDB: %v", err),
			)
		}
	}
	return nil

}

func dataToGweetPb(data *gwitterItem) *gwitterpb.Gweet {
	return &gwitterpb.Gweet{
		Id:      data.ID.Hex(),
		UserId:  data.UserID,
		Content: data.Content,
	}
}

func main() {
	//if crash the code, get the file name and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fmt.Println("hw")
	// client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))

	// connect to database
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Blog Service Started")
	collection = client.Database("mydb").Collection("gwitter")

	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{}
	s := grpc.NewServer(opts...)

	gwitterpb.RegisterGweetServiceServer(s, &server{})
	go func() {
		fmt.Println("Starting server ....")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
	//Wait for Control C to Exit
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	<-ch
	fmt.Println("Stopping the server")
	s.Stop()
	fmt.Println("Close the listener")
	lis.Close()
	fmt.Println("Closeing connection")
	client.Disconnect(context.TODO())
	fmt.Println("end of program")
}
