package main

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	C               Config
	ReleasesDeleted *prometheus.CounterVec
	Clientset       *kubernetes.Clientset
)

func getOwnHash() (hash string) {
	hasher := sha256.New()
	s, err := ioutil.ReadFile(os.Args[0])
	CheckErr(err, "error getting own hash")
	hasher.Write(s)
	return hex.EncodeToString(hasher.Sum(nil))
}

func CheckErr(err error, msg string) {
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Warn(msg)
	}
}

func getKubeCtx() *kubernetes.Clientset {
	log.Debug("get clusterconfig")
	config, err := rest.InClusterConfig()
	CheckErr(err, "error getting cluster config")
	log.Debug("got clusterconfig")
	log.Debug("get clientset")
	clientset, err := kubernetes.NewForConfig(config)
	CheckErr(err, "error getting clientset")
	log.Debug("got clientset")
	return clientset
}

func logInit() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func aliveAndReady(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {
	logInit()
	log.Debug("initializing router")
	router := mux.NewRouter()
	log.Info("process environment")
	err := envconfig.Process("cleaner", &C)
	CheckErr(err, "could not process environment")
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
		"delay":        C.Delay,
		"secret":       "<redacted>",
	}).Info("running config")

	ReleasesDeleted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "releases_deleted",
	}, []string{"namespaces"})
	CheckErr(prometheus.Register(ReleasesDeleted), "could not register prometheus counter")

	Clientset = getKubeCtx()
	handler := http.HandlerFunc(CleanerServer)
	readyHandler := http.HandlerFunc(aliveAndReady)
	router.HandleFunc("/", handler)
	router.HandleFunc("/ready", readyHandler)
	router.HandleFunc("/alive", readyHandler)
	router.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{
		Handler: router,
		Addr:    ":8000",
	}
	log.Fatal(srv.ListenAndServe())
}
