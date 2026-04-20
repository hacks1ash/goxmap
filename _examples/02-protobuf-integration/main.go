// Example 02: Protobuf integration
//
// This example demonstrates mapping between an internal domain model and a
// simulated Protobuf-generated struct using bind tags and getter detection.
//
// The internal struct uses `mapper:"bind:..."` to specify the corresponding
// field names in the external Protobuf struct, since Protobuf field names
// don't follow Go naming conventions.
package main

import (
	"fmt"

	"github.com/hacks1ash/goxmap/_examples/02-protobuf-integration/pb"
)

//go:generate go run github.com/hacks1ash/goxmap -src User -dst _examples/02-protobuf-integration/pb.User -bidi

// User is the internal domain model.
// The bind tags tell the generator which Protobuf field each maps to.
type User struct {
	ID    int64  `mapper:"bind:UserId"`
	Name  string `mapper:"bind:FullName"`
	Email string `mapper:"bind:UserEmail"`
	Age   int32  `mapper:"bind:UserAge"`
}

func main() {
	// Internal -> Proto
	internal := &User{
		ID:    1,
		Name:  "Alice Smith",
		Email: "alice@example.com",
		Age:   30,
	}

	proto := MapUserToPbUser(internal)
	fmt.Printf("Internal -> Proto: %+v\n", proto)

	// Proto -> Internal
	restored := MapPbUserToUser(proto)
	fmt.Printf("Proto -> Internal: %+v\n", restored)

	// Demonstrate nil-safety: pointer function returns nil for nil input
	var empty *pb.User
	safeResult := MapPbUserToUser(empty)
	fmt.Printf("Nil proto -> Internal: %+v\n", safeResult)
}
