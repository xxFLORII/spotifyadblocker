package main

import (
    "context"
    "fmt"
    "log"
    "net"
    "os"
    "regexp"

    "github.com/DavidGamba/go-getoptions"
    "github.com/things-go/go-socks5"
    "io/ioutil"
    "strings"
)

var upStreamProxy string

func main() {
	var listenPort int
	var listenAddress string
	var blackListFileAddress string
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("h", "?"))
	opt.IntVarOptional(&listenPort, "port", 51081,opt.Alias("p"),opt.Description("Listening Port"))
	opt.StringVarOptional(&listenAddress, "address", "127.0.0.1",opt.Alias("a"),opt.Description("Listening Address"))
	opt.StringVarOptional(&blackListFileAddress, "blacklist", "black_list",opt.Alias("b"),opt.Description("Blacklist File path"))
	opt.StringVarOptional(&upStreamProxy, "upstream", "",opt.Alias("u"),opt.Description("UpStream Socks5 Address (ex: 127.0.0.1:1080)"))
	_, err := opt.Parse(os.Args[1:])
	if opt.Called("help") {
		fmt.Fprintf(os.Stderr, opt.Help())
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n\n", err)
		fmt.Fprintf(os.Stderr, opt.Help(getoptions.HelpSynopsis))
		os.Exit(1)
	}
	contentBytes, err := ioutil.ReadFile(blackListFileAddress)
	if err != nil {
		panic(err)
	}
	contentString := strings.ReplaceAll(string(contentBytes),"\r","")
	blackList := strings.Split(contentString ,"\n")
	var hostRegexes = make([]*regexp.Regexp,len(blackList))

	for i , host := range blackList {
		hostRegexes[i] , _ = regexp.Compile(host)
	}
	resolver := new(CustomResolver)
	resolver.blockList = hostRegexes
	var server *socks5.Server
	if upStreamProxy != "" {
		server = socks5.NewServer(
			socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))),
			socks5.WithResolver(resolver),
			socks5.WithDial(dialOut),
		)
	}else{
		server = socks5.NewServer(
			socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))),
			socks5.WithResolver(resolver),
		)
	}
	if err := server.ListenAndServe("tcp", fmt.Sprintf("%s:%d",listenAddress,listenPort)); err != nil {
		panic(err)
	}
}
func dialOut(ctx context.Context, network, addr string) (net.Conn, error) {
    dialer := &net.Dialer{
        Timeout:   30 * time.Second,
        KeepAlive: 30 * time.Second,
        DualStack: true,
        Resolver: &net.Resolver{
            PreferGo: true,
            Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
                return socks5.Dial(network, upStreamProxy, address)
            },
        },
    }

    conn, err := dialer.DialContext(ctx, network, addr)
    if err != nil {
        return nil, err
    }
    return conn, nil
}
