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
	ReleaseLabel   string
	BranchLabel    string
	CommitShaLabel string
	RepoLabel      string
	Dryrun         bool
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
	debug := flag.BoolP("verbose", "v", false, "turn on verbose. Env: DEBUG")
	dryrun := flag.BoolP("dry-run", "d", true, "don't actually do anything. Env: DRYRUN")
	releaseLabel := flag.String("releaseLabel", "helm.sh/release", "label name for releases. Env: RELEASE_LABEL")
	branchLabel := flag.String("branchLabel", "app.fedex.io/git-branch", "label name for branches. Env: BRANCH_LABEL")
	repoLabel := flag.String("repoLabel", "app.fedex.io/git-repository", "label name for repo. Env: REPO_LABEL")
	commitShaLabel := flag.String("commitSha", "app.fedex.io/git-commit", "label name for commit sha. Env: COMMIT_LABEL")
	flag.Parse()

	// TODO this needs to be moved to cobra/viper
	if _, set := os.LookupEnv("DEBUG"); set || *debug {
		log.Info("running in debug")
		log.SetLevel(log.DebugLevel)
	}

	if *debug {
		log.Info("running in verbose")
		log.SetLevel(log.DebugLevel)
	}

	if _, set := os.LookupEnv("DRYRUN"); set {
		log.Debug("set dryrun to true from env")
		Dryrun = true
	} else {
		log.Debug("set dryrun to ", *dryrun, " from flag/default", *dryrun)
		Dryrun = *dryrun
	}

	if r, set := os.LookupEnv("RELEASE_LABEL"); set {
		log.Debug("set releaselabel to", r, " from env")
		ReleaseLabel = r
	} else {
		log.Debug("set releaselabel to ", *releaseLabel, " from flag/default")
		ReleaseLabel = *releaseLabel
	}

	if b, set := os.LookupEnv("BRANCH_LABEL"); set {
		log.Debug("set branchlabel to ", b, " from env")
		BranchLabel = b
	} else {
		log.Debug("set branchlabel to ", *branchLabel, " from flag/default")
		BranchLabel = *branchLabel
	}

	if r, set := os.LookupEnv("REPO_LABEL"); set {
		log.Debug("set repolabel to ", r, " from env")
		RepoLabel = r
	} else {
		log.Debug("set repolabel to ", *repoLabel, " from flag/default")
		RepoLabel = *repoLabel
	}

	if c, set := os.LookupEnv("COMMIT_LABEL"); set {
		log.Debug("set commitlabel to ", c, " from env")
		CommitShaLabel = c
	} else {
		log.Debug("set commitlabel to ", *commitShaLabel, " from flag/default")
		CommitShaLabel = *commitShaLabel
	}

	Clientset = getKubeCtx()
	handler := http.HandlerFunc(CleanerServer)
	err := http.ListenAndServe(":8000", handler)
	CheckErr(err)
}
