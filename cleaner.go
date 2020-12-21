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
	"regexp"
	"strings"
	"time"
)

const shell = "sh"
const nomatch = "nomatch=true"

func runCommand(command string) (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(shell, "-c", command)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	return err, stdout.String(), stderr.String()
}

func trimString(s string, l int) string {
	if len(s) > l {
		return s[:l]
	}
	return s
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
	// If for any reason we fall through the select below, we don't want to have an empty selector
	selector := nomatch

	re := regexp.MustCompile(`[\W_]`)

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
			selector = fmt.Sprintf("%s=PR-%d,%s=%s,%s=%s", C.BranchLabel, *e.Number, C.RepoLabel, *e.Repo.Name, C.OwnerLabel, *e.PullRequest.Head.Repo.Owner.Login)
		}
		if *e.Action == "opened" || *e.Action == "reopened" {
			saneRef := trimString(re.ReplaceAllString(*e.PullRequest.Head.Ref, "-"), 62)
			selector = fmt.Sprintf("%s=%s,%s=%s,%s=%s", C.BranchLabel, saneRef, C.RepoLabel, *e.Repo.Name, C.OwnerLabel, *e.PullRequest.Head.Repo.Owner.Login)
		}
	case *github.PushEvent:
		branchName := strings.ReplaceAll(*e.Ref, "refs/heads/", "")
		saneBranchName := trimString(re.ReplaceAllString(branchName, "-"), 62)

		log.WithFields(log.Fields{
			"branch":                branchName,
			"sanitized branch name": saneBranchName,
			"reponame":              *e.Repo.Name,
			"deleted":               e.Deleted,
			"created":               e.Created,
		}).Debug("received pushevent")
		if *e.Deleted {
			selector = fmt.Sprintf("%s=%s,%s=%s,%s=%s", C.BranchLabel, saneBranchName, C.RepoLabel, *e.Repo.Name, C.OwnerLabel, *e.Repo.Owner.Name)
		}
	}

	if selector == nomatch {
		log.Debug("no action needed")
		return nil
	}

	log.WithFields(log.Fields{
		"selector": selector,
	}).Debug("using selector")

	listOptions := metav1.ListOptions{
		LabelSelector: selector,
	}

	err = findAndDelete(listOptions)
	CheckErr(err, "could not delete releases")

	// Rerun the same after 5 minutes to try and catch race conditions
	go func(l metav1.ListOptions) {
		log.WithFields(log.Fields{
			"selector": selector,
		}).Debugf("scheduling to run after %d seconds", C.Delay)
		time.Sleep(time.Duration(C.Delay) * time.Second)
		log.WithFields(log.Fields{
			"selector": selector,
		}).Debugf("rerunning after %d seconds", C.Delay)
		err = findAndDelete(l)
		CheckErr(err, "could not delete releases")
	}(listOptions)

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
		// If we match more than 1 release, something is very wrong
		if len(strings.Split(release, " ")) > 1 {
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
	CheckErr(err, "could not start cleaner")
}
