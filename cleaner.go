package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
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
	var selector []string

	log.Infof("got hook of type %s", reflect.TypeOf(hook))

	switch e := hook.(type) {
	case *github.PullRequestEvent:
		log.WithFields(log.Fields{
			"action":   *e.Action,
			"number":   e.Number,
			"reponame": *e.Repo.Name,
			"sha":      *e.PullRequest.Head.SHA,
		}).Debug("received pr")

		if *e.PullRequest.State == "closed" {
			selector = append(selector,
				fmt.Sprintf("%s=PR-%d,%s=%s,%s=%s", C.BranchLabel, *e.Number, C.RepoLabel, *e.Repo.Name, C.OwnerLabel, *e.PullRequest.Head.Repo.Owner),
				fmt.Sprintf("%s=%s,%s=%s,%s=%v", C.BranchLabel, *e.PullRequest.Head.Ref, C.RepoLabel, *e.Repo.Name, C.OwnerLabel, *e.PullRequest.Head.Repo.Owner),
			)
		}
		if *e.PullRequest.State == "opened" || *e.PullRequest.State == "reopened" {
			selector = append(selector,
				fmt.Sprintf(
					"%s=%s,%s=%s,%s=%s", C.BranchLabel, *e.PullRequest.Head.Ref, C.RepoLabel, *e.Repo.Name, C.OwnerLabel, *e.PullRequest.Head.Repo.Owner,
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
			selector = append(selector,
				fmt.Sprintf("%s=%s,%s=%s,%s=%s", C.BranchLabel, branchName, C.RepoLabel, *e.Repo.Name, C.OwnerLabel, *e.Repo.Owner.Name),
			)
		}
	default:
		log.Debug("no action needed")
		return nil
	}

	for _, s := range selector {
		log.Info("using selector ", selector)
		listOptions := metav1.ListOptions{
			LabelSelector: s,
		}
		if err := findAndDelete(listOptions); err != nil {
			log.Warn("could not select using selector ", err.Error())
		}
	}

	return nil

}

func findAndDelete(listOptions metav1.ListOptions) error {
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
		if C.Dryrun {
			out, err = exec.Command("/bin/helm", "uninstall", "-n", pod.Namespace, release, "--dry-run").Output()
		} else {
			out, err = exec.Command("/bin/helm", "uninstall", "-n", pod.Namespace, release).Output()
		}
		if err != nil {
			return err
		}
		log.WithFields(log.Fields{
			"helm command": fmt.Sprintf("/bin/helm uninstall -n %s %s --dry-run", pod.Namespace, release),
			"output":       string(out),
		}).Debug()
	}
	return nil
}

func CleanerServer(w http.ResponseWriter, r *http.Request) {
	err := cleaner(w, r)
	CheckErr(err)
}
