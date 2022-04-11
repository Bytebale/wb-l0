package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/jackc/pgx/v4"
	"github.com/nats-io/stan.go"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	cache := make(map[string]Data)

	var (
		name     = "postgres"
		password = "postgres"
	)

	dbURL := fmt.Sprintf("postgres://%s:%s@localhost:5432/postgres?sslmode=disable", name, password)
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Println(err)
	} else {
		log.Println("Connected to DB!")
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), "select data_json from orders")
	if err != nil {
		log.Println(err)
	}

	for rows.Next() {
		var (
			bytes []byte
			data  Data
		)

		err = rows.Scan(&bytes)
		if err != nil {
			log.Println(err)
		}

		err = json.Unmarshal(bytes, &data)
		if err != nil {
			log.Println(err)
		}
		cache[data.OrderUid] = data
		log.Printf("Got data from DB: %s\n", data.OrderUid)
	}
	rows.Close()

	sc, err := stan.Connect("test-cluster", "Sub", stan.NatsURL("nats://localhost:4222"))
	if err != nil && err != io.EOF {
		log.Fatalln(err)
	} else {
		log.Println("Connecting to Nats-streaming!")
	}
	defer sc.Close()

	_, err = sc.Subscribe("foo1", func(m *stan.Msg) {
		var d Data

		err := json.Unmarshal(m.Data, &d)
		if err != nil {
			log.Println(err)
		} else {
			log.Println(" Got data from NATS-streaming!")
		}

		if _, ok := cache[d.OrderUid]; !ok {
			cache[d.OrderUid] = d

			_, err = conn.Exec(context.Background(), "insert into orders values ($1, $2)", d.OrderUid, m.Data)
			if err != nil {
				log.Println(err)
			} else {
				log.Printf("data has been set to DB: %s\n", d.OrderUid)
			}
		}
	}, stan.StartWithLastReceived())
	if err != nil {
		log.Println(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			tmpl, err := template.ParseFiles("interface.html")
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}

			err = tmpl.Execute(w, nil)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}

		case "POST":
			if val, ok := cache[req.PostFormValue("order_uid")]; ok {

				b, err := json.MarshalIndent(val, "", "\t")
				if err != nil {
					log.Println(err)
				}
				log.Printf("Data send: %s\n", req.PostFormValue("order_uid"))
				fmt.Fprint(w, string(b))
			} else {
				log.Println("encorrect UID")
				fmt.Fprint(w, "Data not found")
			}
		}
	})

	log.Fatal(http.ListenAndServe(":8082", nil))

}
