package utils

import (
	"context"
	"fmt"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// Initializes Firebase App
func InitFirebase() (*firebase.App, error) {
	opt := option.WithCredentialsJSON([]byte(os.Getenv("FIREBASE_SERVICE_ACCOUNT")))
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing Firebase app: %v", err)
	}
	return app, nil
}

// Verifies Firebase ID token
func VerifyIDToken(ctx context.Context, app *firebase.App, idToken string) (*auth.Token, error) {
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting Auth client: %v", err)
	}
	token, err := client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("error verifying ID token: %v", err)
	}
	return token, nil
}

// Get Firebase Auth Client (used in Login/Register)
func GetFirebaseAuthClient(ctx context.Context) (*auth.Client, error) {
	app, err := InitFirebase()
	if err != nil {
		return nil, err
	}
	return app.Auth(ctx)
}
