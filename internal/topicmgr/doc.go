// Package topicmgr provides a compile-time safe, strongly-typed topic management system
// that eliminates magic strings and provides centralized topic definitions with
// framework/module scoping.
//
// The package replaces the existing TopicRegistry system with a more robust approach
// that enforces topic safety at compile time while maintaining backward compatibility
// during migration.
//
// Key Features:
//   - Compile-time safety through strongly-typed topic definitions
//   - Framework/module scoping for better organization
//   - Centralized registry with metadata and discovery
//   - Validation and error handling
//   - Backward compatibility with existing systems
//
// Usage:
//
// Framework topics are defined by core services:
//
//	var UserOnline = topicmgr.DefineFramework(topicmgr.TopicConfig{
//		Name:        "presence.user.online",
//		Description: "Published when a user comes online",
//		Pattern:     "presence.user.online",
//		Example:     `{"userID":"user123","timestamp":"2024-01-01T00:00:00Z"}`,
//	})
//
// Module topics are defined by application modules:
//
//	var NewMessage = topicmgr.DefineModule(topicmgr.TopicConfig{
//		Name:        "client.chat.message.new",
//		Module:      "chat",
//		Description: "A new chat message sent by a client",
//		Pattern:     "client.chat.message.new",
//		Example:     `{"action":"client.chat.message.new","payload":{"content":"Hello!"}}`,
//	})
//
// Topics are registered with the manager:
//
//	manager := topicmgr.Default()
//	err := manager.Register(UserOnline)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Topics can be discovered and listed:
//
//	allTopics := manager.List()
//	chatTopics := manager.ListByModule("chat")
//	frameworkTopics := manager.ListFrameworkTopics()
package topicmgr