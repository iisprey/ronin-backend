package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"main/pb"
	"net"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"gopkg.in/gomail.v2"
)

var client, ctx, cancel, _ = connectToMongoDB("mongodb://localhost:27017")

func connectGRPC() {
	lis, _ := net.Listen("tcp", "localhost:8080")
	srv := grpc.NewServer()
	pb.RegisterAuthServiceServer(srv, &server{})
	pb.RegisterUserServiceServer(srv, &server{})
	_ = srv.Serve(lis)
}

func main() {
	pingToMongoDB(client, ctx)
	defer closeMongoDB(client, ctx, cancel)
	connectGRPC()
}

type server struct {
	pb.UnimplementedAuthServiceServer
	pb.UnimplementedUserServiceServer
}

func (s *server) Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginRes, error) {
	col := client.Database("ronin").Collection("users")
	result := col.FindOne(ctx, bson.M{"email": req.Email})
	user := &pb.User{}
	result.Decode(user)
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return &pb.LoginRes{Success: false}, nil
	} else {
		return &pb.LoginRes{Success: true}, nil
	}
}
func (s *server) Register(ctx context.Context, req *pb.RegisterReq) (*pb.RegisterRes, error) {
	conn, _ := grpc.Dial("localhost:8080", grpc.WithInsecure())
	defer conn.Close()
	client := pb.NewUserServiceClient(conn)
	password := []byte(req.Password)
	hashedPassword, _ := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	userId, _ := client.CreateUser(ctx, &pb.CreateUserReq{User: &pb.User{
		Email:    req.Email,
		Password: string(hashedPassword),
	}})
	return &pb.RegisterRes{Id: userId.Id}, nil
}
func (s *server) ResetPw(ctx context.Context, req *pb.ResetPwReq) (*pb.ResetPwRes, error) {
	d := gomail.NewDialer("mail.kudamono.app", 587, "payment@kudamono.app", "5rp2eeph3k")
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	m := gomail.NewMessage()
	m.SetHeader("From", "payment@kudamono.app")
	m.SetHeader("To", "hasanbb1567@gmail.com")
	m.SetAddressHeader("Cc", "payment@kudamono.app", "Dan")
	m.SetHeader("Subject", "Hello!")
	m.SetBody("text/html", "test")
	// TODO
	d.DialAndSend(m)
	return &pb.ResetPwRes{Success: true}, nil
}
func (s *server) CreateUser(ctx context.Context, req *pb.CreateUserReq) (*pb.CreateUserRes, error) {
	col := client.Database("ronin").Collection("users")
	result, _ := col.InsertOne(ctx, bson.M{
		"email":    req.User.Email,
		"password": req.User.Password,
	})
	userId := result.InsertedID.(primitive.ObjectID).Hex()
	return &pb.CreateUserRes{Id: userId}, nil
}
func (s *server) ReadUser(ctx context.Context, req *pb.ReadUserReq) (*pb.ReadUserRes, error) {
	userId, _ := primitive.ObjectIDFromHex(req.Id)
	col := client.Database("ronin").Collection("users")
	result := col.FindOne(ctx, bson.M{"_id": userId})
	user := &pb.User{}
	result.Decode(user)
	return &pb.ReadUserRes{User: user}, nil
}
func (s *server) UpdateUser(ctx context.Context, req *pb.UpdateUserReq) (*pb.UpdateUserRes, error) {
	col := client.Database("ronin").Collection("users")
	userId, _ := primitive.ObjectIDFromHex(req.User.Id)
	user, _ := req.User.Marshal()
	col.FindOneAndUpdate(ctx, bson.M{"_id": userId}, bson.M{"$set": user})
	return &pb.UpdateUserRes{Success: true}, nil
}
func (s *server) DeleteUser(ctx context.Context, req *pb.DeleteUserReq) (*pb.DeleteUserRes, error) {
	col := client.Database("ronin").Collection("users")
	userId, _ := primitive.ObjectIDFromHex(req.Id)
	col.FindOneAndDelete(ctx, bson.M{"_id": userId})
	return &pb.DeleteUserRes{Success: true}, nil
}

func (s *server) ListUsers(req *pb.ListUsersReq, stream pb.UserService_ListUsersServer) error {
	col := client.Database("ronin").Collection("users")
	cursor, _ := col.Find(ctx, bson.M{})
	fmt.Println(cursor.Current)
	defer cursor.Close(ctx)
	user := &pb.User{}
	for cursor.Next(ctx) {
		cursor.Decode(user)
		stream.Send(&pb.ListUsersRes{User: user})
	}
	return nil
}

func connectToMongoDB(uri string) (*mongo.Client, context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	return client, ctx, cancel, err
}

func pingToMongoDB(client *mongo.Client, ctx context.Context) error {
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return err
	}
	fmt.Println("connected successfully")
	return nil
}

func closeMongoDB(client *mongo.Client, ctx context.Context, cancel context.CancelFunc) {
	defer cancel()
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
}

// func find(client *mongo.Client, ctx context.Context, db string, col string, query, field interface{}) (result *mongo.Cursor, err error) {
// 	collection := client.Database(db).Collection(col)
// 	result, err = collection.Find(ctx, query, options.Find().SetProjection(field))
// 	return
// }
// func insertMany(client *mongo.Client, ctx context.Context, db string, col string, docs []interface{}) (*mongo.InsertManyResult, error) {
// 	collection := client.Database(db).Collection(col)
// 	result, err := collection.InsertMany(ctx, docs)
// 	return result, err
// }

// func UpdateOne(client *mongo.Client, ctx context.Context, db string, col string, filter, update interface{}) (result *mongo.UpdateResult, err error) {
// 	collection := client.Database(db).Collection(col)
// 	result, err = collection.UpdateOne(ctx, filter, update)
// 	return
// }

// func UpdateMany(client *mongo.Client, ctx context.Context, db string, col string, filter, update interface{}) (result *mongo.UpdateResult, err error) {
// 	collection := client.Database(db).Collection(col)
// 	result, err = collection.UpdateMany(ctx, filter, update)
// 	return
// }

// func deleteOne(client *mongo.Client, ctx context.Context, db string, col string, query interface{}) (result *mongo.DeleteResult, err error) {
// 	collection := client.Database(db).Collection(col)
// 	result, err = collection.DeleteOne(ctx, query)
// 	return
// }

// func deleteMany(client *mongo.Client, ctx context.Context, db string, col string, query interface{}) (result *mongo.DeleteResult, err error) {
// 	collection := client.Database(db).Collection(col)
// 	result, err = collection.DeleteMany(ctx, query)
// 	return
// }
