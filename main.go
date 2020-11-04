package main

import (
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"os"
)

type Config struct {
	ReleaseLabel   string `default:"helm.sh/release"`
	BranchLabel    string `default:"app.fedex.io/git-branch"`
	CommitShaLabel string `default:"app.fedex.io/git-commit"`
	RepoLabel      string `default:"app.fedex.io/git-repository"`
	Dryrun         bool   `default:"true"`
	Debug          bool   `default:"true"`
}

var (
	C         Config
	Clientset *kubernetes.Clientset
)

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
	err := envconfig.Process("cleaner", &C)
	if C.Debug {
		log.Info("running in debug mode")
		log.SetLevel(log.DebugLevel)
	}

	log.Debugf("using config %+v", C)

	Clientset = getKubeCtx()
	handler := http.HandlerFunc(CleanerServer)
	err = http.ListenAndServe(":8000", handler)
	CheckErr(err)
}
