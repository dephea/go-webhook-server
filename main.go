package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func initConfig() {
	viper.SetConfigFile("config.json")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file %s", err)
	}
	
	viper.AutomaticEnv()
}

// []string{
// 	"127.0.0.1",
// 	"10.240.4.3",
// 	"51.250.84.44",
// 	"51.250.94.65",  
// 	"51.250.89.177",  
// 	"64.226.125.75",  
// 	"158.160.49.84",  
// 	"46.101.109.139",  
// 	"51.250.95.149",  
// 	"51.250.12.167",  
// 	"51.250.78.88", 
// 	"64.227.117.254",   
// 	"64.226.125.75",   
// 	"159.89.1.39",    
// 	"164.92.234.244",   
// 	"64.226.84.16",   
// 	"164.92.255.106",  
// 	"178.128.207.139",   
// 	"104.248.26.103",   
// 	"157.230.23.121",   
// 	"161.35.207.158",   
// 	"167.172.162.71" ,  
// }

var client, _ = mongo.Connect(context.Background(), options.Client().ApplyURI(viper.GetString("database.connectURI")))


func getIP(r *http.Request) string {
	// ip := r.Header.Get("X-Forwarded-For")
	// fmt.Println("X-Forwarded-For: ", r.Header.Get("X-Forwarded-For"))
	// if ip == "" {
	// 	ip = r.RemoteAddr
	// }

	ip := r.RemoteAddr

	if strings.Contains(ip, ":") {
		ip = strings.Split(ip, ":")[0]
	}

	fmt.Println(ip)
	return ip
}

func isAllowedIP(ip string) bool {
	for _, v := range viper.GetStringSlice("allowed_ips") {
		if ip == v {
			return true
		}
	}
	return false
}

func Response(w http.ResponseWriter, r *http.Request) {

	ip := getIP(r)
	if !isAllowedIP(ip) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

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
			fmt.Println(err)
		}
		fmt.Printf("Added document to the %s collection. Object ID: %s", typeOfWebhook, res)
	}(typeOfWebhook, body)

}

func WriteToMongo(collectionName string, object []byte) (*mongo.InsertOneResult, error) {
	collection := client.Database(viper.GetString("database.name")).Collection(collectionName)

	var bsonBody interface{}

	if err := bson.UnmarshalExtJSON(object, true, &bsonBody); err !=nil {
		return nil, err
	}

	res, err := collection.InsertOne(context.Background(), bsonBody)
	if err!=nil { 
		return nil, err
	}
	return res, nil
}

func main() {
	initConfig()

	http.HandleFunc("/webhook", Response)
	fmt.Println("Server is listening on port", viper.GetString("server.port"))
	http.ListenAndServe(fmt.Sprintf(":%s", viper.GetString("server.port")), nil)

}