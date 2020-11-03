package main

import (
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"os"
)

var (
	ReleaseLabel   *string
	BranchLabel    *string
	CommitShaLabel *string
	RepoLabel      *string
	Dryrun         *bool
	Clientset      *kubernetes.Clientset
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
	debug := flag.BoolP("verbose", "v", false, "turn on verbose")
	Dryrun = flag.BoolP("dry-run", "d", true, "don't actually do anything")
	ReleaseLabel = flag.String("releaseLabel", "helm.sh/release", "label name for releases")
	BranchLabel = flag.String("branchLabel", "app.fedex.io/git-branch", "label name for branches")
	RepoLabel = flag.String("repoLabel", "app.fedex.io/git-repository", "label name for repo")
	CommitShaLabel = flag.String("commitSha", "app.fedex.io/git-commit", "label name for commit sha")
	flag.Parse()

	if *debug {
		log.Info("running in verbose")
		log.SetLevel(log.DebugLevel)
	}

	Clientset = getKubeCtx()
	handler := http.HandlerFunc(CleanerServer)
	err := http.ListenAndServe(":8000", handler)
	CheckErr(err)
}
