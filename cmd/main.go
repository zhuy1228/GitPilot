package main

import "github.com/zhuy1228/GitPilot/internal"

func main() {
	proxyInfo, err := internal.GetCurrentProxy()
	if err != nil {
		panic(err)
	}
	println(proxyInfo.String())
}
