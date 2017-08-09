This is a Go package for interfacing with a [UniFi](https://unifi-sdn.ubnt.com/) controller.

[![GoDoc](https://godoc.org/github.com/dsymonds/unifi?status.svg)](https://godoc.org/github.com/dsymonds/unifi)

This is at a very early stage.

To get it,

	go get github.com/dsymonds/unifi

You will need to put auth information in $HOME/.unifi-auth that looks like

	{"Username":"xxx","Password":"yyy","ControllerHost":"unifi"}

Don't forget to `chmod 600 $HOME/.unifi-auth`.
I plan to make it easier to pass to this package programmatically.

To do a quick test that will print out the clients on the
default site,

	go run demo/list-sta.go

The UniFi API is not documented, so this is reverse engineered from a few sources:

   * https://dl.ubnt.com/unifi/5.5.20/unifi_sh_api
   * https://github.com/malle-pietje/UniFi-API-browser/blob/master/phpapi/class.unifi.php
