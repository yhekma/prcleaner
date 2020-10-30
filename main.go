package main

import (
	flag "github.com/spf13/pflag"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	// "k8s.io/client-go/rest"
	"net/http"
	"os"
)

var Clientset *kubernetes.Clientset
var Org string
var App string

func CheckErr(err error) {
	if err != nil {
		log.Panic(err.Error())
	}
}

func getKubeCtx() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	CheckErr(err)
	clientset, err := kubernetes.NewForConfig(config)
	CheckErr(err)
	return clientset
}

func logInit() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	logInit()
	org := flag.StringP("org", "o", "", "org to process for")
	app := flag.StringP("app", "a", "", "app to process for")
	debug := flag.BoolP("verbose", "v", false, "turn on verbose")
	flag.Parse()
	Org = *org
	App = *app
	if *debug {
		log.Info("running in verbose")
		log.SetLevel(log.DebugLevel)
	}

	Clientset = getKubeCtx()
	handler := http.HandlerFunc(CleanerServer)
	err := http.ListenAndServe(":8000", handler)
	CheckErr(err)
}