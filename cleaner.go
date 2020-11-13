package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/go-github/v32/github"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os/exec"
	"reflect"
	"strings"
)

const shell = "sh"

func runCommand(command string) (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(shell, "-c", command)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	return err, stdout.String(), stderr.String()
}

func cleaner(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		_, _ = fmt.Fprintf(w, "400")
	}
	payload, err := github.ValidatePayload(r, []byte(C.Secret))
	if err != nil {
		return err
	}
	defer r.Body.Close()

	hook, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprint(w, http.StatusAccepted)
	var selectors []string

	log.Debugf("got hook of type %s", reflect.TypeOf(hook))

	switch e := hook.(type) {
	case *github.PullRequestEvent:
		log.WithFields(log.Fields{
			"action":   *e.Action,
			"number":   e.Number,
			"reponame": *e.Repo.Name,
			"sha":      *e.PullRequest.Head.SHA,
		}).Debug("received pr")

		if *e.Action == "closed" {
			log.Debug("closed pr")
			selectors = append(selectors,
				fmt.Sprintf("%s=PR-%d,%s=%s,%s=%s", C.BranchLabel, *e.Number, C.RepoLabel, *e.Repo.Name, C.OwnerLabel, *e.PullRequest.Head.Repo.Owner.Login),
				fmt.Sprintf("%s=%s,%s=%s,%s=%s", C.BranchLabel, *e.PullRequest.Head.Ref, C.RepoLabel, *e.Repo.Name, C.OwnerLabel, *e.PullRequest.Head.Repo.Owner.Login),
			)
			log.Debugf("selectors are %s", selectors)
		}
		if *e.Action == "opened" || *e.Action == "reopened" {
			selectors = append(selectors,
				fmt.Sprintf(
					"%s=%s,%s=%s,%s=%s", C.BranchLabel, *e.PullRequest.Head.Ref, C.RepoLabel, *e.Repo.Name, C.OwnerLabel, *e.PullRequest.Head.Repo.Owner.Login,
				),
			)
		}
	case *github.PushEvent:
		branchName := strings.Split(*e.Ref, "/")[2]
		log.WithFields(log.Fields{
			"branch":   branchName,
			"reponame": *e.Repo.Name,
			"deleted":  e.Deleted,
			"created":  e.Created,
		}).Debug("received pushevent")
		if *e.Deleted {
			selectors = append(selectors,
				fmt.Sprintf("%s=%s,%s=%s,%s=%s", C.BranchLabel, branchName, C.RepoLabel, *e.Repo.Name, C.OwnerLabel, *e.Repo.Owner.Name),
			)
		}
	default:
		log.Debug("no action needed")
		return nil
	}

	for _, s := range selectors {
		log.Info("using selector ", s)
		listOptions := metav1.ListOptions{
			LabelSelector: s,
		}
		if err := findAndDelete(listOptions); err != nil {
			log.Info("no deployments found or unable to delete")
		}
	}

	return nil
}

func findAndDelete(listOptions metav1.ListOptions) error {
	var dryrunString string
	if C.Dryrun {
		dryrunString = "--dry-run"
	}
	deployments, err := Clientset.AppsV1().Deployments("").List(context.TODO(), listOptions)
	if err != nil {
		return err
	}
	for _, deployment := range deployments.Items {
		release := deployment.Labels[C.ReleaseLabel]
		if release == "" {
			log.Debugf("release label not set for deployment %s", deployment.Name)
			continue
		}

		log.WithFields(log.Fields{
			"deployment": deployment.Name,
			"namespace":  deployment.Namespace,
		}).Debug("found matching deployment")

		log.Infof("deleting release %s in namespace %s (except when in dryrun mode", release, deployment.Namespace)
		if len(release) > 1 {
			log.WithFields(log.Fields{
				"releases": release,
			}).Panic("more than 1 release matched, aborting")
			panic("bailout")
		}
		deleteCommand := fmt.Sprintf("/bin/helm uninstall -n %s %s %s", deployment.Namespace, release, dryrunString)
		log.WithFields(log.Fields{
			"line to be executed": deleteCommand,
		}).Debug()

		err, out, errout := runCommand(deleteCommand)
		if err != nil {
			log.WithFields(log.Fields{
				"stderr": errout,
			}).Info("could not delete deployments")
		} else {
			log.WithFields(log.Fields{
				"release":            release,
				"namespace":          deployment.Namespace,
				"matched deployment": deployment.Name,
				"repo":               deployment.Labels[C.RepoLabel],
				"branch":             deployment.Labels[C.BranchLabel],
			}).Info("release deleted")
		}
		log.WithFields(log.Fields{
			"stdout": out,
		}).Debug()
	}
	return nil
}

func CleanerServer(w http.ResponseWriter, r *http.Request) {
	err := cleaner(w, r)
	CheckErr(err)
}
