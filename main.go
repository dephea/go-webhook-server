package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client

func initConfig() {
	viper.SetConfigFile("config.json")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file %s", err)
	}
	
	viper.AutomaticEnv()
}

func Response(w http.ResponseWriter, r *http.Request) {

	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}

	fmt.Println(ip, r.Method)

	body, err := io.ReadAll(r.Body)
	if err!=nil {
		fmt.Println(err)
		return 
	}

	var parsedBody map[string]interface{}
	err = json.Unmarshal(body, &parsedBody)
	if err!=nil{
		fmt.Println("failed to unmarshal body")
		w.WriteHeader(400)
		w.Write([]byte("Bad Request"))

		var doc = bson.M{
			"timestamp": time.Now(),
			"body": string(body),
		}
		
		docByte, err := json.Marshal(doc)
		if err!=nil {
			fmt.Println("error while marshaling bson doc")
		}

		res, err := WriteToMongo("undefined", docByte)
		if err!=nil {
			fmt.Println("error while writing undefined object to mongo ", err)
		}

		fmt.Printf("Added document to the %s collection. Object ID: %s\n", "undefined", res)
		return
	}

	typeOfWebhook := "noType"

	if val, ok := parsedBody["typeWebhook"].(string); ok {
		typeOfWebhook = val
	}

	defer r.Body.Close()

	w.WriteHeader(200)

	go func(collectionName string, body []byte){
		res, err := WriteToMongo(typeOfWebhook, body)
		if err!=nil {
			fmt.Println("error writing to mongo: ", err)
		}
		fmt.Printf("Added document to the %s collection. Object ID: %s\n", typeOfWebhook, res)
	}(typeOfWebhook, body)

}

func WriteToMongo(collectionName string, object []byte) (*mongo.InsertOneResult, error) {
	collection := Client.Database(viper.GetString("database.name")).Collection(collectionName)

	var bsonBody interface{}

	if err := bson.UnmarshalExtJSON(object, true,&bsonBody); err !=nil {
		return nil, err
	}

	res, err := collection.InsertOne(context.Background(), bsonBody)
	if err!=nil { 
		return nil, err
	}
	return res, nil
}

func init() {
	initConfig()
	Client, _ = mongo.Connect(context.Background(), options.Client().ApplyURI(viper.GetString("database.connectURI")))
}

func main() {
	http.HandleFunc("/webhook", Response)
	fmt.Println("Server is listening on port", viper.GetString("server.port"))
	http.ListenAndServe(fmt.Sprintf(":%s", viper.GetString("server.port")), nil)
}