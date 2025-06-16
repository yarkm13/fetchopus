package main

import (
	"net/url"
)

var connectorFactories = []ConnectorFactory{
	&FTPConnectorFactory{},
	&SCPConnectorFactory{},
	// add more
}

func getConnectorFactory(u *url.URL) ConnectorFactory {
	for _, factory := range connectorFactories {
		if factory.Accept(u) {
			return factory
		}
	}
	return nil
}
