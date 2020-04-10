/*
* Handles the connection initialization and management for connected
* ships. This module handles all of its own connection logic since the
* shipgate protocol differs from the way game clients are processed.
 */
package shipgate

type ship struct {
	name string
	ip   string
	port string
	id   int
}
