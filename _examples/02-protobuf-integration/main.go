// Example 02: Protobuf integration
//
// This example demonstrates mapping between an internal domain model and a
// simulated Protobuf-generated struct using bind tags and getter detection.
//
// The internal struct uses `mapper:"bind:..."` to specify the corresponding
// field names in the external Protobuf struct, since Protobuf field names
// don't follow Go naming conventions.
//
// In a real project, you would run:
//
//	go run github.com/hacks1ash/goxmap \
//	    -src User -dst User \
//	    -external-pkg github.com/yourorg/yourrepo/proto \
//	    -bidi
//
// This example shows the generated output inline for clarity.
package main

import (
	"fmt"

	"github.com/hacks1ash/goxmap/_examples/02-protobuf-integration/pb"
)

// User is the internal domain model.
// The bind tags tell the generator which Protobuf field each maps to.
type User struct {
	ID    int64  `mapper:"bind:UserId"`
	Name  string `mapper:"bind:FullName"`
	Email string `mapper:"bind:UserEmail"`
	Age   int32  `mapper:"bind:UserAge"`
}

// MapUserToProtoUser maps the internal User to the Protobuf User.
// Generated code uses pointer-based signatures for type-safe nil handling.
func MapUserToProtoUser(src *User) *pb.User {
	if src == nil {
		return nil
	}
	dst := &pb.User{}
	dst.UserId = src.ID
	dst.FullName = src.Name
	dst.UserEmail = src.Email
	dst.UserAge = src.Age
	return dst
}

// MapProtoUserToUser maps the Protobuf User back to the internal User.
// Generated code uses pointer-based signatures and reads via getter methods for nil safety.
func MapProtoUserToUser(src *pb.User) *User {
	if src == nil {
		return nil
	}
	dst := &User{}
	dst.ID = src.GetUserId()
	dst.Name = src.GetFullName()
	dst.Email = src.GetUserEmail()
	dst.Age = src.GetUserAge()
	return dst
}

func main() {
	// Internal -> Proto
	internal := &User{
		ID:    1,
		Name:  "Alice Smith",
		Email: "alice@example.com",
		Age:   30,
	}

	proto := MapUserToProtoUser(internal)
	fmt.Printf("Internal -> Proto: %+v\n", proto)

	// Proto -> Internal
	restored := MapProtoUserToUser(proto)
	fmt.Printf("Proto -> Internal: %+v\n", restored)

	// Demonstrate nil-safety: pointer function returns nil for nil input
	var empty *pb.User
	safeResult := MapProtoUserToUser(empty)
	fmt.Printf("Nil proto -> Internal: %+v\n", safeResult)
}
