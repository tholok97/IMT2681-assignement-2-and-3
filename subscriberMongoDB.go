package main

import (
	"fmt"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type SubscriberMongoDB struct {
	URL                      string
	Name                     string
	SubscriberCollectionName string
}

// SubscriberMongoDB returns a fresh SubscriberMongoDB
func SubscriberMongoDBFactory(url, name, collectionName string) (SubscriberMongoDB, error) {

	db := SubscriberMongoDB{URL: url, Name: name, SubscriberCollectionName: collectionName}

	session, err := mgo.Dial(db.URL)
	if err != nil {
		return SubscriberMongoDB{}, nil
	}
	defer session.Close()

	return db, nil
}

// Add adds a subscriber to the db
func (db *SubscriberMongoDB) Add(s Subscriber) (string, error) {
	session, err := mgo.Dial(db.URL)
	if err != nil {
		return "", err
	}
	defer session.Close()

	s.ID = bson.NewObjectId()
	err = session.DB(db.Name).C(db.SubscriberCollectionName).Insert(s)
	if err != nil {
		return "", err
	}

	fmt.Println("id: ", s.ID.Hex())

	return s.ID.Hex(), nil
}

// Remove subscriber with id. Err if not found
func (db *SubscriberMongoDB) Remove(id string) error {
	session, err := mgo.Dial(db.URL)
	if err != nil {
		return err
	}
	defer session.Close()

	err = session.DB(db.Name).C(db.SubscriberCollectionName).RemoveId(id)
	if err != nil {
		return err
	}

	return nil
}

// Count returns the number of subscribers in the db
func (db *SubscriberMongoDB) Count() (int, error) {
	session, err := mgo.Dial(db.URL)
	if err != nil {
		return 923, err
	}
	defer session.Close()

	count, err := session.DB(db.Name).C(db.SubscriberCollectionName).Count()
	if err != nil {
		return 923, err
	}

	return count, nil
}

// Get gets subscriber with id
func (db *SubscriberMongoDB) Get(id string) (Subscriber, error) {

	session, err := mgo.Dial(db.URL)
	if err != nil {
		return Subscriber{}, err
	}
	defer session.Close()

	var sub Subscriber
	err = session.DB(db.Name).C(db.SubscriberCollectionName).FindId(bson.ObjectIdHex(id)).One(&sub)
	if err != nil {
		return Subscriber{}, err
	}
	return sub, nil
}

// GetAll gets all subscribers as slice
func (db *SubscriberMongoDB) GetAll() ([]Subscriber, error) {
	return nil, nil
}
