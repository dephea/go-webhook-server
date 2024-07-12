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
	
	
	
	body, err := io.ReadAll(r.Body)
	if err!=nil {
		fmt.Println(err)
		return 
	}
	
	var parsedBody map[string]interface{}
	err = json.Unmarshal(body, &parsedBody)


	typeOfWebhook := "noType"
	instanceId := "instanceId"

	if val, ok := parsedBody["typeWebhook"].(string); ok {
		typeOfWebhook = val
	}

	if instanceData, ok := parsedBody["instanceData"].(map[string]interface{}); ok {
		if val, ok := instanceData["idInstance"]; ok {
			instanceId = fmt.Sprintf("%.0f", val)
		}
	}
	fmt.Println(ip, r.Method, instanceId)

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

		res, err := WriteToMongo("undefined", "undefined",docByte) 
		if err!=nil {
			fmt.Println("error while writing undefined object to mongo ", err)
		}

		fmt.Printf("Added document to the %s collection. %s\n", "undefined", res)
		return
	}


	defer r.Body.Close()

	w.WriteHeader(200)

	go func(instanceId, typeOfWebhook string, body []byte){
		res, err := WriteToMongo(instanceId, typeOfWebhook, body)
		if err!=nil {
			fmt.Println("error writing to mongo: ", err)
		}
		fmt.Printf("Added document to the %s collection. %s\n", typeOfWebhook, res)
	}(instanceId, typeOfWebhook, body)

}

func WriteToMongo(instanceId, typeOfWebhook string, object []byte) (*mongo.InsertOneResult, error) {
	//collection := Client.Database(viper.GetString("database.name")).Collection(collectionName)
	collection := Client.Database(instanceId).Collection(typeOfWebhook)
	collection2 := Client.Database(instanceId).Collection("allWebhooks")
	var bsonBody interface{}

	if err := bson.UnmarshalExtJSON(object, true,&bsonBody); err !=nil {
		return nil, err
	}

	res, err := collection.InsertOne(context.Background(), bsonBody)
	if err!=nil { 
		return nil, err
	}
	_, err = collection2.InsertOne(context.Background(), bsonBody)
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