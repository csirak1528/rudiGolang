package firebase

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

// Use the application default credentials
func InitDb() *firestore.Client {
	// Use a service account
	ctx := context.Background()
	sa := option.WithCredentialsFile("rudiweb-dc929-firebase-adminsdk-vyddx-c890c853eb.json")
	conf := &firebase.Config{ProjectID: "rudiweb-dc929"}
	app, err := firebase.NewApp(ctx, conf, sa)
	if err != nil {
		log.Fatalln(err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	return client
}
