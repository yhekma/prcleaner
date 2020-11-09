package main

import (
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

	log.Infof("got hook of type %s", reflect.TypeOf(hook))

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
			log.Debug("selectors are %v", selectors)
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
			log.Warn("could not select using selectors ", err)
		}
	}

	return nil

}

func findAndDelete(listOptions metav1.ListOptions) error {
	var dryrunString string
	if C.Dryrun {
		dryrunString = "--dry-run"
	}
	pods, err := Clientset.CoreV1().Pods("").List(context.TODO(), listOptions)
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		release := pod.Labels[C.ReleaseLabel]
		if release == "" {
			log.Debugf("release label not set for pod %s", pod.Name)
			continue
		}

		log.WithFields(log.Fields{
			"pod":       pod.Name,
			"namespace": pod.Namespace,
		}).Debug("found matching pod")

		log.Infof("deleting release %s (except when in dryrun mode", release)

		var out []byte
		out, err = exec.Command("/bin/helm", "uninstall", "-n", pod.Namespace, release, dryrunString).Output()
		if err != nil {
			return err
		}
		log.WithFields(log.Fields{
			"helm command": fmt.Sprintf("/bin/helm uninstall -n %s %s %s", pod.Namespace, release, dryrunString),
			"output":       string(out),
		}).Debug()
	}
	return nil
}

func CleanerServer(w http.ResponseWriter, r *http.Request) {
	err := cleaner(w, r)
	CheckErr(err)
}
