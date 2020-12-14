package main

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"os"
)

type Config struct {
	ReleaseLabel string `default:"helm.sh/release"`
	BranchLabel  string `default:"app.fedex.io/git-branch"`
	OwnerLabel   string `default:"app.fedex.io/git-owner"`
	RepoLabel    string `default:"app.fedex.io/git-repository"`
	Secret       string `required:"true"`
	Delay        int    `default:"300"`
	Dryrun       bool   `default:"true"`
	Debug        bool   `default:"true"`
}

var (
	C         Config
	Clientset *kubernetes.Clientset
)

func getOwnHash() (hash string) {
	hasher := sha256.New()
	s, err := ioutil.ReadFile(os.Args[0])
	CheckErr(err)
	hasher.Write(s)
	return hex.EncodeToString(hasher.Sum(nil))
}

func CheckErr(err error) {
	if err != nil {
		log.Warn(err.Error())
	}
}

func getKubeCtx() *kubernetes.Clientset {
	log.Debug("get clusterconfig")
	config, err := rest.InClusterConfig()
	CheckErr(err)
	log.Debug("got clusterconfig")
	log.Debug("get clientset")
	clientset, err := kubernetes.NewForConfig(config)
	CheckErr(err)
	log.Debug("got clientset")
	return clientset
}

func logInit() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	logInit()
	log.Info("process environment")
	err := envconfig.Process("cleaner", &C)
	log.Info("processed environment")
	if C.Debug {
		log.Info("running in debug mode")
		log.SetLevel(log.DebugLevel)
	}

	log.WithFields(log.Fields{
		"releaselabel": C.ReleaseLabel,
		"branchlabel":  C.BranchLabel,
		"ownerlabel":   C.OwnerLabel,
		"repolabel":    C.RepoLabel,
		"dryrun":       C.Dryrun,
		"debug":        C.Debug,
		"hash":         getOwnHash(),
		"secret":       "<redacted>",
	}).Info("running config")

	Clientset = getKubeCtx()
	handler := http.HandlerFunc(CleanerServer)
	log.Info("starting server")
	err = http.ListenAndServe(":8000", handler)
	CheckErr(err)
}
